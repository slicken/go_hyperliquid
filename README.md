# go-hyperliquid - fork of "githib.com/Logarithm-Labs/go-hyperliquid/hyperliquid"
A golang SDK for Hyperliquid API with **automatic WebSocket fallback** for optimal performance.

# API reference
- [Hyperliquid](https://app.hyperliquid.xyz/)
- [Hyperliquid API docs](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api)
- [Hyperliquid official Python SDK](https://github.com/hyperliquid-dex/hyperliquid-python-sdk)

# How to install?
```bash
go get github.com/slicken/go_hyperliquid
```

# Documentation

[![GoDoc](https://godoc.org/github.com/adshao/go-binance?status.svg)](https://pkg.go.dev/github.com/Logarithm-Labs/go-hyperliquid/hyperliquid#section-documentation)

# Quick start

## Basic Usage
```go
package main

import (
	"log"

	"github.com/slicken/go_hyperliquid"
)

func main() {
	hyperliquidClient := hyperliquid.NewHyperliquid(&hyperliquid.HyperliquidClientConfig{
		IsMainnet:      true,
		AccountAddress: "0x12345",   // Main address of the Hyperliquid account that you want to use
		PrivateKey:     "abc1234",   // Private key of the account or API private key from Hyperliquid
	})

	// Get balances
	res, err := hyperliquidClient.GetAccountState()
	if err != nil {
		log.Print(err)
	}
	log.Printf("GetAccountState(): %+v", res)
}
```

## WebSocket Fallback Feature

The SDK includes **automatic WebSocket fallback** that optimizes performance by using the most efficient transport method for each request type:

### How It Works

1. **Info Requests** (meta, account state, etc.): Use WebSocket when connected
2. **Exchange Requests** (orders, cancels, modifies): Use WebSocket when connected  
3. **Automatic Fallback**: HTTP if WebSocket fails or is not connected
4. **Zero Code Changes**: All existing API calls work seamlessly

### Example with WebSocket Fallback
```go
package main

import (
	"log"
	"os"
	"strings"

	"github.com/slicken/go_hyperliquid"
)

func main() {
	// Initialize client with automatic WebSocket fallback
	config := &hyperliquid.HyperliquidClientConfig{
		IsMainnet:      true,
		AccountAddress: strings.ToLower(os.Getenv("HYPERLIQUID_API_KEY")),    // Main account address
		PrivateKey:     strings.ToLower(os.Getenv("HYPERLIQUID_API_SECRET")), // API wallet private key
	}
	hl := hyperliquid.NewHyperliquid(config)

	// Connect to WebSocket (optional - will auto-fallback to HTTP if not connected)
	err := hl.WebSocketAPI.Connect()
	if err != nil {
		log.Printf("WebSocket connection failed, will use HTTP: %v", err)
	}

	// These automatically use WebSocket if connected, HTTP if not
	meta, err := hl.InfoAPI.BuildMetaMap()           // WebSocket for info
	orderResponse, err := hl.ExchangeAPI.Order(...)  // WebSocket for orders
	
	// All existing functions work the same way - no code changes needed!
}
```

### Benefits

- **🚀 Faster Info Requests**: WebSocket reduces latency for market data
- **⚡ Real-time Orders**: WebSocket provides faster order execution
- **🔄 Automatic Fallback**: HTTP if WebSocket fails
- **🔧 Zero Code Changes**: All existing API calls work seamlessly
- **📊 Better Performance**: Optimized transport for each request type

### Transport Selection Logic

| Request Type | When WebSocket Connected | When WebSocket Disconnected |
|--------------|-------------------------|----------------------------|
| Info Requests | ✅ WebSocket | ✅ HTTP |
| Exchange Requests | ✅ WebSocket | ✅ HTTP |
| WebSocket Subscriptions | ✅ WebSocket 

## WebSocket Examples

### Orderbook and Trades
```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/slicken/go_hyperliquid"
)

func main() {
	// Initialize SDK
	config := &hyperliquid.HyperliquidClientConfig{
		IsMainnet:      true,
		AccountAddress: "0x12345",   // Main address of the Hyperliquid account that you want to use
		PrivateKey:     "abc1234",   // Private key of the account or API private key from Hyperliquid
	}

	hl := hyperliquid.NewHyperliquid(config)
	if hl == nil {
		log.Fatal("Failed to initialize SDK")
	}

	// Connect to WebSocket
	err := hl.WebSocketAPI.Connect()
	if err != nil {
		log.Fatal("Failed to connect to WebSocket:", err)
	}
	defer hl.WebSocketAPI.Disconnect()

	// Subscribe to BTC orderbook updates
	if err := hl.WebSocketAPI.SubscribeOrderbook("BTC", func(data interface{}) {
		orderbook, ok := data.(map[string]interface{})
		if !ok {
			return
		}

		levels, ok := orderbook["levels"].([]interface{})
		if !ok || len(levels) != 2 {
			return
		}

		// Get bids and asks
		bids, ok := levels[0].([]interface{})
		asks, ok := levels[1].([]interface{})
		if !ok || len(bids) == 0 || len(asks) == 0 {
			return
		}

		// Print top 5 asks and bids
		fmt.Println("\nAsks:")
		for i := 0; i < 5 && i < len(asks); i++ {
			ask := asks[i].(map[string]interface{})
			fmt.Printf("  %s: %s\n", ask["px"], ask["sz"])
		}

		fmt.Println("\nBids:")
		for i := 0; i < 5 && i < len(bids); i++ {
			bid := bids[i].(map[string]interface{})
			fmt.Printf("  %s: %s\n", bid["px"], bid["sz"])
		}
	}); err != nil {
		log.Printf("Failed to subscribe to orderbook: %v", err)
	}

	// Subscribe to BTC trades
	if err := hl.WebSocketAPI.SubscribeTrades("BTC", func(data interface{}) {
		trades, ok := data.([]interface{})
		if !ok || len(trades) == 0 {
			return
		}

		// Process latest trade
		trade := trades[len(trades)-1].(map[string]interface{})
		fmt.Printf("\nTrade: %s %s @ %s\n",
			trade["side"], trade["sz"], trade["px"])
	}); err != nil {
		log.Printf("Failed to subscribe to trades: %v", err)
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
```

### Market Order Example
```go
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/slicken/go_hyperliquid"
)

func main() {
	// Initialize SDK
	config := &hyperliquid.HyperliquidClientConfig{
		IsMainnet:      true,
		AccountAddress: "0x12345",   // Main address of the Hyperliquid account that you want to use
		PrivateKey:     "abc1234",   // Private key of the account or API private key from Hyperliquid
	}

	hl := hyperliquid.NewHyperliquid(config)
	if hl == nil {
		log.Fatal("Failed to initialize SDK")
	}

	// Place a market buy order for BTC
	coin := "BTC"
	size := 0.01      // Size in BTC
	slippage := 0.005 // 0.5% slippage

	response, err := hl.ExchangeAPI.MarketOrder(coin, size, &slippage)
	if err != nil {
		log.Fatalf("Failed to place order: %v", err)
	}

	fmt.Printf("Order placed successfully!\n")
	fmt.Printf("Status: %s\n", response.Status)

	// Get account information
	accountState, err := hl.InfoAPI.GetUserState(hl.AccountAddress())
	if err != nil {
		log.Printf("Failed to get account state: %v", err)
	} else {
		fmt.Printf("\nAccount Information:\n")
		fmt.Printf("  Account Value: %f\n", accountState.MarginSummary.AccountValue)
		fmt.Printf("  Total Position Value: %f\n", accountState.MarginSummary.TotalNtlPos)
		fmt.Printf("  Free Collateral: %f\n", accountState.Withdrawable)
	}
}
```

## Available Features

### WebSocket Fallback
- **Automatic Transport Selection**: WebSocket for info/exchange requests when connected
- **Seamless Fallback**: HTTP when WebSocket fails or is disconnected
- **Zero Code Changes**: All existing API calls work automatically
- **Performance Optimized**: Fastest transport for each request type

### WebSocket Subscriptions
- Real-time orderbook updates
- Real-time trade updates
- User fill updates
- Mid price updates
- Automatic reconnection
- Ping/pong heartbeat

### Trading
- Market orders
- Limit orders
- Cancel orders
- Modify orders
- Close positions
- Update leverage

### Account Management
- Get account state
- Get open orders
- Get order history
- Get fill history
- Get funding history
- Withdraw funds

### Market Data
- Get orderbook snapshots
- Get candle data
- Get funding rates
- Get market prices
- Get asset information

## Error Handling
The SDK includes comprehensive error handling and logging capabilities. Enable debug mode for detailed logs:

```go
hl.ExchangeAPI.SetDebug(true)
hl.InfoAPI.SetDebug(true)
hl.WebSocketAPI.SetDebug(true)
```

## Rate Limiting
The SDK automatically handles rate limiting according to Hyperliquid's API limits. You can check your current rate limit status:

```go
limits, err := hl.InfoAPI.GetAccountRateLimits()
if err != nil {
    log.Printf("Failed to get rate limits: %v", err)
} else {
    fmt.Printf("Requests used: %d/%d\n", limits.NRequestsUsed, limits.NRequestsCap)
}
```
