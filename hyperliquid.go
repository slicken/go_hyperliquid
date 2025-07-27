package hyperliquid

type IHyperliquid interface {
	IExchangeAPI
	IInfoAPI
	ConnectWebSocket() error
	DisconnectWebSocket() error
	SubscribeOrderbook(coin string, handler SubscriptionHandler) error
	SubscribeTrades(coin string, handler SubscriptionHandler) error
	SubscribeUserFills(user string, handler SubscriptionHandler) error
	SubscribeAllMids(handler SubscriptionHandler) error
	SubscribeUserEvents(user string, handler SubscriptionHandler) error
	SubscribeUserFundings(user string, handler SubscriptionHandler) error
	SubscribeUserNonFundingLedgerUpdates(user string, handler SubscriptionHandler) error
	SubscribeUserTwapSliceFills(user string, handler SubscriptionHandler) error
	SubscribeUserTwapHistory(user string, handler SubscriptionHandler) error
	SubscribeActiveAssetCtx(coin string, handler SubscriptionHandler) error
	SubscribeActiveAssetData(user string, coin string, handler SubscriptionHandler) error
	SubscribeBbo(coin string, handler SubscriptionHandler) error
	SubscribeCandle(coin string, interval string, handler SubscriptionHandler) error
	SubscribeOrderUpdates(user string, handler SubscriptionHandler) error
	SubscribeNotification(user string, handler SubscriptionHandler) error
	SubscribeWebData2(user string, handler SubscriptionHandler) error
	GetLatency() int64
}

type Hyperliquid struct {
	*ExchangeAPI
	*InfoAPI
	*WebSocketClient
}

// HyperliquidClientConfig is a configuration struct for Hyperliquid API.
// PrivateKey can be empty if you only need to use the public endpoints.
// AccountAddress is the default account address for the API that can be changed with SetAccountAddress().
// AccountAddress may be different from the address build from the private key due to Hyperliquid's account system.
type HyperliquidClientConfig struct {
	IsMainnet      bool
	PrivateKey     string
	AccountAddress string
}

func NewHyperliquid(config *HyperliquidClientConfig) *Hyperliquid {
	var defaultConfig *HyperliquidClientConfig
	if config == nil {
		defaultConfig = &HyperliquidClientConfig{
			IsMainnet:      true,
			PrivateKey:     "",
			AccountAddress: "",
		}
	} else {
		defaultConfig = config
	}
	exchangeAPI := NewExchangeAPI(defaultConfig.IsMainnet)
	exchangeAPI.SetPrivateKey(defaultConfig.PrivateKey)
	exchangeAPI.SetAccountAddress(defaultConfig.AccountAddress)
	infoAPI := NewInfoAPI(defaultConfig.IsMainnet)
	infoAPI.SetAccountAddress(defaultConfig.AccountAddress)
	webSocket := NewWebSocketClient(defaultConfig.IsMainnet)
	return &Hyperliquid{
		ExchangeAPI:     exchangeAPI,
		InfoAPI:         infoAPI,
		WebSocketClient: webSocket,
	}
}

func (h *Hyperliquid) SetPrivateKey(privateKey string) error {
	err := h.ExchangeAPI.SetPrivateKey(privateKey)
	if err != nil {
		return err
	}
	return nil
}

func (h *Hyperliquid) SetAccountAddress(accountAddress string) {
	h.ExchangeAPI.SetAccountAddress(accountAddress)
	h.InfoAPI.SetAccountAddress(accountAddress)
}

func (h *Hyperliquid) AccountAddress() string {
	return h.ExchangeAPI.AccountAddress()
}

func (h *Hyperliquid) IsMainnet() bool {
	return h.ExchangeAPI.IsMainnet()
}

// WebSocket methods
func (h *Hyperliquid) ConnectWebSocket() error {
	return h.WebSocketClient.Connect()
}

func (h *Hyperliquid) DisconnectWebSocket() error {
	return h.WebSocketClient.Disconnect()
}

func (h *Hyperliquid) SubscribeOrderbook(coin string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeOrderbook(coin, handler)
}

func (h *Hyperliquid) SubscribeTrades(coin string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeTrades(coin, handler)
}

func (h *Hyperliquid) SubscribeUserFills(user string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeUserFills(user, handler)
}

func (h *Hyperliquid) SubscribeAllMids(handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeAllMids(handler)
}

func (h *Hyperliquid) SubscribeUserEvents(user string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeUserEvents(user, handler)
}

func (h *Hyperliquid) SubscribeUserFundings(user string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeUserFundings(user, handler)
}

func (h *Hyperliquid) SubscribeUserNonFundingLedgerUpdates(user string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeUserNonFundingLedgerUpdates(user, handler)
}

func (h *Hyperliquid) SubscribeUserTwapSliceFills(user string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeUserTwapSliceFills(user, handler)
}

func (h *Hyperliquid) SubscribeUserTwapHistory(user string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeUserTwapHistory(user, handler)
}

func (h *Hyperliquid) SubscribeActiveAssetCtx(coin string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeActiveAssetCtx(coin, handler)
}

func (h *Hyperliquid) SubscribeActiveAssetData(user string, coin string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeActiveAssetData(user, coin, handler)
}

func (h *Hyperliquid) SubscribeBbo(coin string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeBbo(coin, handler)
}

func (h *Hyperliquid) SubscribeCandle(coin string, interval string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeCandle(coin, interval, handler)
}

func (h *Hyperliquid) SubscribeOrderUpdates(user string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeOrderUpdates(user, handler)
}

func (h *Hyperliquid) SubscribeNotification(user string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeNotification(user, handler)
}

func (h *Hyperliquid) SubscribeWebData2(user string, handler SubscriptionHandler) error {
	return h.WebSocketClient.SubscribeWebData2(user, handler)
}

// GetLatency returns the current WebSocket latency in milliseconds
func (h *Hyperliquid) GetLatency() int64 {
	return h.WebSocketClient.GetLatency()
}
