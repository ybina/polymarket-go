package encode

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ybina/polymarket-go/client/relayer/model"
	"github.com/ybina/polymarket-go/client/relayer/utils"
)

func CreateSafeMultisendTransaction(
	txns []model.SafeTransaction,
	safeMultisendAddress common.Address,
) model.SafeTransaction {

	var encodedTxns []byte

	for _, tx := range txns {

		toAddr := tx.To

		dataBytes, err := hex.DecodeString(utils.TrimHex(tx.Data))
		if err != nil {
			panic(fmt.Errorf("invalid tx data hex: %w", err))
		}

		valueInt, ok := new(big.Int).SetString(tx.Value, 10)
		if !ok {
			panic(fmt.Errorf("invalid tx value: %s", tx.Value))
		}

		packed := utils.EncodePacked(
			[]string{"uint8", "address", "uint256", "uint256", "bytes"},
			[]interface{}{
				uint8(tx.Operation),
				toAddr,
				valueInt,
				big.NewInt(int64(len(dataBytes))),
				dataBytes,
			},
		)

		encodedTxns = append(encodedTxns, packed...)
	}

	bytesType, _ := abi.NewType("bytes", "", nil)
	args := abi.Arguments{
		{Type: bytesType},
	}
	encodedData, err := args.Pack(encodedTxns)
	if err != nil {
		panic(err)
	}

	selector := crypto.Keccak256([]byte("multiSend(bytes)"))[:4]

	fullData := append(selector, encodedData...)

	return model.SafeTransaction{
		To:        safeMultisendAddress,
		Operation: model.DelegateCall,
		Data:      "0x" + hex.EncodeToString(fullData),
		Value:     "0",
	}
}
