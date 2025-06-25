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
			}
			continue
		}

		if ws.debug {
			log.Printf("Processing WebSocket message for channel: %s", response.Channel)
		}

		// Try to find a matching subscription channel
		ws.mu.RLock()
		var found bool
		for channel := range ws.subscriptions {
			parts := strings.Split(channel, ":")
			if len(parts) != 2 {
				continue
			}
			subType := parts[0]
			param := parts[1]

			// Check if this message matches our subscription
			if subType == "userFills" && response.Channel == "userFills" {
				// For userFills, check if the user matches
				if userData, ok := response.Data.(map[string]interface{}); ok {
					if user, exists := userData["user"]; exists && user == param {
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
			} else if subType == "l2Book" && response.Channel == "l2Book" {
				// For orderbook, check if the coin matches
				if bookData, ok := response.Data.(map[string]interface{}); ok {
					if coin, exists := bookData["coin"]; exists && coin == param {
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
			} else if subType == "trades" && response.Channel == "trades" {
				// For trades, the server should filter by coin, so we just need to match the channel
				// But we need to handle the data structure safely
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
			log.Printf("No matching subscription found for message: %s", string(message))
		}
	}
}

// pingHandler sends periodic ping messages to keep the connection alive
func (ws *WebSocketClient) pingHandler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Check if we should still be connected
		ws.mu.RLock()
		if !ws.isConnected || ws.manualDisconnect {
			ws.mu.RUnlock()
			return
		}
		conn := ws.conn
		ws.mu.RUnlock()

		if conn == nil {
			return
		}

		ws.writeMu.Lock()
		err := conn.WriteMessage(websocket.PingMessage, nil)
		ws.writeMu.Unlock()

		if err != nil {
			if ws.debug {
				log.Printf("Ping failed: %v", err)
			}
			// Don't trigger reconnect here - let readMessages handle it
			// Just close the connection to trigger the readMessages error
			conn.Close()
			return
		}

		if ws.debug {
			log.Printf("Ping sent successfully")
		}
	}
}

// reconnect attempts to reconnect to the WebSocket server
func (ws *WebSocketClient) reconnect() {
	ws.reconnectCount++
	// Exponential backoff: 1s, 2s, 4s, 8s, 16s
	backoff := time.Duration(1<<(ws.reconnectCount-1)) * time.Second

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
		// Don't reset reconnect count on failure
		return
	} else {
		log.Println("Reconnected successfully")
		// Reset reconnect count on successful connection
		ws.reconnectCount = 0

		// Resubscribe to all active subscriptions
		for _, channel := range activeSubs {
			parts := strings.Split(channel, ":")
			if len(parts) != 2 {
				continue
			}

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
			}
		}
	}
}

// subscribe sends a subscription request to the WebSocket server
func (ws *WebSocketClient) subscribe(subType string, params map[string]interface{}) error {
	msg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": subType,
		},
	}

	if params != nil {
		for k, v := range params {
			msg["subscription"].(map[string]interface{})[k] = v
		}
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ws.writeMu.Lock()
	err = ws.conn.WriteMessage(websocket.TextMessage, b)
	ws.writeMu.Unlock()

	if err != nil {
		return err
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
	channel := "userEvents"

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

// SubscribeUserFundings subscribes to user funding updates for a specific user
func (ws *WebSocketClient) SubscribeUserFundings(user string, handler SubscriptionHandler) error {
	channel := "userFundings"

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
	channel := "userNonFundingLedgerUpdates"

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
	channel := "userTwapSliceFills"

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
	channel := "userTwapHistory"

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
	channel := "activeAssetCtx"

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
	channel := "activeAssetData"

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
	channel := "bbo"

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
	channel := "candle"

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
	channel := "orderUpdates"

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
	channel := "notification"

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
	channel := "webData2"

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
