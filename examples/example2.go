package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	hyperliquid "go_hyperliquid"
)

func main() {
	// Initialize client with automatic WebSocket fallback
	config := &hyperliquid.HyperliquidClientConfig{
		IsMainnet:      true,
		AccountAddress: strings.ToLower(os.Getenv("HYPERLIQUID_API_KEY")),    // Main account address
		PrivateKey:     strings.ToLower(os.Getenv("HYPERLIQUID_API_SECRET")), // API wallet private key
	}

	log.Printf("Using account address: %s", config.AccountAddress)
	hl := hyperliquid.NewHyperliquid(config)

	// Enable debug mode to see transport logs
	hl.WebSocketAPI.SetDebug(false)

	// Connect to WebSocket (optional - will auto-fallback to HTTP if not connected)
	err := hl.WebSocketAPI.Connect()
	if err != nil {
		log.Fatalf("WebSocket connection failed, will use HTTP: %v", err)
	} else {
		log.Println("WebSocket connected successfully")
	}

	err = hl.WebSocketAPI.SubscribeUserFills(config.AccountAddress, func(data interface{}) {
		// Try to assert the expected structure: map[string]interface{} with "isSnapshot" key
		if m, ok := data.(map[string]interface{}); ok {
			if isSnapshot, ok := m["isSnapshot"].(bool); ok && isSnapshot {
				// Don't print snapshot
				return
			}
		}
		log.Printf("%+v\n-----\n", data)
	})
	if err != nil {
		log.Printf("error: %v", err)
	}

	log.Println("Subscribed.")

	time.Sleep(2 * time.Second)
	i := 0
	go func() {
		var orderRequest hyperliquid.OrderRequest

		for {
			if i%2 == 0 {
				i++
				log.Printf("sending Marketorder\n\n")

				orderRequest = hyperliquid.OrderRequest{
					Coin:    "BTC",
					IsBuy:   false,
					Sz:      0.0002,
					LimitPx: 109000.0,
					OrderType: hyperliquid.OrderType{
						Limit: &hyperliquid.LimitOrderType{
							Tif: hyperliquid.TifIoc, // Ioc = Market order
						},
					},
					ReduceOnly: false,
					Cloid:      hyperliquid.GetRandomCloid(),
				}

			} else {
				i++
				log.Printf("sending Limitorder\n\n")

				orderRequest = hyperliquid.OrderRequest{
					Coin:    "BTC",
					IsBuy:   true,
					Sz:      0.0002,
					LimitPx: 109000.0,
					OrderType: hyperliquid.OrderType{
						Limit: &hyperliquid.LimitOrderType{
							Tif: hyperliquid.TifGtc, // Gtc = Limit order
						},
					},
					ReduceOnly: false,
					Cloid:      hyperliquid.GetRandomCloid(),
				}
			}

			// fmt.Printf("Placing order: %+v\n", orderRequest)
			response, err := hl.ExchangeAPI.Order(orderRequest, hyperliquid.GroupingNa)
			if err != nil {
				log.Printf("FAILED: %v", err)
			}
			_ = response
			// else {
			// 	fmt.Printf("SUCCESS: %+v\n", response)
			// }

			time.Sleep(10 * time.Second)
		}
	}()

	/* OUTPUT:

	2025/08/09 09:28:59 Using account address: 0x70173d313e536bd2858f4cbaa59bbd9451dc97bd
	2025/08/09 09:29:03 WebSocket connected successfully
	2025/08/09 09:29:03 Subscribed.
	2025/08/09 09:29:05 sending Marketorder // Market order

	2025/08/09 09:29:06 map[fills:[map[cloid:0x328a6818fe077c09ebd7aca549b8c2a9 closedPnl:0.0 coin:BTC crossed:true dir:Open Short fee:0.009598 feeToken:USDC hash:0x63c80338e0da3fd8d24804292c9d3d02025a005d1c32b79c29f0ea515d538a84 oid:1.29575891351e+11 px:116945.0 side:A startPosition:0.0 sz:0.0002 tid:5.45600099941465e+14 time:1.754724546288e+12 twapId:<nil>]] user:0x70173d313e536bd2858f4cbaa59bbd9451dc97bd]
	-----
	2025/08/09 09:29:16 sending Limitorder

	2025/08/09 09:29:28 sending Marketorder

	2025/08/09 09:29:30 map[fills:[map[cloid:0x5171005c71224adb1859944a27117950 closedPnl:0.0 coin:BTC crossed:true dir:Open Short fee:0.009599 feeToken:USDC hash:0xdeae50346937c95e378c04292c9e74020171009c254481e94534ac864cd9f540 oid:1.29576017649e+11 px:116952.0 side:A startPosition:-0.0002 sz:0.0002 tid:4.97570931090393e+14 time:1.75472456949e+12 twapId:<nil>]] user:0x70173d313e536bd2858f4cbaa59bbd9451dc97bd]
	-----
	^C

	*/

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\nShutting down...")
}
