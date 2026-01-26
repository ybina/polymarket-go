package model

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type CreateProxy struct {
	PaymentToken    common.Address
	Payment         *big.Int
	PaymentReceiver common.Address
}

var createProxyTypeHash = crypto.Keccak256Hash(
	[]byte("CreateProxy(address paymentToken,uint256 payment,address paymentReceiver)"),
)

func (c *CreateProxy) StructHash() []byte {

	var enc []byte

	enc = append(enc, createProxyTypeHash.Bytes()...)

	enc = append(enc, common.LeftPadBytes(c.PaymentToken.Bytes(), 32)...)

	enc = append(enc, common.LeftPadBytes(c.Payment.Bytes(), 32)...)

	enc = append(enc, common.LeftPadBytes(c.PaymentReceiver.Bytes(), 32)...)

	return crypto.Keccak256(enc)
}
