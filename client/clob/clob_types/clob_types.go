package clob_types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/ybina/polymarket-go/client/types"
)

type RequestArgs struct {
	Method         string
	RequestPath    string
	Body           []byte
	SerializedBody []byte
}

type OrderArgs struct {
	TokenID    string          `json:"tokenID"`
	Price      decimal.Decimal `json:"price"`
	Size       decimal.Decimal `json:"size"`
	Side       types.Side      `json:"side"`
	FeeRateBps int             `json:"feeRateBps,omitempty"`
	Nonce      int             `json:"nonce,omitempty"`
	Expiration int             `json:"expiration,omitempty"`
	Taker      common.Address  `json:"taker,omitempty"`
}

type PartialCreateOrderOptions struct {
	OrderType      types.OrderType `json:"orderType"`
	TickSize       *types.TickSize `json:"tickSize"`
	NegRisk        *bool           `json:"negRisk"`
	TurnkeyAccount common.Address  `json:"turnkeyAccount"`
	SafeAccount    common.Address  `json:"safeAccount"`
}

type ClobOption struct {
	TurnkeyAccount common.Address `json:"turnkeyAccount"`
	SafeAccount    common.Address `json:"safeAccount"`
}

type MarketOrderArgs struct {
	TokenID string `json:"token_id"`

	Amount decimal.Decimal `json:"amount"`

	Side types.Side `json:"side"`

	Price decimal.Decimal `json:"price"`

	FeeRateBps int `json:"fee_rate_bps"`

	Nonce int `json:"nonce"`

	Taker common.Address `json:"taker"`

	OrderType types.OrderType `json:"order_type"`
}
