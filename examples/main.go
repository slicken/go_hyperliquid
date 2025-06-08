package main

import (
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
	hyperliquidClient.SetDebugActive()

	// Connect to WebSocket
	err := hyperliquidClient.ConnectWebSocket()
	if err != nil {
		log.Fatal("Failed to connect to WebSocket:", err)
	}
	defer hyperliquidClient.DisconnectWebSocket()

	// Wait a bit for the connection to stabilize
	time.Sleep(1 * time.Second)

	// Subscribe to user fills
	accountAddress := hyperliquidClient.AccountAddress()
	err = hyperliquidClient.SubscribeUserFills(accountAddress, func(data interface{}) {
		fills, ok := data.(map[string]interface{})
		if !ok {
			return
		}

		fillsArray, ok := fills["fills"].([]interface{})
		if !ok || len(fillsArray) == 0 {
			return
		}

		// Process latest fill
		fill, ok := fillsArray[len(fillsArray)-1].(map[string]interface{})
		if !ok {
			return
		}

		// Extract fill details
		coin := fill["coin"].(string)
		side := "BUY"
		if fill["side"].(string) == "A" {
			side = "SELL"
		}
		price := fill["px"].(float64)
		size := fill["sz"].(float64)
		fee := fill["fee"].(float64)
		time := fill["time"].(float64)

		log.Printf("New Fill:\n  Coin: %s\n  Side: %s\n  Price: %.2f\n  Size: %.8f\n  Fee: %.8f\n  Time: %v\n",
			coin, side, price, size, fee, time)
	})
	if err != nil {
		log.Printf("Failed to subscribe to user fills: %v", err)
		return
	}

	// Get initial account state
	res, err := hyperliquidClient.GetAccountState()
	if err != nil {
		log.Print(err)
	}
	log.Printf("Account State: Balance: %.8f USDC", res.Withdrawable)

	// Wait for interrupt signal
	log.Println("Waiting for fills... (Press Ctrl+C to exit)")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
