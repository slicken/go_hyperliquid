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
}

type Hyperliquid struct {
	ExchangeAPI
	InfoAPI
	WebSocket *WebSocketClient
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
		ExchangeAPI: *exchangeAPI,
		InfoAPI:     *infoAPI,
		WebSocket:   webSocket,
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
	return h.WebSocket.Connect()
}

func (h *Hyperliquid) DisconnectWebSocket() error {
	return h.WebSocket.Disconnect()
}

func (h *Hyperliquid) SubscribeOrderbook(coin string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeOrderbook(coin, handler)
}

func (h *Hyperliquid) SubscribeTrades(coin string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeTrades(coin, handler)
}

func (h *Hyperliquid) SubscribeUserFills(user string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeUserFills(user, handler)
}

func (h *Hyperliquid) SubscribeAllMids(handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeAllMids(handler)
}

func (h *Hyperliquid) SubscribeUserEvents(user string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeUserEvents(user, handler)
}

func (h *Hyperliquid) SubscribeUserFundings(user string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeUserFundings(user, handler)
}

func (h *Hyperliquid) SubscribeUserNonFundingLedgerUpdates(user string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeUserNonFundingLedgerUpdates(user, handler)
}

func (h *Hyperliquid) SubscribeUserTwapSliceFills(user string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeUserTwapSliceFills(user, handler)
}

func (h *Hyperliquid) SubscribeUserTwapHistory(user string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeUserTwapHistory(user, handler)
}

func (h *Hyperliquid) SubscribeActiveAssetCtx(coin string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeActiveAssetCtx(coin, handler)
}

func (h *Hyperliquid) SubscribeActiveAssetData(user string, coin string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeActiveAssetData(user, coin, handler)
}

func (h *Hyperliquid) SubscribeBbo(coin string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeBbo(coin, handler)
}

func (h *Hyperliquid) SubscribeCandle(coin string, interval string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeCandle(coin, interval, handler)
}

func (h *Hyperliquid) SubscribeOrderUpdates(user string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeOrderUpdates(user, handler)
}

func (h *Hyperliquid) SubscribeNotification(user string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeNotification(user, handler)
}

func (h *Hyperliquid) SubscribeWebData2(user string, handler SubscriptionHandler) error {
	return h.WebSocket.SubscribeWebData2(user, handler)
}
