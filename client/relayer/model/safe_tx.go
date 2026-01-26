package model

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ybina/polymarket-go/client/relayer/utils"
)

var safeTxTypeHash = crypto.Keccak256Hash([]byte(
	"SafeTx(address to,uint256 value,bytes data,uint8 operation,uint256 safeTxGas,uint256 baseGas,uint256 gasPrice,address gasToken,address refundReceiver,uint256 nonce)",
))

var safeDomainTypeHash = crypto.Keccak256Hash([]byte(
	"EIP712Domain(string name,uint256 chainId,address verifyingContract)",
))

type SafeTx struct {
	To             common.Address
	Value          *big.Int
	Data           []byte
	Operation      uint8
	SafeTxGas      *big.Int
	BaseGas        *big.Int
	GasPrice       *big.Int
	GasToken       common.Address
	RefundReceiver common.Address
	Nonce          *big.Int
}

func (tx *SafeTx) StructHash() common.Hash {

	dataHash := crypto.Keccak256Hash(tx.Data)

	return crypto.Keccak256Hash(
		utils.EncodePacked(
			[]string{
				"bytes32", // typehash
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
				safeTxTypeHash,
				tx.To,
				tx.Value,
				dataHash,
				tx.Operation,
				tx.SafeTxGas,
				tx.BaseGas,
				tx.GasPrice,
				tx.GasToken,
				tx.RefundReceiver,
				tx.Nonce,
			},
		),
	)
}

func MakeSafeDomain(
	verifyingContract common.Address,
	chainID int64,
) common.Hash {

	nameHash := crypto.Keccak256Hash([]byte("Gnosis Safe"))

	return crypto.Keccak256Hash(
		utils.EncodePacked(
			[]string{
				"bytes32",
				"bytes32",
				"uint256",
				"address",
			},
			[]interface{}{
				safeDomainTypeHash,
				nameHash,
				big.NewInt(chainID),
				verifyingContract,
			},
		),
	)
}

func (tx *SafeTx) GenerateStructHash(domainSeparator common.Hash) common.Hash {

	structHash := tx.StructHash()

	return crypto.Keccak256Hash(
		append(
			[]byte{0x19, 0x01},
			append(domainSeparator.Bytes(), structHash.Bytes()...)...,
		),
	)
}
