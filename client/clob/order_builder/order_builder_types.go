package order_builder

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/client/signer"
)

type OrderBuilder struct {
	Signer  *signer.Signer
	SigType constants.SigType
	Funder  common.Address
}
