# go-hyperliquid

A high-performance Go SDK for the Hyperliquid API, optimized for low-latency trading with automatic WebSocket fallback and atomic operations.

## Features

- **Ultra-Fast Performance** - Lock-free atomic operations and jsoniter for 5-10x faster JSON processing
- **Automatic WebSocket Fallback** - Uses WebSocket when connected, HTTP as fallback
- **100% Typed Data** - Strongly typed WebSocket subscriptions with compile-time safety
- **HFT Optimized** - Designed for high-frequency trading with minimal latency

## Installation

```bash
go get github.com/slicken/go_hyperliquid
```

## Quick Start

```go
package main

import (
	"log"
	"github.com/slicken/go_hyperliquid"
)

func main() {
	// Initialize client
	client := hyperliquid.NewHyperliquid(&hyperliquid.HyperliquidClientConfig{
		IsMainnet:      true,
		AccountAddress: "0x12345",
		PrivateKey:     "abc1234",
	})

	// Connect to WebSocket (optional - will auto-fallback to HTTP)
	client.WebSocketAPI.Connect()

	// Subscribe to real-time data with typed handlers
	client.WebSocketAPI.SubscribeOrderbook("BTC", func(data *hyperliquid.OrderbookData) {
		asks := data.GetAsks()
		bids := data.GetBids()
		log.Printf("Orderbook: %d asks, %d bids", len(asks), len(bids))
	})

	// Place orders (uses WebSocket if connected, HTTP as fallback)
	response, err := client.ExchangeAPI.MarketOrder("BTC", 0.01, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Order placed: %s", response.Status)
}
```

## WebSocket Subscriptions

All subscriptions use strongly typed data structures:

```go
// Orderbook with typed data
client.WebSocketAPI.SubscribeOrderbook("BTC", func(data *hyperliquid.OrderbookData) {
    asks := data.GetAsks()
    for _, ask := range asks {
        price, _ := ask.GetPrice()
        size, _ := ask.GetSize()
        log.Printf("Ask: %.2f @ %.6f", price, size)
    }
})

// Trades with typed data
client.WebSocketAPI.SubscribeTrades("BTC", func(data *hyperliquid.TradesData) {
     log.Printf("handle ws tradeData: %+v\m", data)
})

// All available subscriptions (replace "BTC" and "user" with your coin/user as needed)
client.WebSocketAPI.SubscribeAllMids(func(data *hyperliquid.AllMidsData) {})
client.WebSocketAPI.SubscribeBbo("BTC", func(data *hyperliquid.BboData) {})
client.WebSocketAPI.SubscribeCandle("BTC", "1m", func(data *hyperliquid.CandleData) {})
client.WebSocketAPI.SubscribeOrderbook("BTC", func(data *hyperliquid.OrderbookData) {})
client.WebSocketAPI.SubscribeTrades("BTC", func(data *hyperliquid.TradesData) {})
client.WebSocketAPI.SubscribeOrderUpdates("user", func(data *hyperliquid.OrderUpdatesData) {})
client.WebSocketAPI.SubscribeUserFills("user", func(data *hyperliquid.UserFillsData) {})
client.WebSocketAPI.SubscribeUserEvents("user", func(data *hyperliquid.UserEventsData) {})
client.WebSocketAPI.SubscribeUserFundings("user", func(data *hyperliquid.UserFundingsData) {})
client.WebSocketAPI.SubscribeUserNonFundingLedgerUpdates("user", func(data *hyperliquid.UserNonFundingLedgerUpdatesData) {})
client.WebSocketAPI.SubscribeUserTwapSliceFills("user", func(data *hyperliquid.UserTwapSliceFillsData) {})
client.WebSocketAPI.SubscribeUserTwapHistory("user", func(data *hyperliquid.UserTwapHistoryData) {})
client.WebSocketAPI.SubscribeActiveAssetCtx("user", func(data *hyperliquid.ActiveAssetCtxData) {})
client.WebSocketAPI.SubscribeActiveAssetData("user", func(data *hyperliquid.ActiveAssetDataData) {})
client.WebSocketAPI.SubscribeNotification("user", func(data *hyperliquid.NotificationData) {})
client.WebSocketAPI.SubscribeWebData2("user", func(data *hyperliquid.WebData2Data) {})
```

## Performance Features

- **Atomic Operations** - Lock-free reads/writes for maximum HFT performance
- **Ultra-Fast JSON** - jsoniter library for 5-10x faster processing
- **Object Pooling** - Eliminates memory allocations with sync.Pool
- **WebSocket Fallback** - Automatic HTTP fallback when WebSocket fails
- **Typed Data** - Compile-time type checking, no runtime assertions

## API Reference

- [Hyperliquid API](https://app.hyperliquid.xyz/)
- [API Documentation](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api)

## License

Apache License, Version 2.0 - see [LICENSE](LICENSE) file for details.

This is a performance-optimized fork of [Logarithm-Labs/go-hyperliquid](https://github.com/Logarithm-Labs/go-hyperliquid).
