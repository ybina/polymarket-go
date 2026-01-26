package utils_order_builder

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ybina/polymarket-go/client/clob/utils"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/client/relayer/model/polyEip712"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
	"github.com/ybina/polymarket-go/tools/eip712"
)

func NewUtilsOrderBuilder(exchange common.Address, chainId int, signerHandler *signer.Signer, option Option) (*UtilsOrderBuilder, error) {
	if signerHandler.SignerType() == signer.Turnkey && option.TurnkeyAccount == constants.ZERO_ADDRESS {
		return &UtilsOrderBuilder{}, errors.New("turnkeyAccount is empty")
	}
	return &UtilsOrderBuilder{
		ExchangeAddress: exchange,
		ChainId:         chainId,
		Signer:          signerHandler,
		Option:          option,
	}, nil
}

func (b *UtilsOrderBuilder) GenerateSalt() *big.Int {
	now := time.Now().UTC().UnixNano()
	r := rand.Float64()

	salt := float64(now) * r
	saltInt := int64(math.Round(salt))

	return big.NewInt(saltInt)
}

func (b *UtilsOrderBuilder) BuildSignedOrder(data OrderData) (SignedOrder, error) {
	order, err := b.buildOrder(data)
	if err != nil {
		return SignedOrder{}, err
	}

	structHash, err := order.OrderStructHash()
	if err != nil {
		return SignedOrder{}, err
	}
	name := "Polymarket CTF Exchange"
	version := "1"
	chainId := b.ChainId
	contract := b.ExchangeAddress.Hex()
	domain := polyEip712.MakeDomain(&name, &version, &chainId, &contract, nil)

	domainSep, err := domain.HashStruct()
	domainSepHash := common.BytesToHash(domainSep[:])
	digest := order.OrderEIP712Digest(domainSepHash, structHash)
	var sig string
	if b.Signer.SignerType() == signer.Turnkey {
		sig, err = b.Signer.SignHashWithTurnkey(digest.Hex(), b.Option.TurnkeyAccount)
		if err != nil {
			return SignedOrder{}, err
		}
	} else {
		sig, err = b.Signer.SignHash(digest.Hex())
		if err != nil {
			return SignedOrder{}, err
		}
	}
	//log.Printf("signed msg: %s", sig)
	var side string
	if order.Side == 0 {
		side = "BUY"
	} else {
		side = "SELL"
	}
	return SignedOrder{
		Salt:          order.Salt.Int64(),
		Maker:         order.Maker,
		Signer:        order.Signer,
		Taker:         order.Taker,
		TokenID:       order.TokenID.String(),
		MakerAmount:   order.MakerAmount.String(),
		TakerAmount:   order.TakerAmount.String(),
		Expiration:    order.Expiration.String(),
		Nonce:         order.Nonce.String(),
		FeeRateBps:    order.FeeRateBps.String(),
		Side:          side,
		SignatureType: order.SignatureType,
		Signature:     sig,
	}, nil
}

func (b *UtilsOrderBuilder) buildOrder(data OrderData) (Order, error) {
	err := b.validateInputs(data)
	if err != nil {
		return Order{}, err
	}
	if data.Signer == constants.ZERO_ADDRESS {
		data.Signer = data.Maker
	}
	if b.Signer.SignerType() == signer.PrivateKey {
		signerAddr, err := b.Signer.GetPubkeyOfPrivateKey()
		if err != nil {
			return Order{}, err
		}
		if data.Signer != signerAddr {
			return Order{}, errors.New("signer does not match data.Signer")
		}
	} else if b.Signer.SignerType() == signer.Turnkey {
		if data.Signer != b.Option.TurnkeyAccount {
			return Order{}, errors.New("turnkeyAccount does not match data.Signer")
		}
	} else {
		return Order{}, errors.New("signer type is invalid")
	}

	if data.Expiration == "" {
		data.Expiration = "0"
	}
	tokenId, err := utils.MustBigInt(data.TokenID)
	if err != nil {
		return Order{}, err
	}
	makerAmt, err := utils.MustBigInt(data.MakerAmount)
	if err != nil {
		return Order{}, fmt.Errorf("invalid makerAmount: %w", err)
	}

	takerAmt, err := utils.MustBigInt(data.TakerAmount)
	if err != nil {
		return Order{}, fmt.Errorf("invalid takerAmount: %w", err)
	}

	nonce, err := strconv.ParseUint(data.Nonce, 10, 64)
	if err != nil {
		return Order{}, fmt.Errorf("invalid nonce: %w", err)
	}
	nonceBig := new(big.Int).SetUint64(nonce)

	expUint, err := strconv.ParseUint(data.Expiration, 10, 64)
	if err != nil {
		return Order{}, fmt.Errorf("invalid expiration: %w", err)
	}
	expBig := new(big.Int).SetUint64(expUint)

	feeRate, err := utils.MustBigInt(data.FeeRateBps)
	if err != nil {
		return Order{}, fmt.Errorf("invalid feeRateBps: %w", err)
	}

	if data.Side != 0 && data.Side != 1 {
		return Order{}, errors.New("side must be 0(BUY) or 1(SELL)")
	}

	side := uint8(data.Side)

	sigType := uint8(data.SignatureType)

	return Order{
		Salt:          b.GenerateSalt(),
		Maker:         data.Maker,
		Signer:        data.Signer,
		Taker:         data.Taker,
		TokenID:       tokenId,
		MakerAmount:   makerAmt,
		TakerAmount:   takerAmt,
		Expiration:    expBig,
		Nonce:         nonceBig,
		FeeRateBps:    feeRate,
		Side:          side,
		SignatureType: sigType,
	}, nil

}

func (b *UtilsOrderBuilder) buildOrderSignature(order Order) (string, error) {
	chainIdStr := strconv.Itoa(b.ChainId)
	typedData := eip712.TypedData{
		Types: map[string][]eip712.EIP712Type{
			"Order": {
				{Name: "salt", Type: "uint256"},
				{Name: "maker", Type: "address"},
				{Name: "signer", Type: "address"},
				{Name: "taker", Type: "address"},
				{Name: "tokenId", Type: "uint256"},
				{Name: "makerAmount", Type: "uint256"},
				{Name: "takerAmount", Type: "uint256"},
				{Name: "expiration", Type: "uint256"},
				{Name: "nonce", Type: "uint256"},
				{Name: "feeRateBps", Type: "uint256"},
				{Name: "side", Type: "uint256"},
				{Name: "signatureType", Type: "uint256"},
			},
		},
		PrimaryType: "Order",
		Domain: eip712.EIP712Domain{
			Name:              constants.PolyExchangeDomainName,
			Version:           "1",
			ChainID:           chainIdStr,
			VerifyingContract: b.ExchangeAddress.Hex(),
		},
		Message: order,
	}
	var err error
	var sig string
	typedDataHash, err := eip712.GetTypedDataHash(typedData)
	if err != nil {
		return "", err
	}
	if b.Signer.SignerType() == signer.Turnkey {
		sig, err = b.Signer.SignHashWithTurnkey(typedDataHash.String(), b.Option.TurnkeyAccount)
		if err != nil {
			return "", err
		}
	} else {
		sig, err = b.Signer.SignHash(typedDataHash.String())
		if err != nil {
			return "", err
		}
	}
	return utils.Prepend0x(sig), nil
}

func (b *UtilsOrderBuilder) validateInputs(data OrderData) error {

	if data.Maker == constants.ZERO_ADDRESS {
		return fmt.Errorf("maker is required")
	}

	if data.TokenID == "" {
		return fmt.Errorf("tokenId is required")
	}

	if data.MakerAmount == "" {
		return fmt.Errorf("makerAmount is required")
	}

	if data.TakerAmount == "" {
		return fmt.Errorf("takerAmount is required")
	}

	if data.Side != types.SideBuy.Int() && data.Side != types.SideSell.Int() {
		return fmt.Errorf("side must be UtilsBuy or UtilsSell")
	}

	feeRate, err := strconv.Atoi(data.FeeRateBps)
	if err != nil || feeRate < 0 {
		return fmt.Errorf("feeRateBps must be non-negative numeric string")
	}

	nonce, err := strconv.Atoi(data.Nonce)
	if err != nil || nonce < 0 {
		return fmt.Errorf("nonce must be non-negative numeric string")
	}

	expiration, err := strconv.ParseInt(data.Expiration, 10, 64)
	if err != nil || expiration < 0 {
		return fmt.Errorf("expiration must be non-negative numeric string")
	}

	switch data.SignatureType {
	case constants.EOA,
		constants.POLY_GNOSIS_SAFE,
		constants.POLY_PROXY:

	default:
		return fmt.Errorf("invalid signatureType")
	}

	return nil
}
