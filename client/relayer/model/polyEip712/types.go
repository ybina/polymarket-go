package polyEip712

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type UintType struct {
	Bits int
}

func (u UintType) TypeName() string {
	return fmt.Sprintf("uint%d", u.Bits)
}

func (u UintType) Encode(value any) ([32]byte, error) {
	v := new(big.Int)
	switch t := value.(type) {
	case int:
		v.SetInt64(int64(t))
	case int64:
		v.SetInt64(t)
	case *big.Int:
		v.Set(t)
	default:
		return [32]byte{}, fmt.Errorf("invalid uint value")
	}
	return leftPad32(v.Bytes()), nil
}

type StringType struct{}

func (s StringType) TypeName() string { return "string" }

func (s StringType) Encode(value any) ([32]byte, error) {
	return keccak([]byte(value.(string))), nil
}

type AddressType struct{}

func (a AddressType) TypeName() string { return "address" }

func (a AddressType) Encode(value any) ([32]byte, error) {
	addr := common.HexToAddress(value.(string))
	return leftPad32(addr.Bytes()), nil
}

type BoolType struct{}

func (b BoolType) TypeName() string { return "bool" }

func (b BoolType) Encode(value any) ([32]byte, error) {
	if value.(bool) {
		return leftPad32([]byte{1}), nil
	}
	return [32]byte{}, nil
}

type BytesType struct {
	Length int // 0 = dynamic
}

func (b BytesType) TypeName() string {
	if b.Length == 0 {
		return "bytes"
	}
	return fmt.Sprintf("bytes%d", b.Length)
}

func (b BytesType) Encode(value any) ([32]byte, error) {
	data := common.FromHex(value.(string))
	if b.Length == 0 {
		return keccak(data), nil
	}
	if len(data) > b.Length {
		return [32]byte{}, fmt.Errorf("bytes overflow")
	}
	return rightPad32(data), nil
}
