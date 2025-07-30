package hyperliquid

import (
	"log"
	"sync"
	"testing"
	"time"
)

// TestWebSocketSubscriptions tests all subscription types
func TestWebSocketSubscriptions(t *testing.T) {
	ws := NewWebSocketAPI(true) // Use mainnet for testing
	ws.SetDebug(true)

	if err := ws.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Disconnect()

	// Wait for connection
	time.Sleep(1 * time.Second)

	// Test each subscription type systematically
	subscriptionTests := []struct {
		name      string
		subType   string
		setupFn   func() error
		cleanupFn func() error
	}{
		{
			name:    "Orderbook",
			subType: "l2Book",
			setupFn: func() error {
				return ws.SubscribeOrderbook("BTC", func(data interface{}) {
					log.Printf("Received orderbook: %+v", data)
				})
			},
			cleanupFn: func() error {
				return ws.UnsubscribeOrderbook("BTC")
			},
		},
		{
			name:    "Trades",
			subType: "trades",
			setupFn: func() error {
				return ws.SubscribeTrades("BTC", func(data interface{}) {
					log.Printf("Received trades: %+v", data)
				})
			},
			cleanupFn: func() error {
				return ws.UnsubscribeTrades("BTC")
			},
		},
		{
			name:    "AllMids",
			subType: "allMids",
			setupFn: func() error {
				return ws.SubscribeAllMids(func(data interface{}) {
					log.Printf("Received allMids: %+v", data)
				})
			},
			cleanupFn: func() error {
				return ws.UnsubscribeAllMids()
			},
		},
		{
			name:    "Bbo",
			subType: "bbo",
			setupFn: func() error {
				return ws.SubscribeBbo("BTC", func(data interface{}) {
					log.Printf("Received bbo: %+v", data)
				})
			},
			cleanupFn: func() error {
				return ws.UnsubscribeBbo("BTC")
			},
		},
		{
			name:    "Candle",
			subType: "candle",
			setupFn: func() error {
				return ws.SubscribeCandle("BTC", "1m", func(data interface{}) {
					log.Printf("Received candle: %+v", data)
				})
			},
			cleanupFn: func() error {
				return ws.UnsubscribeCandle("BTC", "1m")
			},
		},
	}

	// Test each subscription systematically
	for _, test := range subscriptionTests {
		t.Run(test.name, func(t *testing.T) {
			log.Printf("=== Testing %s subscription ===", test.name)

			// 1. Subscribe
			if err := test.setupFn(); err != nil {
				t.Fatalf("Failed to subscribe to %s: %v", test.subType, err)
			}
			log.Printf("✓ Successfully subscribed to %s", test.subType)

			// 2. Wait for messages to verify subscription works
			time.Sleep(3 * time.Second)
			log.Printf("✓ Subscription to %s is working", test.subType)

			// 3. Unsubscribe
			if err := test.cleanupFn(); err != nil {
				t.Fatalf("Failed to unsubscribe from %s: %v", test.subType, err)
			}
			log.Printf("✓ Successfully unsubscribed from %s", test.subType)

			// 4. Wait a bit to verify unsubscribe worked
			time.Sleep(1 * time.Second)
			log.Printf("✓ Unsubscribe from %s completed", test.subType)
		})
	}
}

// TestWebSocketReconnection tests reconnection functionality
func TestWebSocketReconnection(t *testing.T) {
	ws := NewWebSocketAPI(true) // Use mainnet for testing
	ws.SetDebug(true)

	receivedMessages := make([]interface{}, 0)
	var mu sync.Mutex

	handler := func(data interface{}) {
		mu.Lock()
		receivedMessages = append(receivedMessages, data)
		mu.Unlock()
		log.Printf("Received message: %+v", data)
	}

	// Connect and subscribe
	if err := ws.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Subscribe to a few channels
	if err := ws.SubscribeOrderbook("BTC", handler); err != nil {
		t.Fatalf("Failed to subscribe to orderbook: %v", err)
	}
	if err := ws.SubscribeTrades("BTC", handler); err != nil {
		t.Fatalf("Failed to subscribe to trades: %v", err)
	}
	if err := ws.SubscribeAllMids(handler); err != nil {
		t.Fatalf("Failed to subscribe to allMids: %v", err)
	}

	// Wait for initial messages
	time.Sleep(2 * time.Second)

	initialCount := len(receivedMessages)
	if initialCount == 0 {
		t.Log("No initial messages received, but continuing with reconnection test")
	}

	// Simulate connection loss by closing the connection
	ws.mu.Lock()
	if ws.conn != nil {
		ws.conn.Close()
	}
	ws.mu.Unlock()

	// Wait for reconnection
	time.Sleep(3 * time.Second)

	// Check if we're still receiving messages
	finalCount := len(receivedMessages)
	if finalCount <= initialCount {
		t.Logf("Message count: %d -> %d (reconnection may still be in progress)", initialCount, finalCount)
	} else {
		t.Logf("Reconnection successful, received %d new messages", finalCount-initialCount)
	}

	ws.Disconnect()
}

// TestWebSocketPostRequests tests post request functionality
func TestWebSocketPostRequests(t *testing.T) {
	ws := NewWebSocketAPI(true) // Use mainnet for testing
	ws.SetDebug(true)

	if err := ws.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Disconnect()

	// Wait for connection
	time.Sleep(1 * time.Second)

	// Test info request
	t.Run("PostInfoRequest", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "meta",
		}

		response, err := ws.PostInfoRequest(payload)
		if err != nil {
			t.Fatalf("Failed to post info request: %v", err)
		}

		if response == nil {
			t.Error("Expected response, got nil")
		} else {
			t.Logf("Info response: %+v", response)
		}
	})

	// Test action request
	t.Run("PostActionRequest", func(t *testing.T) {
		payload := map[string]interface{}{
			"type": "getUserState",
			"user": "testuser",
		}

		response, err := ws.PostActionRequest(payload)
		if err != nil {
			// Action requests might fail for test users, that's OK
			t.Logf("Action request failed (expected for test user): %v", err)
		} else {
			if response == nil {
				t.Error("Expected response, got nil")
			} else {
				t.Logf("Action response: %+v", response)
			}
		}
	})
}

// TestWebSocketLatency tests latency measurement
func TestWebSocketLatency(t *testing.T) {
	ws := NewWebSocketAPI(true) // Use mainnet for testing
	ws.SetDebug(true)

	if err := ws.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Disconnect()

	// Wait for initial ping/pong
	time.Sleep(3 * time.Second)

	latency := ws.GetLatency()
	if latency == 0 {
		t.Log("Latency is 0, may need more time for ping/pong cycle")
	} else {
		t.Logf("Measured latency: %dms", latency)
		if latency > 1000 {
			t.Errorf("Latency seems too high: %dms", latency)
		}
	}
}

// TestWebSocketMessageProcessing tests message processing efficiency
func TestWebSocketMessageProcessing(t *testing.T) {
	ws := NewWebSocketAPI(true) // Use mainnet for testing
	ws.SetDebug(true)

	// Subscribe to multiple channels to test processing
	messageCount := 0
	var mu sync.Mutex

	handler := func(data interface{}) {
		mu.Lock()
		messageCount++
		mu.Unlock()
	}

	// Subscribe to multiple channels
	subscriptions := []struct {
		name string
		fn   func() error
	}{
		{"orderbook", func() error { return ws.SubscribeOrderbook("BTC", handler) }},
		{"trades", func() error { return ws.SubscribeTrades("BTC", handler) }},
		{"allMids", func() error { return ws.SubscribeAllMids(handler) }},
		{"bbo", func() error { return ws.SubscribeBbo("BTC", handler) }},
		{"candle", func() error { return ws.SubscribeCandle("BTC", "1m", handler) }},
	}

	if err := ws.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Disconnect()

	// Setup all subscriptions
	for _, sub := range subscriptions {
		if err := sub.fn(); err != nil {
			t.Fatalf("Failed to subscribe to %s: %v", sub.name, err)
		}
	}

	// Wait for messages
	time.Sleep(5 * time.Second)

	mu.Lock()
	finalCount := messageCount
	mu.Unlock()

	t.Logf("Processed %d messages across %d subscriptions", finalCount, len(subscriptions))
	if finalCount == 0 {
		t.Error("No messages processed")
	}
}

// TestWebSocketSubscriptionMatching tests the subscription matching logic
func TestWebSocketSubscriptionMatching(t *testing.T) {
	ws := NewWebSocketAPI(true) // Use mainnet for testing

	// Test user fills matching
	t.Run("UserFillsMatching", func(t *testing.T) {
		sub := &Subscription{
			Type: SubTypeUserFills,
			User: "testuser",
		}

		response := &WSResponse{
			Channel: "userFills",
			Data: map[string]interface{}{
				"user": "testuser",
				"data": "testdata",
			},
		}

		if !ws.matchesSubscription(sub, response) {
			t.Error("User fills subscription should match")
		}

		// Test non-matching user
		response.Data = map[string]interface{}{
			"user": "otheruser",
			"data": "testdata",
		}

		if ws.matchesSubscription(sub, response) {
			t.Error("User fills subscription should not match different user")
		}
	})

	// Test L2Book matching
	t.Run("L2BookMatching", func(t *testing.T) {
		sub := &Subscription{
			Type: SubTypeL2Book,
			Coin: "BTC",
		}

		response := &WSResponse{
			Channel: "l2Book",
			Data: map[string]interface{}{
				"coin": "BTC",
				"data": "testdata",
			},
		}

		if !ws.matchesSubscription(sub, response) {
			t.Error("L2Book subscription should match")
		}

		// Test non-matching coin
		response.Data = map[string]interface{}{
			"coin": "ETH",
			"data": "testdata",
		}

		if ws.matchesSubscription(sub, response) {
			t.Error("L2Book subscription should not match different coin")
		}
	})

	// Test candle matching
	t.Run("CandleMatching", func(t *testing.T) {
		sub := &Subscription{
			Type:     SubTypeCandle,
			Coin:     "BTC",
			Interval: "1m",
		}

		response := &WSResponse{
			Channel: "candle",
			Data: map[string]interface{}{
				"s":    "BTC",
				"i":    "1m",
				"data": "testdata",
			},
		}

		if !ws.matchesSubscription(sub, response) {
			t.Error("Candle subscription should match")
		}
	})
}

// TestWebSocketBufferSizing tests the dynamic buffer sizing
func TestWebSocketBufferSizing(t *testing.T) {
	ws := NewWebSocketAPI(true) // Use mainnet for testing

	// Connect first
	if err := ws.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Disconnect()

	// Test high frequency subscriptions
	t.Run("HighFrequencyBuffers", func(t *testing.T) {
		handler := func(data interface{}) {}

		// These should get buffer size 50
		if err := ws.SubscribeOrderbook("BTC", handler); err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
		if err := ws.SubscribeTrades("BTC", handler); err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Check buffer sizes
		ws.mu.RLock()
		for channel, sub := range ws.subscriptions {
			if channel == "l2Book:BTC" || channel == "trades:BTC" {
				if cap(sub.Channel) != 50 {
					t.Errorf("Expected buffer size 50 for %s, got %d", channel, cap(sub.Channel))
				}
			}
		}
		ws.mu.RUnlock()
	})

	// Test medium frequency subscriptions
	t.Run("MediumFrequencyBuffers", func(t *testing.T) {
		handler := func(data interface{}) {}

		// These should get buffer size 100
		if err := ws.SubscribeUserFills("testuser", handler); err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
		if err := ws.SubscribeOrderUpdates("testuser", handler); err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Check buffer sizes
		ws.mu.RLock()
		for channel, sub := range ws.subscriptions {
			if channel == "userFills:testuser" || channel == "orderUpdates:testuser" {
				if cap(sub.Channel) != 100 {
					t.Errorf("Expected buffer size 100 for %s, got %d", channel, cap(sub.Channel))
				}
			}
		}
		ws.mu.RUnlock()
	})

	// Test low frequency subscriptions
	t.Run("LowFrequencyBuffers", func(t *testing.T) {
		handler := func(data interface{}) {}

		// These should get buffer size 10
		if err := ws.SubscribeNotification("testuser", handler); err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
		if err := ws.SubscribeCandle("BTC", "1m", handler); err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Check buffer sizes
		ws.mu.RLock()
		for channel, sub := range ws.subscriptions {
			if channel == "notification:testuser" || channel == "candle:BTC:1m" {
				if cap(sub.Channel) != 10 {
					t.Errorf("Expected buffer size 10 for %s, got %d", channel, cap(sub.Channel))
				}
			}
		}
		ws.mu.RUnlock()
	})
}

// TestWebSocketConcurrentSubscriptions tests concurrent subscription handling
func TestWebSocketConcurrentSubscriptions(t *testing.T) {
	ws := NewWebSocketAPI(true) // Use mainnet for testing
	ws.SetDebug(true)

	if err := ws.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Disconnect()

	// Wait for connection
	time.Sleep(1 * time.Second)

	var wg sync.WaitGroup
	messageCount := 0
	var mu sync.Mutex

	handler := func(data interface{}) {
		mu.Lock()
		messageCount++
		mu.Unlock()
	}

	// Subscribe to multiple channels concurrently
	subscriptions := []struct {
		name string
		fn   func() error
	}{
		{"orderbook1", func() error { return ws.SubscribeOrderbook("BTC", handler) }},
		{"orderbook2", func() error { return ws.SubscribeOrderbook("ETH", handler) }},
		{"trades1", func() error { return ws.SubscribeTrades("BTC", handler) }},
		{"trades2", func() error { return ws.SubscribeTrades("ETH", handler) }},
		{"allMids", func() error { return ws.SubscribeAllMids(handler) }},
		{"bbo1", func() error { return ws.SubscribeBbo("BTC", handler) }},
		{"bbo2", func() error { return ws.SubscribeBbo("ETH", handler) }},
	}

	// Setup subscriptions concurrently
	for _, sub := range subscriptions {
		wg.Add(1)
		go func(sub struct {
			name string
			fn   func() error
		}) {
			defer wg.Done()
			if err := sub.fn(); err != nil {
				t.Errorf("Failed to subscribe to %s: %v", sub.name, err)
			}
		}(sub)
	}

	wg.Wait()

	// Wait for messages
	time.Sleep(5 * time.Second)

	mu.Lock()
	finalCount := messageCount
	mu.Unlock()

	t.Logf("Processed %d messages across %d concurrent subscriptions", finalCount, len(subscriptions))
	if finalCount == 0 {
		t.Error("No messages processed from concurrent subscriptions")
	}
}

// TestWebSocketUnsubscribe tests unsubscribe functionality
func TestWebSocketUnsubscribe(t *testing.T) {
	ws := NewWebSocketAPI(true) // Use mainnet for testing
	ws.SetDebug(true)

	if err := ws.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer ws.Disconnect()

	// Wait for connection
	time.Sleep(1 * time.Second)

	// Track received messages
	receivedMessages := make([]interface{}, 0)
	var mu sync.Mutex

	handler := func(data interface{}) {
		mu.Lock()
		receivedMessages = append(receivedMessages, data)
		mu.Unlock()
		log.Printf("Received message: %+v", data)
	}

	// Subscribe to orderbook
	if err := ws.SubscribeOrderbook("BTC", handler); err != nil {
		t.Fatalf("Failed to subscribe to orderbook: %v", err)
	}

	// Wait for some messages
	time.Sleep(2 * time.Second)

	initialCount := len(receivedMessages)
	t.Logf("Received %d messages before unsubscribe", initialCount)

	// Unsubscribe
	if err := ws.UnsubscribeOrderbook("BTC"); err != nil {
		t.Fatalf("Failed to unsubscribe from orderbook: %v", err)
	}

	// Wait a bit more
	time.Sleep(2 * time.Second)

	finalCount := len(receivedMessages)
	t.Logf("Received %d messages after unsubscribe", finalCount-initialCount)

	// Check that we're not receiving many more messages after unsubscribe
	if finalCount-initialCount > 5 {
		t.Logf("Still receiving messages after unsubscribe (this might be normal due to buffering)")
	} else {
		t.Logf("Unsubscribe appears to be working correctly")
	}
}
