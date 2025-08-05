package hyperliquid

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
)

// TestWebSocketConnection tests basic WebSocket connection functionality
func TestWebSocketConnection(t *testing.T) {
	ws := NewWebSocketAPI(true) // Use mainnet for testing
	ws.SetDebug(true)

	// Test initial connection
	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait a moment for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	// Test connection status
	if !ws.IsConnected() {
		t.Error("WebSocket should be connected")
	}

	// Test disconnect
	err = ws.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	// Test connection status after disconnect
	if ws.IsConnected() {
		t.Error("WebSocket should not be connected after disconnect")
	}
}

// TestWebSocketReconnection tests automatic reconnection functionality
func TestWebSocketReconnection(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	// Connect initially
	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	// Test reconnection by disconnecting for testing
	err = ws.DisconnectForTesting()
	if err != nil {
		t.Fatalf("Failed to disconnect for testing: %v", err)
	}

	// Wait for reconnection to happen
	time.Sleep(5 * time.Second)

	// Check if reconnected
	if !ws.IsConnected() {
		t.Error("WebSocket should have reconnected automatically")
	}

	// Clean up
	ws.Disconnect()
}

// TestSubscribeOrderbook tests orderbook subscription
func TestSubscribeOrderbook(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received orderbook data: %+v", data)
		}
	}

	// Subscribe to orderbook for BTC
	err = ws.SubscribeOrderbook("BTC", handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to orderbook: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No orderbook data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeOrderbook("BTC")
	if err != nil {
		t.Fatalf("Failed to unsubscribe from orderbook: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeTrades tests trades subscription
func TestSubscribeTrades(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received trades data: %+v", data)
		}
	}

	// Subscribe to trades for BTC
	err = ws.SubscribeTrades("BTC", handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to trades: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No trades data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeTrades("BTC")
	if err != nil {
		t.Fatalf("Failed to unsubscribe from trades: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeUserFills tests user fills subscription
func TestSubscribeUserFills(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received user fills data: %+v", data)
		}
	}

	// Subscribe to user fills (using a test address)
	testAddress := "0x1234567890123456789012345678901234567890"
	err = ws.SubscribeUserFills(testAddress, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to user fills: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No user fills data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeUserFills(testAddress)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from user fills: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeAllMids tests all mids subscription
func TestSubscribeAllMids(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received all mids data: %+v", data)
		}
	}

	// Subscribe to all mids
	err = ws.SubscribeAllMids(handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to all mids: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No all mids data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeAllMids()
	if err != nil {
		t.Fatalf("Failed to unsubscribe from all mids: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeUserEvents tests user events subscription
func TestSubscribeUserEvents(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received user events data: %+v", data)
		}
	}

	// Subscribe to user events
	testAddress := "0x1234567890123456789012345678901234567890"
	err = ws.SubscribeUserEvents(testAddress, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to user events: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No user events data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeUserEvents(testAddress)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from user events: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeUserFundings tests user fundings subscription
func TestSubscribeUserFundings(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received user fundings data: %+v", data)
		}
	}

	// Subscribe to user fundings
	testAddress := "0x1234567890123456789012345678901234567890"
	err = ws.SubscribeUserFundings(testAddress, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to user fundings: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No user fundings data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeUserFundings(testAddress)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from user fundings: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeUserNonFundingLedgerUpdates tests user non-funding ledger updates subscription
func TestSubscribeUserNonFundingLedgerUpdates(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received user non-funding ledger updates data: %+v", data)
		}
	}

	// Subscribe to user non-funding ledger updates
	testAddress := "0x1234567890123456789012345678901234567890"
	err = ws.SubscribeUserNonFundingLedgerUpdates(testAddress, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to user non-funding ledger updates: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No user non-funding ledger updates data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeUserNonFundingLedgerUpdates(testAddress)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from user non-funding ledger updates: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeUserTwapSliceFills tests user TWAP slice fills subscription
func TestSubscribeUserTwapSliceFills(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received user TWAP slice fills data: %+v", data)
		}
	}

	// Subscribe to user TWAP slice fills
	testAddress := "0x1234567890123456789012345678901234567890"
	err = ws.SubscribeUserTwapSliceFills(testAddress, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to user TWAP slice fills: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No user TWAP slice fills data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeUserTwapSliceFills(testAddress)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from user TWAP slice fills: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeUserTwapHistory tests user TWAP history subscription
func TestSubscribeUserTwapHistory(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received user TWAP history data: %+v", data)
		}
	}

	// Subscribe to user TWAP history
	testAddress := "0x1234567890123456789012345678901234567890"
	err = ws.SubscribeUserTwapHistory(testAddress, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to user TWAP history: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No user TWAP history data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeUserTwapHistory(testAddress)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from user TWAP history: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeActiveAssetCtx tests active asset context subscription
func TestSubscribeActiveAssetCtx(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received active asset context data: %+v", data)
		}
	}

	// Subscribe to active asset context for BTC
	err = ws.SubscribeActiveAssetCtx("BTC", handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to active asset context: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No active asset context data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeActiveAssetCtx("BTC")
	if err != nil {
		t.Fatalf("Failed to unsubscribe from active asset context: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeActiveAssetData tests active asset data subscription
func TestSubscribeActiveAssetData(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received active asset data: %+v", data)
		}
	}

	// Subscribe to active asset data
	testAddress := "0x1234567890123456789012345678901234567890"
	err = ws.SubscribeActiveAssetData(testAddress, "BTC", handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to active asset data: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No active asset data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeActiveAssetData(testAddress, "BTC")
	if err != nil {
		t.Fatalf("Failed to unsubscribe from active asset data: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeBbo tests best bid/offer subscription
func TestSubscribeBbo(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received BBO data: %+v", data)
		}
	}

	// Subscribe to BBO for BTC
	err = ws.SubscribeBbo("BTC", handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to BBO: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No BBO data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeBbo("BTC")
	if err != nil {
		t.Fatalf("Failed to unsubscribe from BBO: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeCandle tests candle subscription
func TestSubscribeCandle(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received candle data: %+v", data)
		}
	}

	// Subscribe to candle for BTC with 1m interval
	err = ws.SubscribeCandle("BTC", "1m", handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to candle: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No candle data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeCandle("BTC", "1m")
	if err != nil {
		t.Fatalf("Failed to unsubscribe from candle: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeOrderUpdates tests order updates subscription
func TestSubscribeOrderUpdates(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received order updates data: %+v", data)
		}
	}

	// Subscribe to order updates
	testAddress := "0x1234567890123456789012345678901234567890"
	err = ws.SubscribeOrderUpdates(testAddress, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to order updates: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No order updates data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeOrderUpdates(testAddress)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from order updates: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeNotification tests notification subscription
func TestSubscribeNotification(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received notification data: %+v", data)
		}
	}

	// Subscribe to notifications
	testAddress := "0x1234567890123456789012345678901234567890"
	err = ws.SubscribeNotification(testAddress, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to notifications: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No notification data received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeNotification(testAddress)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from notifications: %v", err)
	}

	ws.Disconnect()
}

// TestSubscribeWebData2 tests web data 2 subscription
func TestSubscribeWebData2(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received web data 2: %+v", data)
		}
	}

	// Subscribe to web data 2
	testAddress := "0x1234567890123456789012345678901234567890"
	err = ws.SubscribeWebData2(testAddress, handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to web data 2: %v", err)
	}

	// Wait for some data
	time.Sleep(2 * time.Second)

	// Check if we received data
	mu.Lock()
	if receivedData == nil {
		t.Error("No web data 2 received")
	}
	mu.Unlock()

	// Unsubscribe
	err = ws.UnsubscribeWebData2(testAddress)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from web data 2: %v", err)
	}

	ws.Disconnect()
}

// TestPostRequest tests post request functionality
func TestPostRequest(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	// Test post request
	payload := map[string]interface{}{
		"type": "test",
		"data": "test data",
	}

	response, err := ws.PostRequest("info", payload)
	if err != nil {
		t.Fatalf("Failed to send post request: %v", err)
	}

	if response == nil {
		t.Error("Expected response, got nil")
	}

	ws.Disconnect()
}

// TestPostInfoRequest tests info request functionality
func TestPostInfoRequest(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	// Test info request
	payload := map[string]interface{}{
		"type": "test",
		"data": "test data",
	}

	response, err := ws.PostInfoRequest(payload)
	if err != nil {
		t.Fatalf("Failed to send info request: %v", err)
	}

	if response == nil {
		t.Error("Expected response, got nil")
	}

	ws.Disconnect()
}

// TestPostActionRequest tests action request functionality
func TestPostActionRequest(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	// Test action request
	payload := map[string]interface{}{
		"type": "test",
		"data": "test data",
	}

	response, err := ws.PostActionRequest(payload)
	if err != nil {
		t.Fatalf("Failed to send action request: %v", err)
	}

	if response == nil {
		t.Error("Expected response, got nil")
	}

	ws.Disconnect()
}

// TestMultipleSubscriptions tests multiple subscriptions simultaneously
func TestMultipleSubscriptions(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData map[string]interface{}
	var mu sync.Mutex
	receivedData = make(map[string]interface{})

	// Create handlers for different subscription types
	orderbookHandler := func(data interface{}) {
		mu.Lock()
		receivedData["orderbook"] = data
		mu.Unlock()
	}

	tradesHandler := func(data interface{}) {
		mu.Lock()
		receivedData["trades"] = data
		mu.Unlock()
	}

	// Subscribe to multiple channels
	err = ws.SubscribeOrderbook("BTC", orderbookHandler)
	if err != nil {
		t.Fatalf("Failed to subscribe to orderbook: %v", err)
	}

	err = ws.SubscribeTrades("BTC", tradesHandler)
	if err != nil {
		t.Fatalf("Failed to subscribe to trades: %v", err)
	}

	// Wait for some data
	time.Sleep(3 * time.Second)

	// Check if we received data from both subscriptions
	mu.Lock()
	if receivedData["orderbook"] == nil {
		t.Error("No orderbook data received")
	}
	if receivedData["trades"] == nil {
		t.Error("No trades data received")
	}
	mu.Unlock()

	// Unsubscribe from both
	err = ws.UnsubscribeOrderbook("BTC")
	if err != nil {
		t.Fatalf("Failed to unsubscribe from orderbook: %v", err)
	}

	err = ws.UnsubscribeTrades("BTC")
	if err != nil {
		t.Fatalf("Failed to unsubscribe from trades: %v", err)
	}

	ws.Disconnect()
}

// TestWebSocketReconnectionWithSubscriptions tests reconnection with active subscriptions
func TestWebSocketReconnectionWithSubscriptions(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var receivedData interface{}
	var mu sync.Mutex
	handler := func(data interface{}) {
		mu.Lock()
		receivedData = data
		mu.Unlock()
		if ws.Debug {
			log.Printf("Received data after reconnection: %+v", data)
		}
	}

	// Subscribe to orderbook
	err = ws.SubscribeOrderbook("BTC", handler)
	if err != nil {
		t.Fatalf("Failed to subscribe to orderbook: %v", err)
	}

	// Wait for initial data
	time.Sleep(2 * time.Second)

	// Test reconnection by disconnecting for testing
	err = ws.DisconnectForTesting()
	if err != nil {
		t.Fatalf("Failed to disconnect for testing: %v", err)
	}

	// Wait for reconnection and resubscription
	time.Sleep(10 * time.Second)

	// Check if reconnected and receiving data
	if !ws.IsConnected() {
		t.Error("WebSocket should have reconnected")
	}

	// Wait for data after reconnection
	time.Sleep(2 * time.Second)

	mu.Lock()
	if receivedData == nil {
		t.Error("No data received after reconnection")
	}
	mu.Unlock()

	// Clean up
	ws.Disconnect()
}

// BenchmarkWebSocketConnection benchmarks connection performance
func BenchmarkWebSocketConnection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ws := NewWebSocketAPI(true)
		err := ws.Connect()
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}
		ws.Disconnect()
	}
}

// BenchmarkSubscription benchmarks subscription performance
func BenchmarkSubscription(b *testing.B) {
	ws := NewWebSocketAPI(true)
	err := ws.Connect()
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Disconnect()

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	handler := func(data interface{}) {
		// Benchmark handler - do nothing
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ws.SubscribeOrderbook("BTC", handler)
		if err != nil {
			b.Fatalf("Failed to subscribe: %v", err)
		}
		err = ws.UnsubscribeOrderbook("BTC")
		if err != nil {
			b.Fatalf("Failed to unsubscribe: %v", err)
		}
	}
}

// TestWebSocketDebugMode tests debug mode functionality
func TestWebSocketDebugMode(t *testing.T) {
	ws := NewWebSocketAPI(true)

	// Test debug mode setting
	ws.SetDebug(true)
	if !ws.Debug {
		t.Error("Debug mode should be enabled")
	}

	ws.SetDebug(false)
	if ws.Debug {
		t.Error("Debug mode should be disabled")
	}
}

// TestWebSocketURLs tests different URL configurations
func TestWebSocketURLs(t *testing.T) {
	// Test mainnet
	wsMainnet := NewWebSocketAPI(true)
	if wsMainnet.url != MainnetWSURL {
		t.Errorf("Expected mainnet URL %s, got %s", MainnetWSURL, wsMainnet.url)
	}

	// Test testnet
	wsTestnet := NewWebSocketAPI(false)
	if wsTestnet.url != TestnetWSURL {
		t.Errorf("Expected testnet URL %s, got %s", TestnetWSURL, wsTestnet.url)
	}
}

// TestWebSocketConcurrentAccess tests concurrent access to WebSocket
func TestWebSocketConcurrentAccess(t *testing.T) {
	ws := NewWebSocketAPI(true)
	ws.SetDebug(true)

	err := ws.Connect()
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Wait for connection to stabilize
	time.Sleep(100 * time.Millisecond)

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent subscriptions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			handler := func(data interface{}) {
				// Handle data
			}

			coin := fmt.Sprintf("COIN%d", id)
			err := ws.SubscribeOrderbook(coin, handler)
			if err != nil {
				t.Errorf("Failed to subscribe in goroutine %d: %v", id, err)
				return
			}

			// Wait a bit
			time.Sleep(100 * time.Millisecond)

			err = ws.UnsubscribeOrderbook(coin)
			if err != nil {
				t.Errorf("Failed to unsubscribe in goroutine %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	ws.Disconnect()
}
