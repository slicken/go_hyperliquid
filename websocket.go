package hyperliquid

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketClient handles WebSocket connections and subscriptions
type WebSocketClient struct {
	conn             *websocket.Conn
	url              string
	subscriptions    map[string]chan interface{}
	activeSubs       []string // Store active subscription channel names
	mu               sync.RWMutex
	writeMu          sync.Mutex
	reconnectCount   int
	maxReconnects    int
	isConnected      bool
	debug            bool
	manualDisconnect bool // Flag to prevent auto-reconnect on manual disconnect
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

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(isMainnet bool) *WebSocketClient {
	wsURL := MainnetWSURL
	if !isMainnet {
		wsURL = TestnetWSURL
	}

	return &WebSocketClient{
		url:           wsURL,
		subscriptions: make(map[string]chan interface{}),
		activeSubs:    make([]string, 0),
		maxReconnects: 5,
	}
}

// Connect establishes a WebSocket connection
func (ws *WebSocketClient) Connect() error {
	u, err := url.Parse(ws.url)
	if err != nil {
		return fmt.Errorf("failed to parse WebSocket URL: %w", err)
	}

	if ws.debug {
		log.Printf("Connecting to WebSocket URL: %s", u.String())
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	ws.conn = conn
	ws.isConnected = true
	ws.manualDisconnect = false // Reset manual disconnect flag
	ws.reconnectCount = 0

	if ws.debug {
		log.Println("WebSocket connection established")
	}

	go ws.readMessages()
	go ws.pingHandler()

	return nil
}

// Disconnect closes the WebSocket connection and prevents auto-reconnect
func (ws *WebSocketClient) Disconnect() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.isConnected = false
	ws.manualDisconnect = true // Prevent auto-reconnect
	ws.reconnectCount = 0      // Reset reconnect count on manual disconnect
	if ws.conn != nil {
		return ws.conn.Close()
	}
	return nil
}

// IsConnected returns the connection status
func (ws *WebSocketClient) IsConnected() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.isConnected
}

// SetDebugActive enables debug mode
func (ws *WebSocketClient) SetDebug(status bool) {
	ws.debug = status
}

// readMessages reads messages from the WebSocket connection
func (ws *WebSocketClient) readMessages() {
	defer func() {
		ws.mu.Lock()
		ws.isConnected = false
		ws.mu.Unlock()
	}()

	for {
		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			if ws.debug {
				log.Printf("WebSocket read error: %v", err)
			}
			// Don't reconnect if manually disconnected
			ws.mu.RLock()
			manualDisconnect := ws.manualDisconnect
			ws.mu.RUnlock()

			if !manualDisconnect && ws.reconnectCount < ws.maxReconnects {
				ws.reconnect()
			}
			return
		}

		if ws.debug {
			log.Printf("Received WebSocket message: %s", string(message))
		}

		var response WSResponse
		if err := json.Unmarshal(message, &response); err != nil {
			if ws.debug {
				log.Printf("Failed to unmarshal WebSocket message: %v", err)
				log.Printf("Raw message: %s", string(message))
			}
			continue
		}

		if ws.debug {
			log.Printf("Processing WebSocket message for channel: %s, data type: %T", response.Channel, response.Data)
			if response.Data != nil {
				log.Printf("Data content: %+v", response.Data)
			}
		}

		// Check if this is a subscription acknowledgment
		if response.Channel == "subscribed" || response.Channel == "subscription" || response.Channel == "subscriptionResponse" {
			if ws.debug {
				log.Printf("Received subscription acknowledgment: %+v", response.Data)
			}
			continue
		}

		// Check if this is a pong response to our ping
		if response.Channel == "pong" {
			if ws.debug {
				log.Printf("Received pong response")
			}
			continue
		}

		// Log all messages for debugging
		if ws.debug {
			log.Printf("Processing message with channel: %s", response.Channel)
		}

		// If no channel field, try to parse as a different format
		if response.Channel == "" {
			if ws.debug {
				log.Printf("No channel field found, trying alternative message format")
			}
			// Try to parse as a direct message without channel field
			var directMsg map[string]interface{}
			if err := json.Unmarshal(message, &directMsg); err == nil {
				if ws.debug {
					log.Printf("Parsed as direct message: %+v", directMsg)
				}
				// Try to extract channel from the message
				if channel, exists := directMsg["channel"]; exists {
					response.Channel = channel.(string)
				} else if msgType, exists := directMsg["type"]; exists {
					response.Channel = msgType.(string)
				}
				if data, exists := directMsg["data"]; exists {
					response.Data = data
				}
			}
		}

		// Try to find a matching subscription channel
		ws.mu.RLock()
		var found bool
		for channel := range ws.subscriptions {
			parts := strings.Split(channel, ":")
			if len(parts) < 2 {
				continue
			}
			subType := parts[0]
			param := parts[1]

			// For channels with more than 2 parts (like candle:BTC:1m),
			// reconstruct the param to include all remaining parts
			if len(parts) > 2 {
				param = strings.Join(parts[1:], ":")
			}

			if ws.debug {
				log.Printf("Checking subscription: %s (type: %s, param: %s) against channel: %s", channel, subType, param, response.Channel)
			}

			// Check if this message matches our subscription
			var matches bool

			if subType == "userFills" && response.Channel == "userFills" {
				// For userFills, check if the user matches (case-insensitive)
				if dataMap, ok := response.Data.(map[string]interface{}); ok {
					if user, exists := dataMap["user"]; exists {
						userStr, ok := user.(string)
						if ok && strings.EqualFold(userStr, param) {
							matches = true
						}
					}
				}
			} else if subType == "l2Book" && response.Channel == "l2Book" {
				// For l2Book, check if the coin matches
				if dataMap, ok := response.Data.(map[string]interface{}); ok {
					if coin, exists := dataMap["coin"]; exists && coin == param {
						matches = true
					}
				}
			} else if subType == "trades" && response.Channel == "trades" {
				// For trades, accept any trades message when subscribed to trades
				// This is because Hyperliquid might send all trades regardless of coin subscription
				matches = true
				if ws.debug {
					log.Printf("Accepting trades message for subscription: %s", channel)
				}
			} else if subType == "orderUpdates" && response.Channel == "orderUpdates" {
				// For orderUpdates, accept all messages since they're already filtered by the subscription
				// The data is an array of order updates
				matches = true
				if ws.debug {
					log.Printf("Accepting orderUpdates message for subscription: %s", channel)
				}
			} else if subType == "userEvents" && response.Channel == "userEvents" {
				// For userEvents, accept all messages since they're already filtered by the subscription
				matches = true
				if ws.debug {
					log.Printf("Accepting userEvents message for subscription: %s", channel)
				}
			} else if subType == "userFundings" && response.Channel == "userFundings" {
				// For userFundings, accept all messages since they're already filtered by the subscription
				matches = true
				if ws.debug {
					log.Printf("Accepting userFundings message for subscription: %s", channel)
				}
			} else if subType == "userNonFundingLedgerUpdates" && response.Channel == "userNonFundingLedgerUpdates" {
				// For userNonFundingLedgerUpdates, accept all messages since they're already filtered by the subscription
				matches = true
				if ws.debug {
					log.Printf("Accepting userNonFundingLedgerUpdates message for subscription: %s", channel)
				}
			} else if subType == "userTwapSliceFills" && response.Channel == "userTwapSliceFills" {
				// For userTwapSliceFills, accept all messages since they're already filtered by the subscription
				matches = true
				if ws.debug {
					log.Printf("Accepting userTwapSliceFills message for subscription: %s", channel)
				}
			} else if subType == "userTwapHistory" && response.Channel == "userTwapHistory" {
				// For userTwapHistory, accept all messages since they're already filtered by the subscription
				matches = true
				if ws.debug {
					log.Printf("Accepting userTwapHistory message for subscription: %s", channel)
				}
			} else if subType == "activeAssetCtx" && response.Channel == "activeAssetCtx" {
				// For activeAssetCtx, check if the coin matches
				if dataMap, ok := response.Data.(map[string]interface{}); ok {
					if coin, exists := dataMap["coin"]; exists && coin == param {
						matches = true
					}
				}
			} else if subType == "activeAssetData" && response.Channel == "activeAssetData" {
				// For activeAssetData, check if the user and coin match
				parts := strings.Split(param, ":")
				if len(parts) == 2 {
					userParam := parts[0]
					coinParam := parts[1]
					if dataMap, ok := response.Data.(map[string]interface{}); ok {
						if user, exists := dataMap["user"]; exists {
							userStr, ok := user.(string)
							if ok && strings.EqualFold(userStr, userParam) {
								if coin, exists := dataMap["coin"]; exists && coin == coinParam {
									matches = true
								}
							}
						}
					}
				}
			} else if subType == "bbo" && response.Channel == "bbo" {
				// For bbo, check if the coin matches
				if dataMap, ok := response.Data.(map[string]interface{}); ok {
					if coin, exists := dataMap["coin"]; exists && coin == param {
						matches = true
					}
				}
			} else if subType == "candle" && response.Channel == "candle" {
				// For candle, check if the coin and interval match
				// Channel format is "candle:BTC:1m", so param is "BTC:1m"
				parts := strings.Split(param, ":")
				if len(parts) == 2 {
					coinParam := parts[0]
					intervalParam := parts[1]
					if dataMap, ok := response.Data.(map[string]interface{}); ok {
						// Check for 's' field (symbol/coin) and 'i' field (interval)
						if coin, exists := dataMap["s"]; exists && coin == coinParam {
							if interval, exists := dataMap["i"]; exists && interval == intervalParam {
								matches = true
							}
						}
					}
				}
			} else if subType == "notification" && response.Channel == "notification" {
				// For notification, accept all messages since they're already filtered by the subscription
				matches = true
				if ws.debug {
					log.Printf("Accepting notification message for subscription: %s", channel)
				}
			} else if subType == "webData2" && response.Channel == "webData2" {
				// For webData2, accept all messages since they're already filtered by the subscription
				matches = true
				if ws.debug {
					log.Printf("Accepting webData2 message for subscription: %s", channel)
				}
			}

			if matches {
				if ch, exists := ws.subscriptions[channel]; exists {
					select {
					case ch <- response.Data:
						if ws.debug {
							log.Printf("Sent data to channel: %s", channel)
						}
						found = true
					default:
						if ws.debug {
							log.Printf("Channel full, skipping message for: %s", channel)
						}
					}
				}
			}
		}
		ws.mu.RUnlock()

		if !found && ws.debug {
			log.Printf("No matching subscription found for message. Channel: %s, Data type: %T", response.Channel, response.Data)
		}
	}
}

// pingHandler sends periodic ping messages to keep the connection alive
func (ws *WebSocketClient) pingHandler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if ws.IsConnected() {
			// Use Hyperliquid's custom ping/pong protocol
			// Client sends: { "method": "ping" }
			// Server responds: { "channel": "pong" }
			pingMsg := map[string]interface{}{
				"method": "ping",
			}

			b, err := json.Marshal(pingMsg)
			if err != nil {
				if ws.debug {
					log.Printf("Failed to marshal ping message: %v", err)
				}
				continue
			}

			ws.writeMu.Lock()
			err = ws.conn.WriteMessage(websocket.TextMessage, b)
			ws.writeMu.Unlock()

			if err != nil {
				if ws.debug {
					log.Printf("Ping failed: %v", err)
				}
				return
			}
			if ws.debug {
				log.Printf("Ping sent successfully")
			}
		}
	}
}

// reconnect attempts to reconnect to the WebSocket server
func (ws *WebSocketClient) reconnect() {
	ws.reconnectCount++
	backoff := time.Duration(ws.reconnectCount) * time.Second

	time.Sleep(backoff)

	// Store active subscriptions before reconnecting
	ws.mu.RLock()
	activeSubs := make([]string, len(ws.activeSubs))
	copy(activeSubs, ws.activeSubs)
	ws.mu.RUnlock()

	// Close the old connection first to stop existing goroutines
	ws.mu.Lock()
	if ws.conn != nil {
		ws.conn.Close()
		ws.conn = nil
	}
	ws.isConnected = false
	ws.mu.Unlock()

	if err := ws.Connect(); err != nil {
		log.Printf("Reconnection failed: %v", err)
	} else {
		log.Println("Reconnected successfully")

		// Resubscribe to all active subscriptions
		for _, channel := range activeSubs {
			parts := strings.Split(channel, ":")

			// Handle different channel formats
			if len(parts) == 2 {
				subType := parts[0]
				param := parts[1]

				switch subType {
				case "userFills":
					msg := map[string]interface{}{
						"method": "subscribe",
						"subscription": map[string]interface{}{
							"type": "userFills",
							"user": param,
						},
					}

					b, err := json.Marshal(msg)
					if err != nil {
						log.Printf("Failed to marshal resubscription message: %v", err)
						continue
					}

					ws.writeMu.Lock()
					err = ws.conn.WriteMessage(websocket.TextMessage, b)
					ws.writeMu.Unlock()

					if err != nil {
						log.Printf("Failed to send resubscription message: %v", err)
					} else {
						log.Printf("Resubscribed to user fills for user: %s", param)
					}
				case "l2Book":
					msg := map[string]interface{}{
						"method": "subscribe",
						"subscription": map[string]interface{}{
							"type": "l2Book",
							"coin": param,
						},
					}

					b, err := json.Marshal(msg)
					if err != nil {
						log.Printf("Failed to marshal resubscription message: %v", err)
						continue
					}

					ws.writeMu.Lock()
					err = ws.conn.WriteMessage(websocket.TextMessage, b)
					ws.writeMu.Unlock()

					if err != nil {
						log.Printf("Failed to send resubscription message: %v", err)
					} else {
						log.Printf("Resubscribed to orderbook for coin: %s", param)
					}
				case "trades":
					msg := map[string]interface{}{
						"method": "subscribe",
						"subscription": map[string]interface{}{
							"type": "trades",
							"coin": param,
						},
					}

					b, err := json.Marshal(msg)
					if err != nil {
						log.Printf("Failed to marshal resubscription message: %v", err)
						continue
					}

					ws.writeMu.Lock()
					err = ws.conn.WriteMessage(websocket.TextMessage, b)
					ws.writeMu.Unlock()

					if err != nil {
						log.Printf("Failed to send resubscription message: %v", err)
					} else {
						log.Printf("Resubscribed to trades for coin: %s", param)
					}
				case "orderUpdates":
					msg := map[string]interface{}{
						"method": "subscribe",
						"subscription": map[string]interface{}{
							"type": "orderUpdates",
							"user": param,
						},
					}

					b, err := json.Marshal(msg)
					if err != nil {
						log.Printf("Failed to marshal resubscription message: %v", err)
						continue
					}

					ws.writeMu.Lock()
					err = ws.conn.WriteMessage(websocket.TextMessage, b)
					ws.writeMu.Unlock()

					if err != nil {
						log.Printf("Failed to send resubscription message: %v", err)
					} else {
						log.Printf("Resubscribed to order updates for user: %s", param)
					}
				}
			} else if len(parts) == 3 {
				subType := parts[0]
				param1 := parts[1]
				param2 := parts[2]

				switch subType {
				case "candle":
					msg := map[string]interface{}{
						"method": "subscribe",
						"subscription": map[string]interface{}{
							"type":     "candle",
							"coin":     param1,
							"interval": param2,
						},
					}

					b, err := json.Marshal(msg)
					if err != nil {
						log.Printf("Failed to marshal resubscription message: %v", err)
						continue
					}

					ws.writeMu.Lock()
					err = ws.conn.WriteMessage(websocket.TextMessage, b)
					ws.writeMu.Unlock()

					if err != nil {
						log.Printf("Failed to send resubscription message: %v", err)
					} else {
						log.Printf("Resubscribed to candle for coin: %s, interval: %s", param1, param2)
					}
				}
			}
		}
	}
}

// subscribe sends a subscription request to the WebSocket server
func (ws *WebSocketClient) subscribe(subType string, params map[string]interface{}) error {
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

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if ws.debug {
		log.Printf("Sending subscription message (format 1): %s", string(b))
	}

	ws.writeMu.Lock()
	err = ws.conn.WriteMessage(websocket.TextMessage, b)
	ws.writeMu.Unlock()

	if err != nil {
		return err
	}

	if ws.debug {
		log.Printf("Subscription message sent successfully for type: %s", subType)
	}

	return nil
}

// SubscribeOrderbook subscribes to orderbook updates for a specific coin
func (ws *WebSocketClient) SubscribeOrderbook(coin string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("l2Book:%s", coin)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("l2Book", map[string]interface{}{"coin": coin})
}

// SubscribeTrades subscribes to trade updates for a specific coin
func (ws *WebSocketClient) SubscribeTrades(coin string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("trades:%s", coin)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("trades", map[string]interface{}{"coin": coin})
}

// SubscribeUserFills subscribes to user fill updates
func (ws *WebSocketClient) SubscribeUserFills(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userFills:%s", user)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	// Format the subscription message correctly
	msg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": "userFills",
			"user": user,
		},
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ws.writeMu.Lock()
	err = ws.conn.WriteMessage(websocket.TextMessage, b)
	ws.writeMu.Unlock()

	return err
}

// SubscribeAllMids subscribes to all mid price updates
func (ws *WebSocketClient) SubscribeAllMids(handler SubscriptionHandler) error {
	channel := "allMids"

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("allMids", nil)
}

// SubscribeUserEvents subscribes to user events for a specific user
func (ws *WebSocketClient) SubscribeUserEvents(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userEvents:%s", user)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("userEvents", map[string]interface{}{"user": user})
}

// SubscribeUserFundings subscribes to user fundings for a specific user
func (ws *WebSocketClient) SubscribeUserFundings(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userFundings:%s", user)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("userFundings", map[string]interface{}{"user": user})
}

// SubscribeUserNonFundingLedgerUpdates subscribes to user non-funding ledger updates for a specific user
func (ws *WebSocketClient) SubscribeUserNonFundingLedgerUpdates(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userNonFundingLedgerUpdates:%s", user)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("userNonFundingLedgerUpdates", map[string]interface{}{"user": user})
}

// SubscribeUserTwapSliceFills subscribes to user TWAP slice fills for a specific user
func (ws *WebSocketClient) SubscribeUserTwapSliceFills(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userTwapSliceFills:%s", user)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("userTwapSliceFills", map[string]interface{}{"user": user})
}

// SubscribeUserTwapHistory subscribes to user TWAP history for a specific user
func (ws *WebSocketClient) SubscribeUserTwapHistory(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("userTwapHistory:%s", user)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("userTwapHistory", map[string]interface{}{"user": user})
}

// SubscribeActiveAssetCtx subscribes to active asset context for a specific coin
func (ws *WebSocketClient) SubscribeActiveAssetCtx(coin string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("activeAssetCtx:%s", coin)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("activeAssetCtx", map[string]interface{}{"coin": coin})
}

// SubscribeActiveAssetData subscribes to active asset data for a specific user and coin
func (ws *WebSocketClient) SubscribeActiveAssetData(user string, coin string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("activeAssetData:%s:%s", user, coin)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("activeAssetData", map[string]interface{}{
		"user": user,
		"coin": coin,
	})
}

// SubscribeBbo subscribes to best bid/offer updates for a specific coin
func (ws *WebSocketClient) SubscribeBbo(coin string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("bbo:%s", coin)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("bbo", map[string]interface{}{"coin": coin})
}

// SubscribeCandle subscribes to candle updates for a specific coin and interval
func (ws *WebSocketClient) SubscribeCandle(coin string, interval string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("candle:%s:%s", coin, interval)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("candle", map[string]interface{}{
		"coin":     coin,
		"interval": interval,
	})
}

// SubscribeOrderUpdates subscribes to order updates for a specific user
func (ws *WebSocketClient) SubscribeOrderUpdates(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("orderUpdates:%s", user)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("orderUpdates", map[string]interface{}{"user": user})
}

// SubscribeNotification subscribes to notifications for a specific user
func (ws *WebSocketClient) SubscribeNotification(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("notification:%s", user)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("notification", map[string]interface{}{"user": user})
}

// SubscribeWebData2 subscribes to web data for a specific user
func (ws *WebSocketClient) SubscribeWebData2(user string, handler SubscriptionHandler) error {
	channel := fmt.Sprintf("webData2:%s", user)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.activeSubs = append(ws.activeSubs, channel)
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("webData2", map[string]interface{}{"user": user})
}
