package hyperliquid

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// IWebSocketAPI is the interface for WebSocket operations
type IWebSocketAPI interface {
	Connect() error
	Disconnect() error
	DisconnectForTesting() error // For testing reconnection without manual disconnect flag
	IsConnected() bool
	SetDebug(status bool)
	GetLatency() int64

	// Subscription methods
	SubscribeOrderbook(coin string, handler SubscriptionHandler) error
	SubscribeTrades(coin string, handler SubscriptionHandler) error
	SubscribeUserFills(user string, handler SubscriptionHandler) error
	SubscribeAllMids(handler SubscriptionHandler) error
	SubscribeUserEvents(user string, handler SubscriptionHandler) error
	SubscribeUserFundings(user string, handler SubscriptionHandler) error
	SubscribeUserNonFundingLedgerUpdates(user string, handler SubscriptionHandler) error
	SubscribeUserTwapSliceFills(user string, handler SubscriptionHandler) error
	SubscribeUserTwapHistory(user string, handler SubscriptionHandler) error
	SubscribeActiveAssetCtx(coin string, handler SubscriptionHandler) error
	SubscribeActiveAssetData(user string, coin string, handler SubscriptionHandler) error
	SubscribeBbo(coin string, handler SubscriptionHandler) error
	SubscribeCandle(coin string, interval string, handler SubscriptionHandler) error
	SubscribeOrderUpdates(user string, handler SubscriptionHandler) error
	SubscribeNotification(user string, handler SubscriptionHandler) error
	SubscribeWebData2(user string, handler SubscriptionHandler) error

	// Unsubscribe methods
	UnsubscribeOrderbook(coin string) error
	UnsubscribeTrades(coin string) error
	UnsubscribeUserFills(user string) error
	UnsubscribeAllMids() error
	UnsubscribeUserEvents(user string) error
	UnsubscribeUserFundings(user string) error
	UnsubscribeUserNonFundingLedgerUpdates(user string) error
	UnsubscribeUserTwapSliceFills(user string) error
	UnsubscribeUserTwapHistory(user string) error
	UnsubscribeActiveAssetCtx(coin string) error
	UnsubscribeActiveAssetData(user string, coin string) error
	UnsubscribeBbo(coin string) error
	UnsubscribeCandle(coin string, interval string) error
	UnsubscribeOrderUpdates(user string) error
	UnsubscribeNotification(user string) error
	UnsubscribeWebData2(user string) error

	// Post request methods
	PostRequest(requestType string, payload interface{}) (*WSPostResponseData, error)
	PostInfoRequest(payload interface{}) (*WSPostResponseData, error)
	PostActionRequest(payload interface{}) (*WSPostResponseData, error)
	PostOrderRequest(action interface{}, nonce uint64, signature RsvSignature, vaultAddress *string) (*WSPostResponseData, error)
}

// SubscriptionType represents the type of subscription
type SubscriptionType int

const (
	SubTypeUserFills SubscriptionType = iota
	SubTypeL2Book
	SubTypeTrades
	SubTypeOrderUpdates
	SubTypeUserEvents
	SubTypeUserFundings
	SubTypeUserNonFundingLedgerUpdates
	SubTypeUserTwapSliceFills
	SubTypeUserTwapHistory
	SubTypeActiveAssetCtx
	SubTypeActiveAssetData
	SubTypeBbo
	SubTypeCandle
	SubTypeNotification
	SubTypeWebData2
	SubTypeAllMids
)

// Subscription represents a subscription with optimized matching
type Subscription struct {
	Type     SubscriptionType
	Params   map[string]string // Pre-split parameters
	Handler  SubscriptionHandler
	Channel  chan interface{}
	User     string // For user-specific subscriptions
	Coin     string // For coin-specific subscriptions
	Interval string // For candle subscriptions
}

type WebSocketAPI struct {
	conn             *websocket.Conn
	url              string
	subscriptions    map[string]*Subscription   // Optimized subscription storage
	channelHandlers  map[string][]*Subscription // Fast lookup by channel
	postResponses    map[int]chan WSPostResponseData
	mu               sync.RWMutex
	reconnectCount   atomic.Int32
	maxReconnects    int32
	isConnected      atomic.Bool
	debug            atomic.Bool
	manualDisconnect atomic.Bool  // Flag to prevent auto-reconnect on manual disconnect
	latencyMs        atomic.Int64 // Current latency in milliseconds
	lastPingTime     atomic.Value // Timestamp of the last ping sent (time.Time)
	nextPostID       atomic.Int64 // Next post request ID

	// Pre-marshaled messages for efficiency
	pingMessageBytes []byte

	// Performance optimizations
	messageBufferPool sync.Pool // Pool for message buffers
	responsePool      sync.Pool // Pool for WSResponse objects

	// Goroutine management
	pingStopChan chan struct{} // Channel to stop ping handler
}

// WSSubscription represents a WebSocket subscription request
type WSSubscription struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

// WSResponse represents a WebSocket response
type WSResponse struct {
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}

// SubscriptionHandler is a function type for handling subscription data
type SubscriptionHandler func(data interface{})

// WSPostRequest represents a WebSocket post request
type WSPostRequest struct {
	Method  string     `json:"method"`
	ID      int        `json:"id"`
	Request WSPostData `json:"request"`
}

// WSPostData represents the data part of a post request
type WSPostData struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSPostResponse represents a WebSocket post response
type WSPostResponse struct {
	Channel string             `json:"channel"`
	Data    WSPostResponseData `json:"data"`
}

// WSPostResponseData represents the data part of a post response
type WSPostResponseData struct {
	ID       int                `json:"id"`
	Response WSPostResponseBody `json:"response"`
}

// WSPostResponseBody represents the response body of a post response
type WSPostResponseBody struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// PostResponseHandler is a function type for handling post response data
type PostResponseHandler func(response WSPostResponseData)

func NewWebSocketAPI(isMainnet bool) *WebSocketAPI {
	wsURL := MainnetWSURL
	if !isMainnet {
		wsURL = TestnetWSURL
	}

	// Pre-marshal ping message for efficiency using fast JSON
	pingMsg := map[string]interface{}{
		"method": "ping",
	}
	pingBytes, _ := FastMarshal(pingMsg)

	client := &WebSocketAPI{
		url:              wsURL,
		subscriptions:    make(map[string]*Subscription),
		channelHandlers:  make(map[string][]*Subscription),
		postResponses:    make(map[int]chan WSPostResponseData),
		maxReconnects:    5,
		nextPostID:       atomic.Int64{},
		pingMessageBytes: pingBytes,
		messageBufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 4096) // Pre-allocate 4KB buffer
			},
		},
		responsePool: sync.Pool{
			New: func() interface{} {
				return &WSResponse{}
			},
		},
	}
	client.nextPostID.Store(1)
	client.isConnected.Store(false)
	client.debug.Store(false)
	client.manualDisconnect.Store(false)
	client.reconnectCount.Store(0)
	return client
}

// Connect establishes a WebSocket connection
func (ws *WebSocketAPI) Connect() error {
	u, err := url.Parse(ws.url)
	if err != nil {
		return fmt.Errorf("failed to parse WebSocket URL: %w", err)
	}

	if ws.debug.Load() {
		log.Printf("Connecting to WebSocket URL: %s", u.String())
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	// Stop any existing ping handler before starting a new one
	if ws.pingStopChan != nil {
		close(ws.pingStopChan)
	}

	ws.conn = conn
	ws.isConnected.Store(true)
	ws.manualDisconnect.Store(false) // Reset manual disconnect flag
	ws.reconnectCount.Store(0)
	ws.pingStopChan = make(chan struct{}) // Create new stop channel

	if ws.debug.Load() {
		log.Println("WebSocket connection established")
	}

	go ws.readMessages()
	go ws.pingHandler()

	return nil
}

// Disconnect closes the WebSocket connection and prevents auto-reconnect
func (ws *WebSocketAPI) Disconnect() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.isConnected.Store(false)
	ws.manualDisconnect.Store(true) // Prevent auto-reconnect
	ws.reconnectCount.Store(0)      // Reset reconnect count on manual disconnect

	// Stop ping handler
	if ws.pingStopChan != nil {
		close(ws.pingStopChan)
		ws.pingStopChan = nil
	}

	// Clean up all subscriptions and their goroutines
	for _, sub := range ws.subscriptions {
		// Close the subscription channel to terminate the handler goroutine
		close(sub.Channel)
	}

	// Clear subscription maps
	ws.subscriptions = make(map[string]*Subscription)
	ws.channelHandlers = make(map[string][]*Subscription)

	// Clean up post response channels
	for id, responseChan := range ws.postResponses {
		close(responseChan)
		delete(ws.postResponses, id)
	}

	if ws.conn != nil {
		return ws.conn.Close()
	}
	return nil
}

// DisconnectForTesting closes the connection without setting manualDisconnect flag
// This allows the automatic reconnection logic to work for testing purposes
func (ws *WebSocketAPI) DisconnectForTesting() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.isConnected.Store(false)
	// Don't set manualDisconnect = true, so auto-reconnect will work
	// Don't reset reconnectCount, so it continues from where it left off

	// Stop ping handler
	if ws.pingStopChan != nil {
		close(ws.pingStopChan)
		ws.pingStopChan = nil
	}

	if ws.conn != nil {
		return ws.conn.Close()
	}
	return nil
}

// IsConnected returns the connection status
func (ws *WebSocketAPI) IsConnected() bool {
	return ws.isConnected.Load()
}

// SetDebugActive enables debug mode
func (ws *WebSocketAPI) SetDebug(status bool) {
	ws.debug.Store(status)
}

// readMessages reads messages from the WebSocket connection with optimized processing
func (ws *WebSocketAPI) readMessages() {
	defer func() {
		ws.isConnected.Store(false)
	}()

	for {
		// Get message buffer from pool
		messageBuffer := ws.messageBufferPool.Get().([]byte)
		messageBuffer = messageBuffer[:0] // Reset buffer

		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			// Return buffer to pool on error
			ws.messageBufferPool.Put(messageBuffer)

			if ws.debug.Load() {
				log.Printf("WebSocket read error: %v", err)
			}
			// Don't reconnect if manually disconnected
			manualDisconnect := ws.manualDisconnect.Load()

			if !manualDisconnect && ws.reconnectCount.Load() < ws.maxReconnects {
				ws.reconnect()
			}
			return
		}

		if ws.debug.Load() {
			log.Printf("Received WebSocket message: %s", string(message))
		}

		// Fast path: Handle special messages without full JSON parsing
		if len(message) > 0 {
			switch message[0] {
			case '{':
				ws.processJSONMessage(message)
			default:
				if ws.debug.Load() {
					log.Printf("Unknown message format: %s", string(message))
				}
			}
		}

		// Return buffer to pool after processing
		ws.messageBufferPool.Put(messageBuffer)
	}
}

// processJSONMessage efficiently processes JSON messages
func (ws *WebSocketAPI) processJSONMessage(message []byte) {
	// Get response object from pool
	response := ws.responsePool.Get().(*WSResponse)
	defer ws.responsePool.Put(response)

	// Reset response object
	*response = WSResponse{}

	if err := PooledUnmarshal(message, response); err != nil {
		if ws.debug.Load() {
			log.Printf("Failed to unmarshal WebSocket message: %v", err)
		}
		return
	}

	// Fast path for special messages
	switch response.Channel {
	case "subscribed", "subscription", "subscriptionResponse":
		if ws.debug.Load() {
			log.Printf("Received subscription acknowledgment: %+v", response.Data)
		}
		return

	case "pong":
		ws.handlePong()
		return

	case "post":
		ws.handlePostResponse(message)
		return

	case "error":
		if ws.debug.Load() {
			log.Printf("Received error message: %+v", response.Data)
		}
		return
	}

	// Process subscription messages with optimized matching
	ws.processSubscriptionMessage(response)
}

// handlePong processes pong messages and updates latency
func (ws *WebSocketAPI) handlePong() {
	if lastPingTimeValue := ws.lastPingTime.Load(); lastPingTimeValue != nil {
		if lastPingTime, ok := lastPingTimeValue.(time.Time); ok && !lastPingTime.IsZero() {
			latency := time.Since(lastPingTime).Milliseconds()
			ws.latencyMs.Store(latency)
			if ws.debug.Load() {
				log.Printf("Received pong response, latency: %dms", latency)
			}
		}
	}
}

// handlePostResponse processes post response messages
func (ws *WebSocketAPI) handlePostResponse(message []byte) {
	var postResponse WSPostResponse
	if err := FastUnmarshal(message, &postResponse); err != nil {
		if ws.debug.Load() {
			log.Printf("Failed to unmarshal post response: %v", err)
		}
		return
	}

	ws.mu.RLock()
	if ch, exists := ws.postResponses[postResponse.Data.ID]; exists {
		select {
		case ch <- postResponse.Data:
			if ws.debug.Load() {
				log.Printf("Sent post response to channel for ID: %d", postResponse.Data.ID)
			}
		default:
			if ws.debug.Load() {
				log.Printf("Post response channel full for ID: %d", postResponse.Data.ID)
			}
		}
	}
	ws.mu.RUnlock()
}

// processSubscriptionMessage efficiently processes subscription messages
func (ws *WebSocketAPI) processSubscriptionMessage(response *WSResponse) {
	ws.mu.RLock()
	handlers, exists := ws.channelHandlers[response.Channel]
	ws.mu.RUnlock()

	if !exists {
		if ws.debug.Load() {
			log.Printf("No handlers found for channel: %s", response.Channel)
		}
		return
	}

	// Process all handlers for this channel
	for _, handler := range handlers {
		if ws.matchesSubscription(handler, response) {
			select {
			case handler.Channel <- response.Data:
				if ws.debug.Load() {
					log.Printf("Sent data to subscription: %s", response.Channel)
				}
			default:
				if ws.debug.Load() {
					log.Printf("Handler channel full, skipping message")
				}
			}
		}
	}
}

// addSubscription adds a subscription with optimized storage
func (ws *WebSocketAPI) addSubscription(channel string, subType SubscriptionType, handler SubscriptionHandler, params map[string]string) error {
	// Create subscription with appropriate buffer size
	var bufferSize int
	switch subType {
	case SubTypeL2Book, SubTypeTrades:
		bufferSize = 50 // High frequency
	case SubTypeUserFills, SubTypeOrderUpdates:
		bufferSize = 100 // Medium frequency
	default:
		bufferSize = 10 // Low frequency
	}

	sub := &Subscription{
		Type:     subType,
		Params:   params,
		Handler:  handler,
		Channel:  make(chan interface{}, bufferSize),
		User:     params["user"],
		Coin:     params["coin"],
		Interval: params["interval"],
	}

	ws.mu.Lock()
	ws.subscriptions[channel] = sub

	// Add to channel handlers for fast lookup
	channelName := getChannelName(subType)
	ws.channelHandlers[channelName] = append(ws.channelHandlers[channelName], sub)
	ws.mu.Unlock()

	// Start handler goroutine
	go func() {
		for data := range sub.Channel {
			handler(data)
		}
	}()

	return nil
}

// getChannelName returns the channel name for a subscription type
func getChannelName(subType SubscriptionType) string {
	switch subType {
	case SubTypeUserFills:
		return "userFills"
	case SubTypeL2Book:
		return "l2Book"
	case SubTypeTrades:
		return "trades"
	case SubTypeOrderUpdates:
		return "orderUpdates"
	case SubTypeUserEvents:
		return "userEvents"
	case SubTypeUserFundings:
		return "userFundings"
	case SubTypeUserNonFundingLedgerUpdates:
		return "userNonFundingLedgerUpdates"
	case SubTypeUserTwapSliceFills:
		return "userTwapSliceFills"
	case SubTypeUserTwapHistory:
		return "userTwapHistory"
	case SubTypeActiveAssetCtx:
		return "activeAssetCtx"
	case SubTypeActiveAssetData:
		return "activeAssetData"
	case SubTypeBbo:
		return "bbo"
	case SubTypeCandle:
		return "candle"
	case SubTypeNotification:
		return "notification"
	case SubTypeWebData2:
		return "webData2"
	case SubTypeAllMids:
		return "allMids"
	default:
		return ""
	}
}

// matchesSubscription efficiently checks if a message matches a subscription
func (ws *WebSocketAPI) matchesSubscription(sub *Subscription, response *WSResponse) bool {
	switch sub.Type {
	case SubTypeUserFills:
		return ws.matchUserFills(sub, response)
	case SubTypeL2Book:
		return ws.matchL2Book(sub, response)
	case SubTypeTrades:
		return true // Accept all trades
	case SubTypeOrderUpdates, SubTypeUserEvents, SubTypeUserFundings,
		SubTypeUserNonFundingLedgerUpdates, SubTypeUserTwapSliceFills,
		SubTypeUserTwapHistory, SubTypeNotification, SubTypeWebData2:
		return true // Accept all messages for these types
	case SubTypeActiveAssetCtx:
		return ws.matchActiveAssetCtx(sub, response)
	case SubTypeActiveAssetData:
		return ws.matchActiveAssetData(sub, response)
	case SubTypeBbo:
		return ws.matchBbo(sub, response)
	case SubTypeCandle:
		return ws.matchCandle(sub, response)
	case SubTypeAllMids:
		return true // Accept all mids
	default:
		return false
	}
}

// matchUserFills checks if user fills message matches subscription
func (ws *WebSocketAPI) matchUserFills(sub *Subscription, response *WSResponse) bool {
	if dataMap, ok := response.Data.(map[string]interface{}); ok {
		if user, exists := dataMap["user"]; exists {
			if userStr, ok := user.(string); ok {
				return strings.EqualFold(userStr, sub.User)
			}
		}
	}
	return false
}

// matchL2Book checks if l2book message matches subscription
func (ws *WebSocketAPI) matchL2Book(sub *Subscription, response *WSResponse) bool {
	if dataMap, ok := response.Data.(map[string]interface{}); ok {
		if coin, exists := dataMap["coin"]; exists {
			if coinStr, ok := coin.(string); ok {
				return coinStr == sub.Coin
			}
		}
	}
	return false
}

// matchActiveAssetCtx checks if active asset context message matches subscription
func (ws *WebSocketAPI) matchActiveAssetCtx(sub *Subscription, response *WSResponse) bool {
	if dataMap, ok := response.Data.(map[string]interface{}); ok {
		if coin, exists := dataMap["coin"]; exists {
			if coinStr, ok := coin.(string); ok {
				return coinStr == sub.Coin
			}
		}
	}
	return false
}

// matchActiveAssetData checks if active asset data message matches subscription
func (ws *WebSocketAPI) matchActiveAssetData(sub *Subscription, response *WSResponse) bool {
	if dataMap, ok := response.Data.(map[string]interface{}); ok {
		if user, exists := dataMap["user"]; exists {
			if userStr, ok := user.(string); ok {
				if !strings.EqualFold(userStr, sub.User) {
					return false
				}
			}
		}
		if coin, exists := dataMap["coin"]; exists {
			if coinStr, ok := coin.(string); ok {
				return coinStr == sub.Coin
			}
		}
	}
	return false
}

// matchBbo checks if bbo message matches subscription
func (ws *WebSocketAPI) matchBbo(sub *Subscription, response *WSResponse) bool {
	if dataMap, ok := response.Data.(map[string]interface{}); ok {
		if coin, exists := dataMap["coin"]; exists {
			if coinStr, ok := coin.(string); ok {
				return coinStr == sub.Coin
			}
		}
	}
	return false
}

// matchCandle checks if candle message matches subscription
func (ws *WebSocketAPI) matchCandle(sub *Subscription, response *WSResponse) bool {
	if dataMap, ok := response.Data.(map[string]interface{}); ok {
		if coin, exists := dataMap["s"]; exists {
			if coinStr, ok := coin.(string); ok {
				if coinStr != sub.Coin {
					return false
				}
			}
		}
		if interval, exists := dataMap["i"]; exists {
			if intervalStr, ok := interval.(string); ok {
				return intervalStr == sub.Interval
			}
		}
	}
	return false
}

// pingHandler sends periodic ping messages to keep the connection alive
func (ws *WebSocketAPI) pingHandler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ws.IsConnected() {
				// Use pre-marshaled ping message for efficiency
				ws.mu.Lock()
				err := ws.conn.WriteMessage(websocket.TextMessage, ws.pingMessageBytes)
				ws.mu.Unlock()

				if err != nil {
					if ws.debug.Load() {
						log.Printf("Ping failed: %v", err)
					}
					return
				}

				// Record the timestamp when ping was sent
				ws.lastPingTime.Store(time.Now())

				if ws.debug.Load() {
					log.Printf("Ping sent successfully")
				}
			}
		case <-ws.pingStopChan:
			if ws.debug.Load() {
				log.Printf("Ping handler stopped")
			}
			return
		}
	}
}

// reconnect attempts to reconnect to the WebSocket server
func (ws *WebSocketAPI) reconnect() {
	ws.reconnectCount.Add(1)

	var backoff time.Duration
	switch ws.reconnectCount.Load() {
	case 1:
		backoff = 1 * time.Second
	case 2:
		backoff = 3 * time.Second
	case 3:
		backoff = 10 * time.Second
	case 4:
		backoff = 1 * time.Minute
	case 5:
		backoff = 1 * time.Hour
	default:
		backoff = 5 * time.Second
	}

	time.Sleep(backoff)

	// Store active subscriptions before reconnecting
	ws.mu.RLock()
	activeSubs := make([]*Subscription, 0, len(ws.subscriptions))
	for _, sub := range ws.subscriptions {
		activeSubs = append(activeSubs, sub)
	}
	ws.mu.RUnlock()

	// Close the old connection first to stop existing goroutines
	ws.mu.Lock()
	if ws.conn != nil {
		ws.conn.Close()
		ws.conn = nil
	}
	ws.isConnected.Store(false)

	// Stop ping handler
	if ws.pingStopChan != nil {
		close(ws.pingStopChan)
		ws.pingStopChan = nil
	}
	ws.mu.Unlock()

	// Wait longer for goroutines to properly terminate
	time.Sleep(2 * time.Second)

	if err := ws.Connect(); err != nil {
		log.Printf("Reconnection failed: %v", err)
		return
	}

	log.Println("Reconnected successfully")

	// Wait a moment for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	// Clear existing subscriptions to prevent duplicates
	ws.mu.Lock()
	ws.subscriptions = make(map[string]*Subscription)
	ws.channelHandlers = make(map[string][]*Subscription)
	ws.mu.Unlock()

	// Resubscribe to all active subscriptions using the new optimized approach
	successCount := 0
	for _, sub := range activeSubs {
		// Reconstruct the original subscription parameters
		params := make(map[string]interface{})

		switch sub.Type {
		case SubTypeUserFills:
			params["user"] = sub.User
			if err := ws.subscribe("userFills", params); err == nil {
				// Restore the handler function
				ws.addSubscription("userFills", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe userFills for user %s: %v", sub.User, err)
			}
		case SubTypeL2Book:
			params["coin"] = sub.Coin
			if err := ws.subscribe("l2Book", params); err == nil {
				// Restore the handler function
				ws.addSubscription("l2Book", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe l2Book for coin %s: %v", sub.Coin, err)
			}
		case SubTypeTrades:
			params["coin"] = sub.Coin
			if err := ws.subscribe("trades", params); err == nil {
				// Restore the handler function
				ws.addSubscription("trades", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe trades for coin %s: %v", sub.Coin, err)
			}
		case SubTypeAllMids:
			if err := ws.subscribe("allMids", params); err == nil {
				// Restore the handler function
				ws.addSubscription("allMids", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe allMids: %v", err)
			}
		case SubTypeUserEvents:
			params["user"] = sub.User
			if err := ws.subscribe("userEvents", params); err == nil {
				// Restore the handler function
				ws.addSubscription("userEvents", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe userEvents for user %s: %v", sub.User, err)
			}
		case SubTypeUserFundings:
			params["user"] = sub.User
			if err := ws.subscribe("userFundings", params); err == nil {
				// Restore the handler function
				ws.addSubscription("userFundings", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe userFundings for user %s: %v", sub.User, err)
			}
		case SubTypeUserNonFundingLedgerUpdates:
			params["user"] = sub.User
			if err := ws.subscribe("userNonFundingLedgerUpdates", params); err == nil {
				// Restore the handler function
				ws.addSubscription("userNonFundingLedgerUpdates", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe userNonFundingLedgerUpdates for user %s: %v", sub.User, err)
			}
		case SubTypeUserTwapSliceFills:
			params["user"] = sub.User
			if err := ws.subscribe("userTwapSliceFills", params); err == nil {
				// Restore the handler function
				ws.addSubscription("userTwapSliceFills", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe userTwapSliceFills for user %s: %v", sub.User, err)
			}
		case SubTypeUserTwapHistory:
			params["user"] = sub.User
			if err := ws.subscribe("userTwapHistory", params); err == nil {
				// Restore the handler function
				ws.addSubscription("userTwapHistory", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe userTwapHistory for user %s: %v", sub.User, err)
			}
		case SubTypeActiveAssetCtx:
			params["coin"] = sub.Coin
			if err := ws.subscribe("activeAssetCtx", params); err == nil {
				// Restore the handler function
				ws.addSubscription("activeAssetCtx", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe activeAssetCtx for coin %s: %v", sub.Coin, err)
			}
		case SubTypeActiveAssetData:
			params["user"] = sub.User
			params["coin"] = sub.Coin
			if err := ws.subscribe("activeAssetData", params); err == nil {
				// Restore the handler function
				ws.addSubscription("activeAssetData", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe activeAssetData for user %s coin %s: %v", sub.User, sub.Coin, err)
			}
		case SubTypeBbo:
			params["coin"] = sub.Coin
			if err := ws.subscribe("bbo", params); err == nil {
				// Restore the handler function
				ws.addSubscription("bbo", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe bbo for coin %s: %v", sub.Coin, err)
			}
		case SubTypeCandle:
			params["coin"] = sub.Coin
			params["interval"] = sub.Interval
			if err := ws.subscribe("candle", params); err == nil {
				// Restore the handler function
				ws.addSubscription("candle", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe candle for coin %s interval %s: %v", sub.Coin, sub.Interval, err)
			}
		case SubTypeOrderUpdates:
			params["user"] = sub.User
			if err := ws.subscribe("orderUpdates", params); err == nil {
				// Restore the handler function
				ws.addSubscription("orderUpdates", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe orderUpdates for user %s: %v", sub.User, err)
			}
		case SubTypeNotification:
			params["user"] = sub.User
			if err := ws.subscribe("notification", params); err == nil {
				// Restore the handler function
				ws.addSubscription("notification", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe notification for user %s: %v", sub.User, err)
			}
		case SubTypeWebData2:
			params["user"] = sub.User
			if err := ws.subscribe("webData2", params); err == nil {
				// Restore the handler function
				ws.addSubscription("webData2", sub.Type, sub.Handler, sub.Params)
				successCount++
			} else {
				log.Printf("Failed to resubscribe webData2 for user %s: %v", sub.User, err)
			}
		default:
			log.Printf("Unknown subscription type during resubscribe: %v", sub.Type)
		}
	}

	log.Printf("Resubscribed to %d/%d active subscriptions successfully", successCount, len(activeSubs))
}

// subscribe sends a subscription request to the WebSocket server
func (ws *WebSocketAPI) subscribe(subType string, params map[string]interface{}) error {
	// Try different subscription message formats
	var msg map[string]interface{}

	// Format 1: Standard format
	subscription := map[string]interface{}{
		"type": subType,
	}

	// Add parameters to the subscription
	if params != nil {
		for k, v := range params {
			subscription[k] = v
		}
	}

	msg = map[string]interface{}{
		"method":       "subscribe",
		"subscription": subscription,
	}

	b, err := FastMarshal(msg)
	if err != nil {
		return err
	}

	if ws.debug.Load() {
		log.Printf("Sending subscription message (format 1): %s", string(b))
	}

	ws.mu.Lock()
	err = ws.conn.WriteMessage(websocket.TextMessage, b)
	ws.mu.Unlock()

	if err != nil {
		return err
	}

	if ws.debug.Load() {
		log.Printf("Subscription message sent successfully for type: %s", subType)
	}

	return nil
}

// unsubscribe sends an unsubscribe request to the WebSocket server
func (ws *WebSocketAPI) unsubscribe(subType string, params map[string]interface{}) error {
	// Try different unsubscribe message formats
	var msg map[string]interface{}

	// Format 1: Standard format
	subscription := map[string]interface{}{
		"type": subType,
	}

	// Add parameters to the subscription
	if params != nil {
		for k, v := range params {
			subscription[k] = v
		}
	}

	msg = map[string]interface{}{
		"method":       "unsubscribe",
		"subscription": subscription,
	}

	b, err := FastMarshal(msg)
	if err != nil {
		return err
	}

	if ws.debug.Load() {
		log.Printf("Sending unsubscribe message: %s", string(b))
	}

	ws.mu.Lock()
	err = ws.conn.WriteMessage(websocket.TextMessage, b)
	ws.mu.Unlock()

	if err != nil {
		return err
	}

	if ws.debug.Load() {
		log.Printf("Unsubscribe message sent successfully for type: %s", subType)
	}

	return nil
}

// removeSubscription removes a subscription from the internal storage
func (ws *WebSocketAPI) removeSubscription(channel string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	// Remove from subscriptions map
	if sub, exists := ws.subscriptions[channel]; exists {
		// Close the channel
		close(sub.Channel)
		delete(ws.subscriptions, channel)

		// Remove from channelHandlers
		channelName := getChannelName(sub.Type)
		if handlers, exists := ws.channelHandlers[channelName]; exists {
			for i, handler := range handlers {
				if handler == sub {
					// Remove from slice
					ws.channelHandlers[channelName] = append(handlers[:i], handlers[i+1:]...)
					break
				}
			}
		}

		if ws.debug.Load() {
			log.Printf("Removed subscription for channel: %s", channel)
		}
	}

	return nil
}

// SubscribeOrderbook subscribes to orderbook updates for a specific coin
func (ws *WebSocketAPI) SubscribeOrderbook(coin string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("l2Book:%s", coin)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeL2Book, handler, map[string]string{
		"coin": coin,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("l2Book", map[string]interface{}{"coin": coin})
}

// SubscribeTrades subscribes to trade updates for a specific coin
func (ws *WebSocketAPI) SubscribeTrades(coin string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("trades:%s", coin)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeTrades, handler, map[string]string{
		"coin": coin,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("trades", map[string]interface{}{"coin": coin})
}

// SubscribeUserFills subscribes to user fill updates
func (ws *WebSocketAPI) SubscribeUserFills(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userFills:%s", user)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeUserFills, handler, map[string]string{
		"user": user,
	})
	if err != nil {
		return err
	}

	// Format the subscription message correctly
	msg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": "userFills",
			"user": user,
		},
	}

	b, err := FastMarshal(msg)
	if err != nil {
		return err
	}

	ws.mu.Lock()
	err = ws.conn.WriteMessage(websocket.TextMessage, b)
	ws.mu.Unlock()

	return err
}

// SubscribeAllMids subscribes to all mid price updates
func (ws *WebSocketAPI) SubscribeAllMids(handler SubscriptionHandler) error {
	channel := "allMids"

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeAllMids, handler, map[string]string{})
	if err != nil {
		return err
	}

	return ws.subscribe("allMids", nil)
}

// SubscribeUserEvents subscribes to user events for a specific user
func (ws *WebSocketAPI) SubscribeUserEvents(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userEvents:%s", user)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeUserEvents, handler, map[string]string{
		"user": user,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("userEvents", map[string]interface{}{"user": user})
}

// SubscribeUserFundings subscribes to user fundings for a specific user
func (ws *WebSocketAPI) SubscribeUserFundings(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userFundings:%s", user)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeUserFundings, handler, map[string]string{
		"user": user,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("userFundings", map[string]interface{}{"user": user})
}

// SubscribeUserNonFundingLedgerUpdates subscribes to user non-funding ledger updates for a specific user
func (ws *WebSocketAPI) SubscribeUserNonFundingLedgerUpdates(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userNonFundingLedgerUpdates:%s", user)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeUserNonFundingLedgerUpdates, handler, map[string]string{
		"user": user,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("userNonFundingLedgerUpdates", map[string]interface{}{"user": user})
}

// SubscribeUserTwapSliceFills subscribes to user TWAP slice fills for a specific user
func (ws *WebSocketAPI) SubscribeUserTwapSliceFills(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userTwapSliceFills:%s", user)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeUserTwapSliceFills, handler, map[string]string{
		"user": user,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("userTwapSliceFills", map[string]interface{}{"user": user})
}

// SubscribeUserTwapHistory subscribes to user TWAP history for a specific user
func (ws *WebSocketAPI) SubscribeUserTwapHistory(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userTwapHistory:%s", user)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeUserTwapHistory, handler, map[string]string{
		"user": user,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("userTwapHistory", map[string]interface{}{"user": user})
}

// SubscribeActiveAssetCtx subscribes to active asset context for a specific coin
func (ws *WebSocketAPI) SubscribeActiveAssetCtx(coin string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("activeAssetCtx:%s", coin)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeActiveAssetCtx, handler, map[string]string{
		"coin": coin,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("activeAssetCtx", map[string]interface{}{"coin": coin})
}

// SubscribeActiveAssetData subscribes to active asset data for a specific user and coin
func (ws *WebSocketAPI) SubscribeActiveAssetData(user string, coin string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("activeAssetData:%s:%s", user, coin)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeActiveAssetData, handler, map[string]string{
		"user": user,
		"coin": coin,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("activeAssetData", map[string]interface{}{
		"user": user,
		"coin": coin,
	})
}

// SubscribeBbo subscribes to best bid/offer updates for a specific coin
func (ws *WebSocketAPI) SubscribeBbo(coin string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("bbo:%s", coin)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeBbo, handler, map[string]string{
		"coin": coin,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("bbo", map[string]interface{}{"coin": coin})
}

// SubscribeCandle subscribes to candle updates for a specific coin and interval
func (ws *WebSocketAPI) SubscribeCandle(coin string, interval string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("candle:%s:%s", coin, interval)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeCandle, handler, map[string]string{
		"coin":     coin,
		"interval": interval,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("candle", map[string]interface{}{
		"coin":     coin,
		"interval": interval,
	})
}

// SubscribeOrderUpdates subscribes to order updates for a specific user
func (ws *WebSocketAPI) SubscribeOrderUpdates(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("orderUpdates:%s", user)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeOrderUpdates, handler, map[string]string{
		"user": user,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("orderUpdates", map[string]interface{}{"user": user})
}

// SubscribeNotification subscribes to notifications for a specific user
func (ws *WebSocketAPI) SubscribeNotification(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("notification:%s", user)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeNotification, handler, map[string]string{
		"user": user,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("notification", map[string]interface{}{"user": user})
}

// SubscribeWebData2 subscribes to web data for a specific user
func (ws *WebSocketAPI) SubscribeWebData2(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("webData2:%s", user)

	// Add subscription with optimized storage
	err := ws.addSubscription(channel, SubTypeWebData2, handler, map[string]string{
		"user": user,
	})
	if err != nil {
		return err
	}

	return ws.subscribe("webData2", map[string]interface{}{"user": user})
}

// GetLatency returns the current latency in milliseconds
func (ws *WebSocketAPI) GetLatency() int64 {
	return ws.latencyMs.Load()
}

// PostRequest sends a post request through WebSocket and returns the response
func (ws *WebSocketAPI) PostRequest(requestType string, payload interface{}) (*WSPostResponseData, error) {
	postID := int(ws.nextPostID.Add(1))

	// Create response channel
	responseCh := make(chan WSPostResponseData, 1)
	ws.mu.Lock()
	ws.postResponses[postID] = responseCh
	ws.mu.Unlock()

	// Clean up response channel after use
	defer func() {
		ws.mu.Lock()
		delete(ws.postResponses, postID)
		ws.mu.Unlock()
	}()

	// Create post request
	postRequest := WSPostRequest{
		Method: "post",
		ID:     postID,
		Request: WSPostData{
			Type:    requestType,
			Payload: payload,
		},
	}

	// Marshal and send request
	b, err := FastMarshal(postRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal post request: %w", err)
	}

	if ws.debug.Load() {
		log.Printf("Sending post request: %s", string(b))
	}

	ws.mu.Lock()
	err = ws.conn.WriteMessage(websocket.TextMessage, b)
	ws.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to send post request: %w", err)
	}

	// Wait for response with timeout
	select {
	case response := <-responseCh:
		return &response, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("post request timeout")
	}
}

// PostInfoRequest sends an info request through WebSocket
func (ws *WebSocketAPI) PostInfoRequest(payload interface{}) (*WSPostResponseData, error) {
	return ws.PostRequest("info", payload)
}

// PostActionRequest sends an action request through WebSocket
func (ws *WebSocketAPI) PostActionRequest(payload interface{}) (*WSPostResponseData, error) {
	return ws.PostRequest("action", payload)
}

// PostOrderRequest sends an order request through WebSocket
func (ws *WebSocketAPI) PostOrderRequest(action interface{}, nonce uint64, signature RsvSignature, vaultAddress *string) (*WSPostResponseData, error) {
	payload := map[string]interface{}{
		"action":    action,
		"nonce":     nonce,
		"signature": signature,
	}
	if vaultAddress != nil {
		payload["vaultAddress"] = *vaultAddress
	}

	return ws.PostActionRequest(payload)
}

// UnsubscribeOrderbook unsubscribes from orderbook updates for a specific coin
func (ws *WebSocketAPI) UnsubscribeOrderbook(coin string) error {
	channel := fmt.Sprintf("l2Book:%s", coin)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("l2Book", map[string]interface{}{"coin": coin})
}

// UnsubscribeTrades unsubscribes from trade updates for a specific coin
func (ws *WebSocketAPI) UnsubscribeTrades(coin string) error {
	channel := fmt.Sprintf("trades:%s", coin)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("trades", map[string]interface{}{"coin": coin})
}

// UnsubscribeUserFills unsubscribes from user fill updates
func (ws *WebSocketAPI) UnsubscribeUserFills(user string) error {
	channel := fmt.Sprintf("userFills:%s", user)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("userFills", map[string]interface{}{"user": user})
}

// UnsubscribeAllMids unsubscribes from all mids updates
func (ws *WebSocketAPI) UnsubscribeAllMids() error {
	channel := "allMids"

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("allMids", nil)
}

// UnsubscribeUserEvents unsubscribes from user events
func (ws *WebSocketAPI) UnsubscribeUserEvents(user string) error {
	channel := fmt.Sprintf("userEvents:%s", user)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("userEvents", map[string]interface{}{"user": user})
}

// UnsubscribeUserFundings unsubscribes from user fundings
func (ws *WebSocketAPI) UnsubscribeUserFundings(user string) error {
	channel := fmt.Sprintf("userFundings:%s", user)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("userFundings", map[string]interface{}{"user": user})
}

// UnsubscribeUserNonFundingLedgerUpdates unsubscribes from user non-funding ledger updates
func (ws *WebSocketAPI) UnsubscribeUserNonFundingLedgerUpdates(user string) error {
	channel := fmt.Sprintf("userNonFundingLedgerUpdates:%s", user)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("userNonFundingLedgerUpdates", map[string]interface{}{"user": user})
}

// UnsubscribeUserTwapSliceFills unsubscribes from user TWAP slice fills
func (ws *WebSocketAPI) UnsubscribeUserTwapSliceFills(user string) error {
	channel := fmt.Sprintf("userTwapSliceFills:%s", user)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("userTwapSliceFills", map[string]interface{}{"user": user})
}

// UnsubscribeUserTwapHistory unsubscribes from user TWAP history
func (ws *WebSocketAPI) UnsubscribeUserTwapHistory(user string) error {
	channel := fmt.Sprintf("userTwapHistory:%s", user)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("userTwapHistory", map[string]interface{}{"user": user})
}

// UnsubscribeActiveAssetCtx unsubscribes from active asset context
func (ws *WebSocketAPI) UnsubscribeActiveAssetCtx(coin string) error {
	channel := fmt.Sprintf("activeAssetCtx:%s", coin)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("activeAssetCtx", map[string]interface{}{"coin": coin})
}

// UnsubscribeActiveAssetData unsubscribes from active asset data
func (ws *WebSocketAPI) UnsubscribeActiveAssetData(user string, coin string) error {
	channel := fmt.Sprintf("activeAssetData:%s:%s", user, coin)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("activeAssetData", map[string]interface{}{"user": user, "coin": coin})
}

// UnsubscribeBbo unsubscribes from best bid/offer updates
func (ws *WebSocketAPI) UnsubscribeBbo(coin string) error {
	channel := fmt.Sprintf("bbo:%s", coin)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("bbo", map[string]interface{}{"coin": coin})
}

// UnsubscribeCandle unsubscribes from candle updates
func (ws *WebSocketAPI) UnsubscribeCandle(coin string, interval string) error {
	channel := fmt.Sprintf("candle:%s:%s", coin, interval)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("candle", map[string]interface{}{"coin": coin, "interval": interval})
}

// UnsubscribeOrderUpdates unsubscribes from order updates
func (ws *WebSocketAPI) UnsubscribeOrderUpdates(user string) error {
	channel := fmt.Sprintf("orderUpdates:%s", user)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("orderUpdates", map[string]interface{}{"user": user})
}

// UnsubscribeNotification unsubscribes from notifications
func (ws *WebSocketAPI) UnsubscribeNotification(user string) error {
	channel := fmt.Sprintf("notification:%s", user)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("notification", map[string]interface{}{"user": user})
}

// UnsubscribeWebData2 unsubscribes from web data 2
func (ws *WebSocketAPI) UnsubscribeWebData2(user string) error {
	channel := fmt.Sprintf("webData2:%s", user)

	// Remove from internal storage
	if err := ws.removeSubscription(channel); err != nil {
		return err
	}

	return ws.unsubscribe("webData2", map[string]interface{}{"user": user})
}
