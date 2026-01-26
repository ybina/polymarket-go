package utils

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func EncodePacked(types []string, values []interface{}) []byte {
	if len(types) != len(values) {
		panic("types and values length mismatch")
	}

	var out []byte

	for i, t := range types {
		v := values[i]

		switch t {

		case "bytes32":
			b, ok := v.(common.Hash)
			if !ok {
				panic("bytes32 expects common.Hash")
			}
			out = append(out, b.Bytes()...)

		case "address":
			addr, ok := v.(common.Address)
			if !ok {
				panic("address expects common.AddressHex")
			}
			out = append(out, addr.Bytes()...)

		case "uint256":
			bi, ok := v.(*big.Int)
			if !ok {
				panic("uint256 expects *big.Int")
			}
			out = append(out, common.LeftPadBytes(bi.Bytes(), 32)...)

		case "uint8":
			switch x := v.(type) {
			case uint8:
				out = append(out, x)
			case int:
				out = append(out, byte(x))
			case *big.Int:
				out = append(out, byte(x.Uint64()))
			default:
				panic("uint8 expects uint8 / int / *big.Int")
			}

		case "bytes":
			b, ok := v.([]byte)
			if !ok {
				panic("bytes expects []byte")
			}
			out = append(out, b...)

		default:
			panic(fmt.Sprintf("unsupported packed type: %s", t))
		}
	}

	return out
}
