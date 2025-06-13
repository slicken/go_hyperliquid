package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	hyperliquid "github.com/slicken/go_hyperliquid"
)

func main() {
	hyperliquidClient := hyperliquid.NewHyperliquid(&hyperliquid.HyperliquidClientConfig{
		IsMainnet:      true,
		AccountAddress: os.Getenv("HYPERLIQUID_API_KEY"),
		PrivateKey:     os.Getenv("HYPERLIQUID_API_SECRET"),
	})

	// Connect to WebSocket
	err := hyperliquidClient.ConnectWebSocket()
	if err != nil {
		log.Fatal("Failed to connect to WebSocket:", err)
	}
	defer hyperliquidClient.DisconnectWebSocket()

	// Wait a bit for the connection to stabilize
	time.Sleep(1 * time.Second)

	// Subscribe to user notifications
	accountAddress := hyperliquidClient.AccountAddress()
	if accountAddress != "" {
		err = hyperliquidClient.SubscribeNotification(accountAddress, func(data interface{}) {
			notification, ok := data.(map[string]interface{})
			if !ok {
				return
			}

			// Print notification details
			fmt.Printf("\nNotification received:\n")
			printData(notification)
			// for key, value := range notification {
			// 	fmt.Printf("  %s: %v\n", key, value)
			// }
		})
		if err != nil {
			log.Printf("Failed to subscribe to notifications: %v", err)
		}

		// Subscribe to user events
		err = hyperliquidClient.SubscribeUserEvents(accountAddress, func(data interface{}) {
			events, ok := data.([]interface{})
			if !ok {
				return
			}

			fmt.Printf("\nUser Events received:\n")
			printData(events)
			// for _, event := range events {
			// 	if eventMap, ok := event.(map[string]interface{}); ok {
			// 		for key, value := range eventMap {
			// 			fmt.Printf("  %s: %v\n", key, value)
			// 		}
			// 	}
			// }
		})
		if err != nil {
			log.Printf("Failed to subscribe to user events: %v", err)
		}

		// // Subscribe to active asset context for BTC
		// err = hyperliquidClient.SubscribeActiveAssetCtx("BTC", func(data interface{}) {
		// 	ctx, ok := data.(map[string]interface{})
		// 	if !ok {
		// 		return
		// 	}

		// 	fmt.Printf("\nActive Asset Context (BTC) received:\n")
		// 	printData(ctx)
		// })
		// if err != nil {
		// 	log.Printf("Failed to subscribe to active asset context: %v", err)
		// }

		// // Subscribe to active asset data for BTC
		// err = hyperliquidClient.SubscribeActiveAssetData(accountAddress, "BTC", func(data interface{}) {
		// 	assetData, ok := data.(map[string]interface{})
		// 	if !ok {
		// 		return
		// 	}

		// 	fmt.Printf("\nActive Asset Data (BTC) received:\n")
		// 	printData(assetData)
		// 	// for key, value := range assetData {
		// 	// 	fmt.Printf("  %s: %v\n", key, value)
		// 	// }
		// })
		// if err != nil {
		// 	log.Printf("Failed to subscribe to active asset data: %v", err)
		// }

		// Subscribe to order updates
		err = hyperliquidClient.SubscribeOrderUpdates(accountAddress, func(data interface{}) {
			orders, ok := data.([]interface{})
			if !ok {
				return
			}

			fmt.Printf("\nOrder Updates received:\n")
			printData(orders)
			// for _, order := range orders {
			// 	if orderMap, ok := order.(map[string]interface{}); ok {
			// 		for key, value := range orderMap {
			// 			fmt.Printf("  %s: %v\n", key, value)
			// 		}
			// 	}
			// }
		})
		if err != nil {
			log.Printf("Failed to subscribe to order updates: %v", err)
		}

		// // Subscribe to web data 2
		// err = hyperliquidClient.SubscribeWebData2(accountAddress, func(data interface{}) {
		// 	webData, ok := data.(map[string]interface{})
		// 	if !ok {
		// 		return
		// 	}

		// 	fmt.Printf("\nWeb Data 2 received:\n")
		// 	printData(webData)
		// 	// for key, value := range webData {
		// 	// 	fmt.Printf("  %s: %v\n", key, value)
		// 	// }
		// })
		// if err != nil {
		// 	log.Printf("Failed to subscribe to web data 2: %v", err)
		// }
	}

	// Wait for interrupt signal
	log.Println("Waiting for WebSocket updates... (Press Ctrl+C to exit)")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

func printData(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal data: %v", err)
		return
	}

	fmt.Println(string(jsonData))
}
