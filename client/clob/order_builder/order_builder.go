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

func (b *OrderBuilder) CreateOrder(signerHandler *signer.Signer, args clob_types.OrderArgs, options clob_types.PartialCreateOrderOptions) (utils_order_builder.SignedOrder, error) {
	if options.TickSize == nil {
		return utils_order_builder.SignedOrder{}, errors.New("options.TickSize cannot be nil")
	}
	if options.NegRisk == nil {
		return utils_order_builder.SignedOrder{}, errors.New("options.NegRisk cannot be nil")
	}
	roundConfig := types.GetRoundConfig(*options.TickSize)
	if roundConfig == nil {
		return utils_order_builder.SignedOrder{}, errors.New("get round config error")
	}
	sideInt, makerAmount, takerAmount, err := b.GetOrderAmounts(args.Side, args.Size, args.Price, *roundConfig)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	var signerAddr common.Address
	if signerHandler.SignerType() == signer.Turnkey {
		signerAddr = options.TurnkeyAccount
	} else if signerHandler.SignerType() == signer.PrivateKey {
		signerAddr, err = signerHandler.GetPubkeyOfPrivateKey()
		if err != nil {
			return utils_order_builder.SignedOrder{}, err
		}
	} else {
		return utils_order_builder.SignedOrder{}, errors.New("signer type error")
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
	contractConfig, err := config.GetContractConfig(types.Chain(signerHandler.ChainID()))
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	var exchange common.Address
	if *options.NegRisk {
		exchange = contractConfig.NegExchange
	} else {
		exchange = contractConfig.Exchange
	}
	utilsOption := utils_order_builder.Option{TurnkeyAccount: options.TurnkeyAccount}
	utilsOrderBuilder, err := utils_order_builder.NewUtilsOrderBuilder(exchange, int(signerHandler.ChainID()), signerHandler, utilsOption)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	return utilsOrderBuilder.BuildSignedOrder(data)

}

func (b *OrderBuilder) CreateMarketOrder(signerHandler *signer.Signer, args clob_types.MarketOrderArgs, options clob_types.PartialCreateOrderOptions) (utils_order_builder.SignedOrder, error) {
	if options.TickSize == nil {
		return utils_order_builder.SignedOrder{}, errors.New("options.TickSize cannot be nil")
	}
	if options.NegRisk == nil {
		return utils_order_builder.SignedOrder{}, errors.New("options.NegRisk cannot be nil")
	}
	roundConfig := types.GetRoundConfig(*options.TickSize)
	if roundConfig == nil {
		return utils_order_builder.SignedOrder{}, errors.New("get round config error")
	}
	side, makerAmount, takerAmount, err := b.GetMarketOrderAmounts(args.Side, args.Amount, args.Price, *roundConfig)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	var signerAddr common.Address
	if signerHandler.SignerType() == signer.Turnkey {
		signerAddr = options.TurnkeyAccount
	} else if signerHandler.SignerType() == signer.PrivateKey {
		signerAddr, err = signerHandler.GetPubkeyOfPrivateKey()
		if err != nil {
			return utils_order_builder.SignedOrder{}, err
		}
	} else {
		return utils_order_builder.SignedOrder{}, errors.New("signer type error")
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
	contractConfig, err := config.GetContractConfig(types.Chain(signerHandler.ChainID()))
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	var exchange common.Address
	if *options.NegRisk {
		exchange = contractConfig.NegExchange
	} else {
		exchange = contractConfig.Exchange
	}
	utilsOption := utils_order_builder.Option{TurnkeyAccount: options.TurnkeyAccount}
	utilsOrderBuilder, err := utils_order_builder.NewUtilsOrderBuilder(exchange, int(signerHandler.ChainID()), signerHandler, utilsOption)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	return utilsOrderBuilder.BuildSignedOrder(data)

}

func (b *OrderBuilder) GetOrderAmounts(
	side types.Side,
	size decimal.Decimal,
	price decimal.Decimal,
	roundConfig types.RoundConfig,
) (sideInt int, makerAmount string, takerAmount string, err error) {

	roundedPrice := utils.RoundNormal(price, roundConfig.Price)

	if side == types.SideBuy {
		takerAmt := utils.RoundDown(size, roundConfig.Size)

		makerRaw := takerAmt.Mul(roundedPrice)

		if utils.DecimalPlaces(makerRaw) > roundConfig.Amount {
			makerRaw = utils.RoundUp(makerRaw, roundConfig.Amount+4)

			if utils.DecimalPlaces(makerRaw) > roundConfig.Amount {
				makerRaw = utils.RoundDown(makerRaw, roundConfig.Amount)
			}
		}

		makerAmount = utils.ToTokenDecimals(makerRaw)
		takerAmount = utils.ToTokenDecimals(takerAmt)

		return side.Int(), makerAmount, takerAmount, nil
	}

	if side == types.SideSell {
		makerAmt := utils.RoundDown(size, roundConfig.Size)

		takerRaw := makerAmt.Mul(roundedPrice)

		if utils.DecimalPlaces(takerRaw) > int(roundConfig.Amount) {
			takerRaw = utils.RoundUp(takerRaw, int(roundConfig.Amount)+4)

			if utils.DecimalPlaces(takerRaw) > int(roundConfig.Amount) {
				takerRaw = utils.RoundDown(takerRaw, int(roundConfig.Amount))
			}
		}

		makerAmount = utils.ToTokenDecimals(makerAmt)
		takerAmount = utils.ToTokenDecimals(takerRaw)

		return side.Int(), makerAmount, takerAmount, nil
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

	if side == types.SideBuy {
		makerAmt := utils.RoundDown(amount, roundConfig.Size)

		if roundedPrice.IsZero() {
			return 0, "", "", fmt.Errorf("price cannot be 0")
		}
		takerRaw := makerAmt.Div(roundedPrice)

		if utils.DecimalPlaces(takerRaw) > roundConfig.Amount {
			takerRaw = utils.RoundUp(takerRaw, roundConfig.Amount+4)
			if utils.DecimalPlaces(takerRaw) > roundConfig.Amount {
				takerRaw = utils.RoundDown(takerRaw, roundConfig.Amount)
			}
		}

		makerAmount = utils.ToTokenDecimals(makerAmt)
		takerAmount = utils.ToTokenDecimals(takerRaw)

		return side.Int(), makerAmount, takerAmount, nil
	}
	if side == types.SideSell {
		makerAmt := utils.RoundDown(amount, roundConfig.Size)

		takerRaw := makerAmt.Mul(roundedPrice)

		if utils.DecimalPlaces(takerRaw) > int(roundConfig.Amount) {
			takerRaw = utils.RoundUp(takerRaw, int(roundConfig.Amount)+4)
			if utils.DecimalPlaces(takerRaw) > int(roundConfig.Amount) {
				takerRaw = utils.RoundDown(takerRaw, int(roundConfig.Amount))
			}
		}

		makerAmount = utils.ToTokenDecimals(makerAmt)
		takerAmount = utils.ToTokenDecimals(takerRaw)

		return side.Int(), makerAmount, takerAmount, nil
	}

	return 0, "", "", fmt.Errorf("invalid side: must be BUY or SELL")
}
