package hyperliquid

import (
	"testing"
	"time"
)

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
