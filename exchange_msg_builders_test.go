package hyperliquid

import (
	"math"
	"testing"
)

func GetEmptyExchangeAPI() *ExchangeAPI {
	exchangeAPI := NewExchangeAPI(true, "", "")
	return exchangeAPI
}

func TestBuildOrderEIP712(t *testing.T) {
	exchangeAPI := NewExchangeAPI(true, "", "")
	// input params
	coin := "ETH"
	size := 0.1
	price := 2500.0

	isBuy := IsBuy(size)
	orderType := OrderType{
		Limit: &LimitOrderType{
			Tif: TifIoc,
		},
	}
	orderRequest := OrderRequest{
		Coin:       coin,
		IsBuy:      isBuy,
		Sz:         math.Abs(size),
		LimitPx:    price,
		OrderType:  orderType,
		ReduceOnly: false,
	}
	res, err := exchangeAPI.BuildOrderEIP712(orderRequest, GroupingNa)
	if err != nil {
		t.Errorf("BuildOrder() error = %v", err)
	}
	t.Logf("BuildOrder() = %+v", res)
}
