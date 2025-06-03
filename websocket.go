package hyperliquid

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
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

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	ws.conn = conn
	ws.isConnected = true
	ws.reconnectCount = 0

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
func (ws *WebSocketClient) SetDebugActive() {
	ws.debug = true
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
			if ws.reconnectCount < ws.maxReconnects {
				ws.reconnect()
			}
			return
		}

		var response WSResponse
		if err := json.Unmarshal(message, &response); err != nil {
			continue
		}

		ws.mu.RLock()
		if ch, exists := ws.subscriptions[response.Channel]; exists {
			select {
			case ch <- response.Data:
			default:
				// Channel full, skip message
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

	if err := ws.Connect(); err != nil {
		log.Printf("Reconnection failed: %v", err)
	} else {
		log.Println("Reconnected successfully")
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
	channel := fmt.Sprintf("userFills:%s", user)

	ws.mu.Lock()
	ch := make(chan interface{}, 100)
	ws.subscriptions[channel] = ch
	ws.mu.Unlock()

	go func() {
		for data := range ch {
			handler(data)
		}
	}()

	return ws.subscribe("userFills", map[string]interface{}{"user": user})
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
