package order_builder

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/ybina/polymarket-go/client/clob/clob_types"
	"github.com/ybina/polymarket-go/client/clob/utils"
	"github.com/ybina/polymarket-go/client/clob/utils_order_builder"
	"github.com/ybina/polymarket-go/client/config"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
)

func NewOrderBuilder(signer *signer.Signer, sigType constants.SigType, funder common.Address) (*OrderBuilder, error) {
	if signer == nil {
		return nil, errors.New("NewOrderBuilder: signer cannot be nil")
	}
	return &OrderBuilder{
		Signer:  signer,
		SigType: sigType,
		Funder:  funder,
	}, nil
}

// validateAndGetRoundConfig checks TickSize and NegRisk are set and returns the matching RoundConfig.
func validateAndGetRoundConfig(options clob_types.PartialCreateOrderOptions) (*types.RoundConfig, error) {
	if options.TickSize == nil {
		return nil, errors.New("options.TickSize cannot be nil")
	}
	if options.NegRisk == nil {
		return nil, errors.New("options.NegRisk cannot be nil")
	}
	rc := types.GetRoundConfig(*options.TickSize)
	if rc == nil {
		return nil, errors.New("get round config error")
	}
	return rc, nil
}

// resolveSignerAddress returns the on-chain address that will appear in the order's signer field.
func resolveSignerAddress(signerHandler *signer.Signer, turnkeyAccount common.Address) (common.Address, error) {
	switch signerHandler.SignerType() {
	case signer.Turnkey:
		return turnkeyAccount, nil
	case signer.PrivateKey:
		return signerHandler.GetPubkeyOfPrivateKey()
	default:
		return common.Address{}, errors.New("signer type error")
	}
}

// resolveExchange selects standard or neg-risk exchange address.
func resolveExchange(contractConfig config.ContractConfig, negRisk bool) common.Address {
	if negRisk {
		return contractConfig.NegExchange
	}
	return contractConfig.Exchange
}

// adjustToDecimalPlaces rounds v down to maxPlaces decimal places, preferring rounding up first
// to minimise loss, then rounding down only if still too many places.
func adjustToDecimalPlaces(v decimal.Decimal, maxPlaces int) decimal.Decimal {
	if utils.DecimalPlaces(v) > maxPlaces {
		v = utils.RoundUp(v, maxPlaces+4)
		if utils.DecimalPlaces(v) > maxPlaces {
			v = utils.RoundDown(v, maxPlaces)
		}
	}
	return v
}

// buildSignedOrder resolves the contract addresses and delegates to UtilsOrderBuilder.
func (b *OrderBuilder) buildSignedOrder(
	signerHandler *signer.Signer,
	data utils_order_builder.OrderData,
	options clob_types.PartialCreateOrderOptions,
) (utils_order_builder.SignedOrder, error) {
	contractConfig, err := config.GetContractConfig(types.Chain(signerHandler.ChainID()))
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	exchange := resolveExchange(contractConfig, *options.NegRisk)
	utilsOB, err := utils_order_builder.NewUtilsOrderBuilder(
		exchange, int(signerHandler.ChainID()), signerHandler,
		utils_order_builder.Option{TurnkeyAccount: options.TurnkeyAccount},
	)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	return utilsOB.BuildSignedOrder(data)
}

func (b *OrderBuilder) CreateOrder(signerHandler *signer.Signer, args clob_types.OrderArgs, options clob_types.PartialCreateOrderOptions) (utils_order_builder.SignedOrder, error) {
	roundConfig, err := validateAndGetRoundConfig(options)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	sideInt, makerAmount, takerAmount, err := b.GetOrderAmounts(args.Side, args.Size, args.Price, *roundConfig)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	signerAddr, err := resolveSignerAddress(signerHandler, options.TurnkeyAccount)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	data := utils_order_builder.OrderData{
		Maker:         b.Funder,
		Taker:         args.Taker,
		TokenID:       args.TokenID,
		MakerAmount:   makerAmount,
		TakerAmount:   takerAmount,
		Side:          sideInt,
		FeeRateBps:    strconv.Itoa(args.FeeRateBps),
		Nonce:         strconv.Itoa(args.Nonce),
		Signer:        signerAddr,
		Expiration:    strconv.Itoa(args.Expiration),
		SignatureType: b.SigType,
	}
	return b.buildSignedOrder(signerHandler, data, options)
}

func (b *OrderBuilder) CreateMarketOrder(signerHandler *signer.Signer, args clob_types.MarketOrderArgs, options clob_types.PartialCreateOrderOptions) (utils_order_builder.SignedOrder, error) {
	roundConfig, err := validateAndGetRoundConfig(options)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	side, makerAmount, takerAmount, err := b.GetMarketOrderAmounts(args.Side, args.Amount, args.Price, *roundConfig)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	signerAddr, err := resolveSignerAddress(signerHandler, options.TurnkeyAccount)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	data := utils_order_builder.OrderData{
		Maker:         b.Funder,
		Taker:         args.Taker,
		TokenID:       args.TokenID,
		MakerAmount:   makerAmount,
		TakerAmount:   takerAmount,
		Side:          side,
		FeeRateBps:    strconv.Itoa(args.FeeRateBps),
		Nonce:         strconv.Itoa(args.Nonce),
		Signer:        signerAddr,
		Expiration:    "0",
		SignatureType: b.SigType,
	}
	return b.buildSignedOrder(signerHandler, data, options)
}

func (b *OrderBuilder) GetOrderAmounts(
	side types.Side,
	size decimal.Decimal,
	price decimal.Decimal,
	roundConfig types.RoundConfig,
) (sideInt int, makerAmount string, takerAmount string, err error) {

	roundedPrice := utils.RoundNormal(price, roundConfig.Price)

	switch side {
	case types.SideBuy:
		takerAmt := utils.RoundDown(size, roundConfig.Size)
		makerRaw := adjustToDecimalPlaces(takerAmt.Mul(roundedPrice), roundConfig.Amount)
		return side.Int(), utils.ToTokenDecimals(makerRaw), utils.ToTokenDecimals(takerAmt), nil

	case types.SideSell:
		makerAmt := utils.RoundDown(size, roundConfig.Size)
		takerRaw := adjustToDecimalPlaces(makerAmt.Mul(roundedPrice), int(roundConfig.Amount))
		return side.Int(), utils.ToTokenDecimals(makerAmt), utils.ToTokenDecimals(takerRaw), nil
	}

	return 0, "", "", fmt.Errorf("invalid side: must be BUY or SELL")
}

func (b *OrderBuilder) GetMarketOrderAmounts(
	side types.Side,
	amount decimal.Decimal,
	price decimal.Decimal,
	roundConfig types.RoundConfig,
) (sideInt int, makerAmount string, takerAmount string, err error) {

	roundedPrice := utils.RoundNormal(price, roundConfig.Price)

	switch side {
	case types.SideBuy:
		makerAmt := utils.RoundDown(amount, roundConfig.Size)
		if roundedPrice.IsZero() {
			return 0, "", "", fmt.Errorf("price cannot be 0")
		}
		takerRaw := adjustToDecimalPlaces(makerAmt.Div(roundedPrice), roundConfig.Amount)
		return side.Int(), utils.ToTokenDecimals(makerAmt), utils.ToTokenDecimals(takerRaw), nil

	case types.SideSell:
		makerAmt := utils.RoundDown(amount, roundConfig.Size)
		takerRaw := adjustToDecimalPlaces(makerAmt.Mul(roundedPrice), int(roundConfig.Amount))
		return side.Int(), utils.ToTokenDecimals(makerAmt), utils.ToTokenDecimals(takerRaw), nil
	}

	return 0, "", "", fmt.Errorf("invalid side: must be BUY or SELL")
}
