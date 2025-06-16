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
	conn           *websocket.Conn
	url            string
	subscriptions  map[string]chan interface{}
	mu             sync.RWMutex
	writeMu        sync.Mutex
	reconnectCount int
	maxReconnects  int
	isConnected    bool
	debug          bool
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
	ws.reconnectCount = 0

	if ws.debug {
		log.Println("WebSocket connection established")
	}

	go ws.readMessages()
	go ws.pingHandler()

	return nil
}

// Disconnect closes the WebSocket connection
func (ws *WebSocketClient) Disconnect() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.isConnected = false
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
			if ws.reconnectCount < ws.maxReconnects {
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

		ws.mu.RLock()
		if ch, exists := ws.subscriptions[response.Channel]; exists {
			select {
			case ch <- response.Data:
				if ws.debug {
					log.Printf("Sent data to channel: %s", response.Channel)
				}
			default:
				if ws.debug {
					log.Printf("Channel full, skipping message for: %s", response.Channel)
				}
			}
		} else {
			if ws.debug {
				log.Printf("No subscription found for channel: %s", response.Channel)
			}
		}
		ws.mu.RUnlock()
	}
}

// pingHandler sends periodic ping messages to keep the connection alive
func (ws *WebSocketClient) pingHandler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ws.IsConnected() {
				ws.writeMu.Lock()
				err := ws.conn.WriteMessage(websocket.PingMessage, nil)
				ws.writeMu.Unlock()
				if err != nil {
					return
				}
			}
		}
	}
}

// reconnect attempts to reconnect to the WebSocket server
func (ws *WebSocketClient) reconnect() {
	ws.reconnectCount++
	backoff := time.Duration(ws.reconnectCount) * time.Second

	time.Sleep(backoff)

	// Store current subscriptions before reconnecting
	ws.mu.RLock()
	activeSubscriptions := make(map[string]map[string]interface{})
	for channel := range ws.subscriptions {
		activeSubscriptions[channel] = make(map[string]interface{})
		// Get the subscription parameters from the channel name
		// This assumes the channel name contains the necessary parameters
		// For example: "userFills:0x123" or "orderbook:BTC"
		parts := strings.Split(channel, ":")
		if len(parts) > 1 {
			activeSubscriptions[channel]["type"] = parts[0]
			activeSubscriptions[channel]["params"] = parts[1:]
		}
	}
	ws.mu.RUnlock()

	if err := ws.Connect(); err != nil {
		log.Printf("Reconnection failed: %v", err)
	} else {
		log.Println("Reconnected successfully")

		// Resubscribe to all active subscriptions
		for channel, params := range activeSubscriptions {
			if subType, ok := params["type"].(string); ok {
				subParams := make(map[string]interface{})
				if paramList, ok := params["params"].([]string); ok {
					for i, param := range paramList {
						subParams[fmt.Sprintf("param%d", i)] = param
					}
				}

				// Resubscribe using the appropriate method based on subscription type
				switch subType {
				case "userFills":
					if len(subParams) > 0 {
						ws.SubscribeUserFills(subParams["param0"].(string), func(data interface{}) {
							if ch, exists := ws.subscriptions[channel]; exists {
								ch <- data
							}
						})
					}
				case "orderbook":
					if len(subParams) > 0 {
						ws.SubscribeOrderbook(subParams["param0"].(string), func(data interface{}) {
							if ch, exists := ws.subscriptions[channel]; exists {
								ch <- data
							}
						})
					}
				case "trades":
					if len(subParams) > 0 {
						ws.SubscribeTrades(subParams["param0"].(string), func(data interface{}) {
							if ch, exists := ws.subscriptions[channel]; exists {
								ch <- data
							}
						})
					}
				case "userEvents":
					if len(subParams) > 0 {
						ws.SubscribeUserEvents(subParams["param0"].(string), func(data interface{}) {
							if ch, exists := ws.subscriptions[channel]; exists {
								ch <- data
							}
						})
					}
				case "orderUpdates":
					if len(subParams) > 0 {
						ws.SubscribeOrderUpdates(subParams["param0"].(string), func(data interface{}) {
							if ch, exists := ws.subscriptions[channel]; exists {
								ch <- data
							}
						})
					}
				case "notification":
					if len(subParams) > 0 {
						ws.SubscribeNotification(subParams["param0"].(string), func(data interface{}) {
							if ch, exists := ws.subscriptions[channel]; exists {
								ch <- data
							}
						})
					}
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
	channel := "l2Book"

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
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
	channel := "trades"

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
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
	channel := "userFills"

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
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
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("webData2", map[string]interface{}{"user": user})
}
