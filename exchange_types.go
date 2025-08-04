package hyperliquid

import (
	"fmt"
)

type RsvSignature struct {
	R string `json:"r"`
	S string `json:"s"`
	V byte   `json:"v"`
}

// Base request for /exchange endpoint
type ExchangeRequest struct {
	Action       any          `json:"action"`
	Nonce        uint64       `json:"nonce"`
	Signature    RsvSignature `json:"signature"`
	VaultAddress *string      `json:"vaultAddress,omitempty" msgpack:",omitempty"`
}

type AssetInfo struct {
	SzDecimals  int
	WeiDecimals int
	AssetId     int
	SpotName    string // for spot asset (e.g. "@107")
}

type OrderRequest struct {
	Coin       string    `json:"coin"`
	IsBuy      bool      `json:"is_buy"`
	Sz         float64   `json:"sz"`
	LimitPx    float64   `json:"limit_px"`
	OrderType  OrderType `json:"order_type"`
	ReduceOnly bool      `json:"reduce_only"`
	Cloid      string    `json:"cloid,omitempty"`
}

type OrderType struct {
	Limit   *LimitOrderType   `json:"limit,omitempty" msgpack:"limit,omitempty"`
	Trigger *TriggerOrderType `json:"trigger,omitempty"  msgpack:"trigger,omitempty"`
}

type LimitOrderType struct {
	Tif string `json:"tif" msgpack:"tif"`
}

const (
	TifGtc string = "Gtc"
	TifIoc string = "Ioc"
	TifAlo string = "Alo"
)

type TriggerOrderType struct {
	IsMarket  bool   `json:"isMarket" msgpack:"isMarket"`
	TriggerPx string `json:"triggerPx" msgpack:"triggerPx"`
	TpSl      TpSl   `json:"tpsl" msgpack:"tpsl"`
}

type TpSl string

const TriggerTp TpSl = "tp"
const TriggerSl TpSl = "sl"

type Grouping string

const GroupingNa Grouping = "na"
const GroupingTpSl Grouping = "positionTpsl"

type Message struct {
	Source       string `json:"source"`
	ConnectionId []byte `json:"connectionId"`
}

type OrderWire struct {
	Asset      int           `msgpack:"a" json:"a"`
	IsBuy      bool          `msgpack:"b" json:"b"`
	LimitPx    string        `msgpack:"p" json:"p"`
	SizePx     string        `msgpack:"s" json:"s"`
	ReduceOnly bool          `msgpack:"r" json:"r"`
	OrderType  OrderTypeWire `msgpack:"t" json:"t"`
	Cloid      string        `msgpack:"c,omitempty" json:"c,omitempty"`
}
type ModifyResponse struct {
	Status   string             `json:"status"`
	Response OrderInnerResponse `json:"response"`
}
type ModifyOrderWire struct {
	OrderId int       `msgpack:"oid" json:"oid"`
	Order   OrderWire `msgpack:"order" json:"order"`
}
type ModifyOrderAction struct {
	Type     string            `msgpack:"type" json:"type"`
	Modifies []ModifyOrderWire `msgpack:"modifies" json:"modifies"`
}

type ModifyOrderRequest struct {
	OrderId    int       `json:"oid"`
	Coin       string    `json:"coin"`
	IsBuy      bool      `json:"is_buy"`
	Sz         float64   `json:"sz"`
	LimitPx    float64   `json:"limit_px"`
	OrderType  OrderType `json:"order_type"`
	ReduceOnly bool      `json:"reduce_only"`
	Cloid      string    `json:"cloid,omitempty"`
}

type OrderTypeWire struct {
	Limit   *LimitOrderType   `json:"limit,omitempty" msgpack:"limit,omitempty"`
	Trigger *TriggerOrderType `json:"trigger,omitempty" msgpack:"trigger,omitempty"`
}

type PlaceOrderAction struct {
	Type     string      `msgpack:"type" json:"type"`
	Orders   []OrderWire `msgpack:"orders" json:"orders"`
	Grouping Grouping    `msgpack:"grouping" json:"grouping"`
}

type OrderResponse struct {
	Status   string             `json:"status"`
	Response OrderInnerResponse `json:"response"`
}

type OrderInnerResponse struct {
	Type string       `json:"type"`
	Data DataResponse `json:"data"`
}

type DataResponse struct {
	Statuses []StatusResponse `json:"statuses"`
}

type StatusResponse struct {
	Resting RestingStatus `json:"resting,omitempty"`
	Filled  FilledStatus  `json:"filled,omitempty"`
	Error   string        `json:"error,omitempty"`
	Status  string        `json:"status,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling for StatusResponse.
// It first checks if the incoming JSON is a simple string. If so, it assigns the
// value to the Status field. Otherwise, it unmarshals the JSON into the struct normally.
func (sr *StatusResponse) UnmarshalJSON(data []byte) error {
	// Try to unmarshal data as a string.
	var s string
	if err := FastUnmarshal(data, &s); err == nil {
		sr.Status = s
		return nil
	}

	// Otherwise, unmarshal as a full object.
	// Use an alias to avoid infinite recursion.
	type Alias StatusResponse
	var alias Alias
	if err := FastUnmarshal(data, &alias); err != nil {
		return fmt.Errorf("StatusResponse: unable to unmarshal data as string or object: %w", err)
	}
	*sr = StatusResponse(alias)
	return nil
}

type CancelRequest struct {
	OrderId int `json:"oid"`
	Coin    int `json:"coin"`
}

type CancelOidOrderAction struct {
	Type    string          `msgpack:"type" json:"type"`
	Cancels []CancelOidWire `msgpack:"cancels" json:"cancels"`
}

type CancelOidWire struct {
	Asset int `msgpack:"a" json:"a"`
	Oid   int `msgpack:"o" json:"o"`
}

type CancelCloidWire struct {
	Asset int    `msgpack:"asset" json:"asset"`
	Cloid string `msgpack:"cloid" json:"cloid"`
}

type CancelCloidOrderAction struct {
	Type    string            `msgpack:"type" json:"type"`
	Cancels []CancelCloidWire `msgpack:"cancels" json:"cancels"`
}

type RestingStatus struct {
	OrderId int    `json:"oid"`
	Cloid   string `json:"cloid,omitempty"`
}

type CloseRequest struct {
	Coin     string
	Px       float64
	Sz       float64
	Slippage float64
	Cloid    string
}

type FilledStatus struct {
	OrderId int     `json:"oid"`
	AvgPx   float64 `json:"avgPx,string"`
	TotalSz float64 `json:"totalSz,string"`
	Cloid   string  `json:"cloid,omitempty"`
}

type Liquidation struct {
	User      string `json:"liquidatedUser"`
	MarkPrice string `json:"markPx"`
	Method    string `json:"method"`
}

type UpdateLeverageAction struct {
	Type     string `msgpack:"type" json:"type"`
	Asset    int    `msgpack:"asset" json:"asset"`
	IsCross  bool   `msgpack:"isCross" json:"isCross"`
	Leverage int    `msgpack:"leverage" json:"leverage"`
}

type DefaultExchangeResponse struct {
	Status   string `json:"status"`
	Response struct {
		Type string `json:"type"`
	} `json:"response"`
}

// Depending on Type this struct can has different non-nil fields
type NonFundingDelta struct {
	Type   string  `json:"type"`
	Usdc   float64 `json:"usdc,string,omitempty"`
	Amount float64 `json:"amount,string,omitempty"`
	ToPerp bool    `json:"toPerp,omitempty"`
	Token  string  `json:"token,omitempty"`
	Fee    float64 `json:"fee,string,omitempty"`
	Nonce  int64   `json:"nonce"`
}

type FundingDelta struct {
	Asset       string `json:"coin"`
	FundingRate string `json:"fundingRate"`
	Size        string `json:"szi"`
	UsdcAmount  string `json:"usdc"`
}

type Withdrawal struct {
	Time   int64   `json:"time"`
	Hash   string  `json:"hash"`
	Amount float64 `json:"usdc"`
	Fee    float64 `json:"fee"`
	Nonce  int64   `json:"nonce"`
}

type Deposit struct {
	Hash   string  `json:"hash,omitempty"`
	Time   int64   `json:"time,omitempty"`
	Amount float64 `json:"usdc,omitempty"`
}

type WithdrawAction struct {
	Type             string `msgpack:"type" json:"type"`
	Destination      string `msgpack:"destination" json:"destination"`
	Amount           string `msgpack:"amount" json:"amount"`
	Time             uint64 `msgpack:"time" json:"time"`
	HyperliquidChain string `msgpack:"hyperliquidChain" json:"hyperliquidChain"`
	SignatureChainID string `msgpack:"signatureChainId" json:"signatureChainId"`
}

type WithdrawResponse struct {
	Status string `json:"status"`
	Nonce  int64
}
