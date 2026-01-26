package utils_order_builder

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/client/signer"
)

type UtilsOrderBuilder struct {
	ExchangeAddress common.Address
	ChainId         int
	Signer          *signer.Signer
	Option          Option
}

type Option struct {
	TurnkeyAccount common.Address
}
type OrderData struct {
	Maker         common.Address    `json:"maker"`
	Taker         common.Address    `json:"taker"`
	TokenID       string            `json:"tokenId"`
	MakerAmount   string            `json:"makerAmount"`
	TakerAmount   string            `json:"takerAmount"`
	Side          int               `json:"side"`
	FeeRateBps    string            `json:"feeRateBps"`
	Nonce         string            `json:"nonce"`
	Signer        common.Address    `json:"signer"`
	Expiration    string            `json:"expiration"`
	SignatureType constants.SigType `json:"signatureType"`
}

const orderTypeString = "Order(uint256 salt,address maker,address signer,address taker,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint256 expiration,uint256 nonce,uint256 feeRateBps,uint8 side,uint8 signatureType)"

type Order struct {
	Salt *big.Int `json:"salt"`

	Maker common.Address `json:"maker"`

	Signer common.Address `json:"signer"`

	Taker common.Address `json:"taker"`

	TokenID *big.Int `json:"tokenId"`

	MakerAmount *big.Int `json:"makerAmount"`

	TakerAmount *big.Int `json:"takerAmount"`

	Expiration *big.Int `json:"expiration"`

	Nonce *big.Int `json:"nonce"`

	FeeRateBps *big.Int `json:"feeRateBps"`

	Side uint8 `json:"side"`

	SignatureType uint8 `json:"signatureType"`
}

func (d *Order) OrderTypeHash() common.Hash {
	return crypto.Keccak256Hash([]byte(orderTypeString))
}

func (d *Order) OrderStructHash() (common.Hash, error) {
	typeHash := d.OrderTypeHash()

	enc := make([]byte, 0, 32*13)

	// helper
	padUint := func(x *big.Int) []byte {
		b := make([]byte, 32)
		x.FillBytes(b)
		return b
	}
	padAddr := func(a common.Address) []byte {
		b := make([]byte, 32)
		copy(b[12:], a.Bytes())
		return b
	}

	enc = append(enc, typeHash.Bytes()...)
	enc = append(enc, padUint(d.Salt)...)
	enc = append(enc, padAddr(d.Maker)...)
	enc = append(enc, padAddr(d.Signer)...)
	enc = append(enc, padAddr(d.Taker)...)
	enc = append(enc, padUint(d.TokenID)...)
	enc = append(enc, padUint(d.MakerAmount)...)
	enc = append(enc, padUint(d.TakerAmount)...)
	enc = append(enc, padUint(d.Expiration)...)
	enc = append(enc, padUint(d.Nonce)...)
	enc = append(enc, padUint(d.FeeRateBps)...)

	enc = append(enc, padUint(new(big.Int).SetUint64(uint64(d.Side)))...)

	enc = append(enc, padUint(new(big.Int).SetUint64(uint64(d.SignatureType)))...)

	return crypto.Keccak256Hash(enc), nil
}

func (d *Order) OrderDomainSeparator(
	name string,
	version string,
	chainId *big.Int,
	verifyingContract common.Address,
) common.Hash {

	typeHash := crypto.Keccak256Hash(
		[]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"),
	)

	nameHash := crypto.Keccak256Hash([]byte(name))
	versionHash := crypto.Keccak256Hash([]byte(version))

	enc := make([]byte, 0, 32*5)

	padUint := func(x *big.Int) []byte {
		b := make([]byte, 32)
		x.FillBytes(b)
		return b
	}
	padAddr := func(a common.Address) []byte {
		b := make([]byte, 32)
		copy(b[12:], a.Bytes())
		return b
	}

	enc = append(enc, typeHash.Bytes()...)
	enc = append(enc, nameHash.Bytes()...)
	enc = append(enc, versionHash.Bytes()...)
	enc = append(enc, padUint(chainId)...)
	enc = append(enc, padAddr(verifyingContract)...)

	return crypto.Keccak256Hash(enc)
}

func (d *Order) OrderEIP712Digest(
	domainSeparator common.Hash,
	structHash common.Hash,
) common.Hash {

	return crypto.Keccak256Hash(
		[]byte("\x19\x01"),
		domainSeparator.Bytes(),
		structHash.Bytes(),
	)
}

type SignedOrder struct {
	Salt int64 `json:"salt"`

	Maker common.Address `json:"maker"`

	Signer common.Address `json:"signer"`

	Taker common.Address `json:"taker"`

	TokenID string `json:"tokenId"`

	MakerAmount string `json:"makerAmount"`

	TakerAmount string `json:"takerAmount"`

	Expiration string `json:"expiration"`

	Nonce string `json:"nonce"`

	FeeRateBps string `json:"feeRateBps"`

	Side string `json:"side"`

	SignatureType uint8  `json:"signatureType"`
	Signature     string `json:"signature"`
}
