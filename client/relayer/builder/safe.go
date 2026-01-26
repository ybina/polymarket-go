package builder

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ybina/polymarket-go/client/config"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/client/relayer/encode"
	"github.com/ybina/polymarket-go/client/relayer/model"
	"github.com/ybina/polymarket-go/client/relayer/utils"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
)

const safeTxTypeStr = "SafeTx(address to,uint256 value,bytes data,uint8 operation,uint256 safeTxGas,uint256 baseGas,uint256 gasPrice,address gasToken,address refundReceiver,uint256 nonce)"

func BuildSafeTransactionRequest(
	signerHandler *signer.Signer,
	args model.SafeTransactionArgs,
	cfg config.ContractConfig,
	metadata string,
	turnkeyAccount common.Address,
) (*model.TransactionRequest, error) {

	factory := cfg.SafeFactory

	safeAddr := Derive(args.FromAddress, factory)

	tx := aggregateTransaction(args.Transactions, cfg.SafeMultisend)

	safeTxnGas := "0"
	baseGas := "0"
	gasPrice := "0"
	gasToken := constants.ZERO_ADDRESS
	refundReceiver := constants.ZERO_ADDRESS

	structHashHex, err := createSafeStructHashHex(
		args.ChainID,
		safeAddr,
		tx.To,
		tx.Value,
		tx.Data,
		uint8(tx.Operation),
		safeTxnGas,
		baseGas,
		gasPrice,
		gasToken,
		refundReceiver,
		fmt.Sprintf("%d", args.Nonce),
	)
	if err != nil {
		return nil, err
	}

	var sig string

	if signerHandler.SignerType() == signer.Turnkey {
		sig, err = signerHandler.SignEIP712StructHash(structHashHex, turnkeyAccount)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("only support TURNKEY, please wait develop")
	}

	packedSig, err := splitAndPackSig(sig)
	if err != nil {
		return nil, err
	}

	if metadata == "" {
		metadata = ""
	}
	nonceStr := strconv.Itoa(int(args.Nonce))
	op := fmt.Sprintf("%d", tx.Operation)
	req := &model.TransactionRequest{
		Type:        model.TransactionTypeSafe,
		From:        args.FromAddress,
		To:          tx.To,
		ProxyWallet: safeAddr,
		Value:       &tx.Value,
		Data:        utils.Prepend0x(tx.Data),
		Nonce:       &nonceStr,
		Signature:   packedSig,
		SignatureParams: &model.SignatureParams{
			GasPrice:       &gasPrice,
			Operation:      &op,
			SafeTxnGas:     &safeTxnGas,
			BaseGas:        &baseGas,
			GasToken:       &gasToken,
			RefundReceiver: &refundReceiver,
		},
		Metadata: &metadata,
	}

	return req, nil
}

func aggregateTransaction(txns []model.SafeTransaction, multisend common.Address) model.SafeTransaction {
	if len(txns) == 1 {
		return txns[0]
	}
	return encode.CreateSafeMultisendTransaction(txns, multisend)
}

func createSafeStructHashHex(
	chainID types.Chain,
	safe common.Address,
	to common.Address,
	value string,
	dataHex string,
	operation uint8,
	safeTxGas string,
	baseGas string,
	gasPrice string,
	gasToken common.Address,
	refundReceiver common.Address,
	nonce string,
) (string, error) {

	valueInt, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return "", fmt.Errorf("invalid value: %s", value)
	}
	safeTxGasInt, ok := new(big.Int).SetString(safeTxGas, 10)
	if !ok {
		return "", fmt.Errorf("invalid safeTxGas: %s", safeTxGas)
	}
	baseGasInt, ok := new(big.Int).SetString(baseGas, 10)
	if !ok {
		return "", fmt.Errorf("invalid baseGas: %s", baseGas)
	}
	gasPriceInt, ok := new(big.Int).SetString(gasPrice, 10)
	if !ok {
		return "", fmt.Errorf("invalid gasPrice: %s", gasPrice)
	}
	nonceInt, ok := new(big.Int).SetString(nonce, 10)
	if !ok {
		return "", fmt.Errorf("invalid nonce: %s", nonce)
	}

	dataBytes, err := hex.DecodeString(utils.TrimHex(dataHex))
	if err != nil {
		return "", fmt.Errorf("invalid data hex: %w", err)
	}
	dataHash := crypto.Keccak256Hash(dataBytes) // EIP712 bytes field is hashed

	typeHash := crypto.Keccak256Hash([]byte(safeTxTypeStr))

	enc := utils.ABIEncode(
		[]string{
			"bytes32",
			"address",
			"uint256",
			"bytes32",
			"uint8",
			"uint256",
			"uint256",
			"uint256",
			"address",
			"address",
			"uint256",
		},
		[]interface{}{
			typeHash,
			to,
			valueInt,
			dataHash,
			operation,
			safeTxGasInt,
			baseGasInt,
			gasPriceInt,
			gasToken,
			refundReceiver,
			nonceInt,
		},
	)

	messageStructHash := crypto.Keccak256Hash(enc)

	domain := utils.EIP712Domain{
		Name:              "",
		Version:           "",
		ChainID:           big.NewInt(int64(chainID)),
		VerifyingContract: safe,
		HasName:           false,
		HasVersion:        false,
		HasChainID:        true,
		HasVerifying:      true,
	}

	eip712Hash := utils.EIP712Hash(domain, messageStructHash)

	return "0x" + hex.EncodeToString(eip712Hash[:]), nil
}

func splitAndPackSig(sigHex string) (string, error) {
	sigBytes, err := hex.DecodeString(utils.TrimHex(sigHex))
	if err != nil {
		return "", err
	}
	if len(sigBytes) != 65 {
		return "", fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:64])
	vRaw := sigBytes[64]

	var v uint8
	switch vRaw {
	case 0, 1:
		v = vRaw + 31
	case 27, 28:
		v = vRaw + 4
	default:
		return "", fmt.Errorf("invalid v: %d", vRaw)
	}

	packed := utils.EncodePacked(
		[]string{"uint256", "uint256", "uint8"},
		[]interface{}{r, s, v},
	)
	return "0x" + hex.EncodeToString(packed), nil
}

func splitAndPackSig2(sigHex string) (string, error) {
	sigBytes, err := hex.DecodeString(utils.TrimHex(sigHex))
	if err != nil {
		return "", err
	}
	if len(sigBytes) != 65 {
		return "", fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:64])
	vRaw := sigBytes[64]

	if vRaw >= 27 {
		vRaw -= 27
	}
	if vRaw != 0 && vRaw != 1 {
		return "", fmt.Errorf("invalid v after normalize: %d", vRaw)
	}

	v := vRaw + 31

	packed := utils.EncodePacked(
		[]string{"uint256", "uint256", "uint8"},
		[]interface{}{r, s, v},
	)
	return "0x" + hex.EncodeToString(packed), nil
}

var _ = accounts.TextHash
