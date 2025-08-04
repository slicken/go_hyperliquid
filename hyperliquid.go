package hyperliquid

// IHyperliquid is the main interface that embeds all other APIs
type IHyperliquid interface {
	IExchangeAPI
	IInfoAPI
	IWebSocketAPI
}

// Hyperliquid is the main struct that contains all APIs
type Hyperliquid struct {
	*ExchangeAPI
	*InfoAPI
	*WebSocketAPI
}

// HyperliquidClientConfig is a configuration struct for Hyperliquid API.
// PrivateKey can be empty if you only need to use the public endpoints.
// AccountAddress is the default account address for the API that can be changed with SetAccountAddress().
// AccountAddress may be different from the address build from the private key due to Hyperliquid's account system.
type HyperliquidClientConfig struct {
	IsMainnet      bool
	AccountAddress string
	PrivateKey     string
}

func NewHyperliquid(config *HyperliquidClientConfig) *Hyperliquid {
	var defaultConfig *HyperliquidClientConfig
	if config == nil {
		defaultConfig = &HyperliquidClientConfig{
			IsMainnet:      true,
			AccountAddress: "",
			PrivateKey:     "",
		}
	} else {
		defaultConfig = config
	}

	// Create single instances of each API - they handle their own setup
	exchangeAPI := NewExchangeAPI(defaultConfig.IsMainnet, defaultConfig.AccountAddress, defaultConfig.PrivateKey)
	infoAPI := NewInfoAPI(defaultConfig.IsMainnet, defaultConfig.AccountAddress, defaultConfig.PrivateKey)
	webSocketAPI := NewWebSocketAPI(defaultConfig.IsMainnet)

	// Connect WebSocket API to Client instances for automatic fallback
	exchangeAPI.SetWebSocketAPI(webSocketAPI)
	infoAPI.SetWebSocketAPI(webSocketAPI)

	return &Hyperliquid{
		ExchangeAPI:  exchangeAPI,
		InfoAPI:      infoAPI,
		WebSocketAPI: webSocketAPI,
	}
}

// AccountAddress returns the account address
func (h *Hyperliquid) AccountAddress() string {
	return h.ExchangeAPI.AccountAddress()
}

// IsMainnet returns true if the client is connected to mainnet
func (h *Hyperliquid) IsMainnet() bool {
	return h.ExchangeAPI.IsMainnet()
}
