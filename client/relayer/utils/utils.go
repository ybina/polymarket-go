package utils

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func ABIEncode(types []string, values []interface{}) []byte {
	if len(types) != len(values) {
		panic("ABIEncode: types/values length mismatch")
	}
	args := make(abi.Arguments, 0, len(types))
	for _, t := range types {
		ty, err := abi.NewType(t, "", nil)
		if err != nil {
			panic(err)
		}
		args = append(args, abi.Argument{Type: ty})
	}
	b, err := args.Pack(values...)
	if err != nil {
		panic(fmt.Errorf("ABIEncode pack error: %w", err))
	}
	return b
}

func Prepend0x(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s
	}
	return "0x" + s
}

func TrimHex(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}

var _ = big.NewInt
var _ = common.Address{}

type EIP712Domain struct {
	Name              string
	Version           string
	ChainID           *big.Int
	VerifyingContract common.Address

	HasName      bool
	HasVersion   bool
	HasChainID   bool
	HasVerifying bool
}

func DomainTypeString(d EIP712Domain) string {
	fields := ""
	add := func(s string) {
		if fields == "" {
			fields = s
		} else {
			fields += "," + s
		}
	}
	if d.HasName {
		add("string name")
	}
	if d.HasVersion {
		add("string version")
	}
	if d.HasChainID {
		add("uint256 chainId")
	}
	if d.HasVerifying {
		add("address verifyingContract")
	}
	return "EIP712Domain(" + fields + ")"
}

func DomainSeparator(d EIP712Domain) ([32]byte, error) {
	typeStr := DomainTypeString(d)
	typeHash := crypto.Keccak256Hash([]byte(typeStr))

	types := []string{"bytes32"}
	values := []interface{}{typeHash}

	if d.HasName {
		types = append(types, "bytes32")
		values = append(values, crypto.Keccak256Hash([]byte(d.Name)))
	}
	if d.HasVersion {
		types = append(types, "bytes32")
		values = append(values, crypto.Keccak256Hash([]byte(d.Version)))
	}
	if d.HasChainID {
		types = append(types, "uint256")
		values = append(values, d.ChainID)
	}
	if d.HasVerifying {
		types = append(types, "address")
		values = append(values, d.VerifyingContract)
	}

	enc := ABIEncode(types, values)
	return crypto.Keccak256Hash(enc), nil
}

func EIP712Hash(domain EIP712Domain, messageStructHash [32]byte) [32]byte {
	ds, _ := DomainSeparator(domain)
	b := append([]byte{0x19, 0x01}, ds[:]...)
	b = append(b, messageStructHash[:]...)
	return crypto.Keccak256Hash(b)
}
