package hyperliquid

import (
	"testing"
	"time"
)

func TestLatencyMeasurement(t *testing.T) {
	// Create a new WebSocket client
	ws := NewWebSocketClient(true)

	// Initially, latency should be 0
	latency := ws.GetLatency()
	if latency != 0 {
		t.Errorf("Expected initial latency to be 0, got %d", latency)
	}

	// Set a mock ping time
	ws.mu.Lock()
	ws.lastPingTime = time.Now().Add(-50 * time.Millisecond) // 50ms ago
	ws.mu.Unlock()

	// Simulate receiving a pong response
	// This would normally happen in the readMessages goroutine
	ws.mu.Lock()
	latency = time.Since(ws.lastPingTime).Milliseconds()
	ws.latencyMs = latency
	ws.mu.Unlock()

	// Check that latency is now set
	latency = ws.GetLatency()
	if latency <= 0 {
		t.Errorf("Expected latency to be > 0 after pong, got %d", latency)
	}

	// Latency should be around 50ms (with some tolerance for test execution time)
	if latency < 40 || latency > 100 {
		t.Errorf("Expected latency to be around 50ms, got %d", latency)
	}
}

func TestWebSocket_ConnectAndSubscribe(t *testing.T) {
	hl := GetHyperliquidAPI()
	if hl == nil {
		t.Skip("Skipping test: API not configured")
	}

	// Connect to WebSocket
	err := hl.ConnectWebSocket()
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer hl.DisconnectWebSocket()

	// Test orderbook subscription
	orderbookReceived := make(chan bool)
	err = hl.SubscribeOrderbook("ETH", func(data interface{}) {
		orderbookReceived <- true
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to orderbook: %v", err)
	}

	// Wait for orderbook data
	select {
	case <-orderbookReceived:
		t.Log("Received orderbook data")
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for orderbook data")
	}

	// Test trades subscription
	tradesReceived := make(chan bool)
	err = hl.SubscribeTrades("ETH", func(data interface{}) {
		tradesReceived <- true
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to trades: %v", err)
	}

	// Wait for trades data
	select {
	case <-tradesReceived:
		t.Log("Received trades data")
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for trades data")
	}

	// Test all mids subscription
	midsReceived := make(chan bool)
	err = hl.SubscribeAllMids(func(data interface{}) {
		midsReceived <- true
	})
	if err != nil {
		t.Fatalf("Failed to subscribe to all mids: %v", err)
	}

	// Wait for mids data
	select {
	case <-midsReceived:
		t.Log("Received mids data")
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for mids data")
	}

	// Test user fills subscription (only if we have an account address)
	if hl.AccountAddress() != "" {
		fillsReceived := make(chan bool)
		err = hl.SubscribeUserFills(hl.AccountAddress(), func(data interface{}) {
			fillsReceived <- true
		})
		if err != nil {
			t.Fatalf("Failed to subscribe to user fills: %v", err)
		}

		// Wait for fills data
		select {
		case <-fillsReceived:
			t.Log("Received user fills data")
		case <-time.After(5 * time.Second):
			t.Error("Timeout waiting for user fills data")
		}
	}
}
