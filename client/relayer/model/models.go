package model

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ybina/polymarket-go/client/types"
)

type OperationType uint8

const (
	Call         OperationType = 0
	DelegateCall OperationType = 1
)

type SafeTransaction struct {
	To        common.Address `json:"to"`
	Operation OperationType  `json:"operation"`
	Data      string         `json:"data"`
	Value     string         `json:"value"`
}

type TransactionType string

const (
	TransactionTypeSafe       TransactionType = "SAFE"
	TransactionTypeSafeCreate TransactionType = "SAFE-CREATE"
)

type SignatureParams struct {
	// SAFE
	GasPrice       *string         `json:"gasPrice,omitempty"`
	Operation      *string         `json:"operation,omitempty"`
	SafeTxnGas     *string         `json:"safeTxnGas,omitempty"`
	BaseGas        *string         `json:"baseGas,omitempty"`
	GasToken       *common.Address `json:"gasToken,omitempty"`
	RefundReceiver *common.Address `json:"refundReceiver,omitempty"`

	// SAFE-CREATE
	PaymentToken    *common.Address `json:"paymentToken,omitempty"`
	Payment         *string         `json:"payment,omitempty"`
	PaymentReceiver *common.Address `json:"paymentReceiver,omitempty"`
}

type TransactionRequest struct {
	Type            TransactionType  `json:"type"`
	From            common.Address   `json:"from"`
	To              common.Address   `json:"to"`
	ProxyWallet     common.Address   `json:"proxyWallet"`
	Data            string           `json:"data"`
	Signature       string           `json:"signature"`
	SignatureParams *SignatureParams `json:"signatureParams,omitempty"`

	Value    *string `json:"value,omitempty"`
	Nonce    *string `json:"nonce,omitempty"`
	Metadata *string `json:"metadata,omitempty"`
}

func (t *TransactionRequest) ToMap() map[string]interface{} {
	raw, _ := json.Marshal(t)
	out := map[string]interface{}{}
	_ = json.Unmarshal(raw, &out)
	return out
}

type SafeTransactionArgs struct {
	FromAddress  common.Address
	Nonce        uint64
	ChainID      types.Chain
	Transactions []SafeTransaction
}

type SafeCreateTransactionArgs struct {
	FromAddress     common.Address
	ChainID         types.Chain
	PaymentToken    common.Address
	Payment         string
	PaymentReceiver common.Address
}

type RelayerTransactionState string

const (
	StateNew       RelayerTransactionState = "STATE_NEW"
	StateExecuted  RelayerTransactionState = "STATE_EXECUTED"
	StateMined     RelayerTransactionState = "STATE_MINED"
	StateInvalid   RelayerTransactionState = "STATE_INVALID"
	StateConfirmed RelayerTransactionState = "STATE_CONFIRMED"
	StateFailed    RelayerTransactionState = "STATE_FAILED"
)

type SplitSig struct {
	R string
	S string
	V string
}
