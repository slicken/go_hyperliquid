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

	meta, err := hl.GetMeta()
	if err != nil {
		log.Fatalf("Error getting meta: %v", err)
	} else {
		for _, asset := range meta.Universe {
			err = hl.WebSocketAPI.SubscribeTrades(asset.Name, func(data interface{}) {
				trades, ok := data.([]interface{})
				if !ok {
					log.Fatalf("error: %v", ok)
				}
				log.Printf("%s: %d trades", asset.Name, len(trades))
			})
			if err != nil {
				log.Fatalf("Failed to subscribe to trades: %v", err)
			}
			err = hl.WebSocketAPI.SubscribeOrderUpdates(config.AccountAddress, func(data interface{}) {
				log.Printf("%s: ORDER UPDATE RECEIVED", asset.Name)
				orders, ok := data.([]interface{})
				if !ok {
					log.Printf("Order data is not array: %T - %+v", data, data)
					return
				}
				for _, orderData := range orders {
					orderMap, ok := orderData.(map[string]interface{})
					if !ok {
						continue
					}
					orderInfo, ok := orderMap["order"].(map[string]interface{})
					if !ok {
						continue
					}
					orderID, ok := orderInfo["oid"].(float64)
					if !ok {
						continue
					}
					coin, ok := orderInfo["coin"].(string)
					if !ok {
						continue
					}

					fmt.Printf("Cancelling order with ID: %d\n", int64(orderID))
					cancelResponse, err := hl.ExchangeAPI.CancelOrderByOID(coin, int64(orderID))
					if err != nil {
						log.Printf("Failed to cancel order: %v", err)
					} else {
						fmt.Printf("Order cancelled successfully: %+v\n", cancelResponse)
					}

				}
			})
			if err != nil {
				log.Fatalf("Failed to subscribe to order updates: %v", err)
			}

		}
	}

	log.Println("Subscribed to all assets")
	time.Sleep(10 * time.Second)
	go func() {
		for {
			log.Println("Starting trade loop")

			// Create a limit order
			orderRequest := hyperliquid.OrderRequest{
				Coin:    "BTC",
				IsBuy:   true,
				Sz:      0.001,    // Small size for testing
				LimitPx: 109000.0, // Example price
				OrderType: hyperliquid.OrderType{
					Limit: &hyperliquid.LimitOrderType{
						Tif: hyperliquid.TifGtc, // Good till cancelled
					},
				},
				ReduceOnly: false,
				Cloid:      hyperliquid.GetRandomCloid(),
			}

			fmt.Printf("Placing order: %+v\n", orderRequest)
			response, err := hl.ExchangeAPI.Order(orderRequest, hyperliquid.GroupingNa)
			if err != nil {
				log.Printf("Failed to place order: %v", err)
			} else {
				fmt.Printf("Order placed successfully: %+v\n", response)
			}

			time.Sleep(10 * time.Second)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\nShutting down...")
}
