package builder

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ybina/polymarket-go/client/config"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/client/relayer/model"
	"github.com/ybina/polymarket-go/client/relayer/utils"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
)

const createProxyTypeStr = "CreateProxy(address paymentToken,uint256 payment,address paymentReceiver)"

func BuildSafeCreateTransactionRequest(
	signer *signer.Signer,
	args model.SafeCreateTransactionArgs,
	cfg config.ContractConfig,
	turnkeyAccount common.Address,
) (*model.TransactionRequest, error) {

	fromAddr := args.FromAddress

	safeAddr := Derive(fromAddr, cfg.SafeFactory)

	structHashHex, err := createSafeCreateStructHashHex(
		cfg.SafeFactory,
		args.ChainID,
		args.PaymentToken,
		args.Payment,
		args.PaymentReceiver,
	)
	if err != nil {
		return nil, err
	}

	sig, err := signer.SignHashWithTurnkey(structHashHex, turnkeyAccount)
	if err != nil {
		return nil, err
	}

	return &model.TransactionRequest{
		Type:        model.TransactionTypeSafeCreate,
		From:        fromAddr,
		To:          cfg.SafeFactory,
		ProxyWallet: safeAddr,
		Data:        "0x",
		Signature:   sig,
		SignatureParams: &model.SignatureParams{
			PaymentToken:    &args.PaymentToken,
			Payment:         &args.Payment,
			PaymentReceiver: &args.PaymentReceiver,
		},
	}, nil
}

func createSafeCreateStructHashHex(
	safeFactory common.Address,
	chainID types.Chain,
	paymentToken common.Address,
	payment string,
	paymentReceiver common.Address,
) (string, error) {

	paymentInt, ok := new(big.Int).SetString(payment, 10)
	if !ok {
		return "", fmt.Errorf("invalid payment: %s", payment)
	}

	typeHash := crypto.Keccak256Hash([]byte(createProxyTypeStr))

	enc := utils.ABIEncode(
		[]string{"bytes32", "address", "uint256", "address"},
		[]interface{}{typeHash, paymentToken, paymentInt, paymentReceiver},
	)

	structHash := crypto.Keccak256Hash(enc)

	domain := utils.EIP712Domain{
		Name:              constants.SAFE_FACTORY_NAME,
		ChainID:           big.NewInt(int64(chainID)),
		VerifyingContract: safeFactory,

		HasName:      true,
		HasChainID:   true,
		HasVerifying: true,
	}

	eip712Hash := utils.EIP712Hash(domain, structHash)

	return "0x" + hex.EncodeToString(eip712Hash[:]), nil
}
