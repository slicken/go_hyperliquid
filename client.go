package hyperliquid

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// IClient is the interface that wraps the basic Requst method.
//
// Request method sends a POST request to the HyperLiquid API.
// IsMainnet method returns true if the client is connected to the mainnet.
// debug method enables debug mode.
// SetPrivateKey method sets the private key for the client.
type IClient interface {
	IAPIService
	// SetPrivateKey(privateKey string) error
	// SetAccountAddress(address string)
	// AccountAddress() string
	Debug(status bool)
	IsMainnet() bool
}

// Client is the default implementation of the Client interface.
//
// It contains the base URL of the HyperLiquid API, the HTTP client, the debug mode,
// the network type, the private key, and the logger.
// The debug method prints the debug messages.
type Client struct {
	baseUrl        string        // Base URL of the HyperLiquid API
	privateKey     string        // Private key for the client
	defaultAddress string        // Default address for the client
	isMainnet      bool          // Network type
	Debug          bool          // Debug mode
	httpClient     *http.Client  // HTTP client
	keyManager     *PKeyManager  // Private key manager
	Logger         *log.Logger   // Logger for debug messages
	webSocketAPI   *WebSocketAPI // WebSocket API for automatic fallback
}

// Returns the private key manager connected to the API.
func (client *Client) KeyManager() *PKeyManager {
	return client.keyManager
}

// getAPIURL returns the API URL based on the network type.
func getURL(isMainnet bool) string {
	if isMainnet {
		return MAINNET_API_URL
	} else {
		return TESTNET_API_URL
	}
}

// NewClient returns a new instance of the Client struct.
func NewClient(isMainnet bool) *Client {
	logger := log.New()
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		PadLevelText:  true,
	})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(log.DebugLevel)
	return &Client{
		baseUrl:        getURL(isMainnet),
		httpClient:     http.DefaultClient,
		Debug:          false,
		isMainnet:      isMainnet,
		privateKey:     "",
		defaultAddress: "",
		Logger:         logger,
		keyManager:     nil,
	}
}

// debug prints the debug messages.
func (client *Client) debug(format string, v ...interface{}) {
	if client.Debug {
		client.Logger.Debugf(format, v...)
	}
}

// SetPrivateKey sets the private key for the client.
func (client *Client) SetPrivateKey(privateKey string) error {
	switch {
	case strings.HasPrefix(privateKey, "0x"):
		privateKey = strings.TrimPrefix(privateKey, "0x") // remove 0x prefix from private key
	}
	client.privateKey = privateKey
	var err error
	client.keyManager, err = NewPKeyManager(privateKey)
	return err
}

// Some methods need public address to gather info (from infoAPI).
// In case you use PKeyManager from API section https://app.hyperliquid.xyz/API
// Then you can use this method to set the address.
func (client *Client) SetAccountAddress(address string) {
	client.defaultAddress = address
}

// Returns the public address connected to the API.
func (client *Client) AccountAddress() string {
	return client.defaultAddress
}

// IsMainnet returns true if the client is connected to the mainnet.
func (client *Client) IsMainnet() bool {
	return client.isMainnet
}

// SetDebugActive enables debug mode.
func (client *Client) SetDebug(status bool) {
	client.Debug = status
}

// SetWebSocketAPI sets the WebSocket API reference for automatic fallback
func (client *Client) SetWebSocketAPI(wsAPI *WebSocketAPI) {
	client.webSocketAPI = wsAPI
}

// Request sends a POST request to the HyperLiquid API.
// If WebSocket is connected, it will use WebSocket instead of HTTP.
func (client *Client) Request(endpoint string, payload any) ([]byte, error) {
	// Try WebSocket first if connected
	if client.webSocketAPI != nil && client.webSocketAPI.IsConnected() {
		client.debug("WebSocket connected, checking if endpoint supports WebSocket...")
		return client.requestViaWebSocket(endpoint, payload)
	}

	// Fallback to HTTP
	client.debug("Using HTTP for request to %s (WebSocket not connected or not supported)", endpoint)
	return client.requestViaHTTP(endpoint, payload)
}

// requestViaWebSocket sends a request via WebSocket
func (client *Client) requestViaWebSocket(endpoint string, payload any) ([]byte, error) {
	// Clean endpoint by removing leading slash
	cleanEndpoint := strings.TrimPrefix(endpoint, "/")

	// Use WebSocket for info requests
	if cleanEndpoint == "info" {
		client.debug("Using WebSocket for info request")
		response, err := client.webSocketAPI.PostRequest("info", payload)
		if err != nil {
			return nil, err
		}

		// The WebSocket response has a nested structure:
		// response.Response.Payload contains the actual API response
		// We need to extract the data from the nested structure
		if payloadMap, ok := response.Response.Payload.(map[string]interface{}); ok {
			if data, exists := payloadMap["data"]; exists {
				// Return the data directly
				responseBytes, err := FastMarshal(data)
				if err != nil {
					return nil, err
				}
				return responseBytes, nil
			}
		}

		// Fallback: convert response to bytes as before
		responseBytes, err := FastMarshal(response.Response.Payload)
		if err != nil {
			return nil, err
		}

		return responseBytes, nil
	}

	// Use WebSocket for exchange requests (orders, cancels, modifies)
	if cleanEndpoint == "exchange" {
		client.debug("Using WebSocket for exchange request")

		// Handle ExchangeRequest struct
		exchangeReq, ok := payload.(ExchangeRequest)
		if !ok {
			client.debug("Invalid payload format for WebSocket exchange request")
			return client.requestViaHTTP(endpoint, payload)
		}

		client.debug("Sending via WebSocket - Action: %+v", exchangeReq.Action)
		client.debug("Sending via WebSocket - Nonce: %d", exchangeReq.Nonce)
		client.debug("Sending via WebSocket - Signature: %+v", exchangeReq.Signature)

		// Send via WebSocket
		response, err := client.webSocketAPI.PostOrderRequest(exchangeReq.Action, exchangeReq.Nonce, exchangeReq.Signature, exchangeReq.VaultAddress)
		if err != nil {
			client.debug("WebSocket exchange request failed: %v", err)
			return client.requestViaHTTP(endpoint, payload)
		}

		client.debug("WebSocket response received: %+v", response)

		// The WebSocket response has a nested structure:
		// response.Response.Payload contains the actual API response
		// We need to extract the nested response from the payload
		if payloadMap, ok := response.Response.Payload.(map[string]interface{}); ok {
			if nestedResponse, exists := payloadMap["response"]; exists {
				// The nested response is the actual API response
				// We need to wrap it in the expected OrderResponse format
				wrappedResponse := map[string]interface{}{
					"status":   "ok",
					"response": nestedResponse,
				}

				// Return the wrapped response
				responseBytes, err := FastMarshal(wrappedResponse)
				if err != nil {
					return nil, err
				}
				client.debug("WebSocket wrapped response bytes: %s", string(responseBytes))
				return responseBytes, nil
			}
		}

		// Fallback: convert response to bytes as before
		responseBytes, err := FastMarshal(response.Response.Payload)
		if err != nil {
			return nil, err
		}

		client.debug("WebSocket response bytes: %s", string(responseBytes))
		return responseBytes, nil
	}

	// Fallback to HTTP for unsupported endpoints
	client.debug("WebSocket not supported for %s endpoint, falling back to HTTP", endpoint)
	return client.requestViaHTTP(endpoint, payload)
}

// requestViaHTTP sends a request via HTTP (original implementation)
func (client *Client) requestViaHTTP(endpoint string, payload any) ([]byte, error) {
	client.debug("Using HTTP for %s request", endpoint)
	endpoint = strings.TrimPrefix(endpoint, "/") // Remove leading slash if present
	url := fmt.Sprintf("%s/%s", client.baseUrl, endpoint)
	client.debug("Request to %s", url)
	payloadBytes, err := FastMarshal(payload)
	if err != nil {
		client.debug("Error FastMarshal: %s", err)
		return nil, err
	}
	client.debug("Request payload: %s", string(payloadBytes))
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		client.debug("Error http.NewRequest: %s", err)
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		client.debug("Error client.httpClient.Do: %s", err)
		return nil, err
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	defer func() {
		cerr := response.Body.Close()
		// Only overwrite the retured error if the original error was nil and an
		// error occurred while closing the body.
		if err == nil && cerr != nil {
			err = cerr
		}
	}()
	client.debug("response: %#v", response)
	client.debug("response body: %s", string(data))
	client.debug("response status code: %d", response.StatusCode)
	if response.StatusCode >= http.StatusBadRequest {
		// If the status code is 400 or greater, return an error
		return nil, APIError{Message: fmt.Sprintf("HTTP %d: %s", response.StatusCode, data)}
	}
	return data, nil
}
