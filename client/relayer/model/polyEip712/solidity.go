package polyEip712

import (
	"fmt"
	"regexp"
	"strconv"
)

func FromSolidityType(t string) (EIP712Type, error) {
	re := regexp.MustCompile(`([a-z]+)(\d+)?(\[(\d+)?\])?`)
	m := re.FindStringSubmatch(t)
	if m == nil {
		return nil, fmt.Errorf("invalid solidity type")
	}

	base := m[1]
	lenStr := m[2]
	isArray := m[3] != ""

	var typ EIP712Type
	switch base {
	case "uint":
		bits := 256
		if lenStr != "" {
			bits, _ = strconv.Atoi(lenStr)
		}
		typ = UintType{Bits: bits}
	case "string":
		typ = StringType{}
	case "address":
		typ = AddressType{}
	case "bool":
		typ = BoolType{}
	case "bytes":
		l := 0
		if lenStr != "" {
			l, _ = strconv.Atoi(lenStr)
		}
		typ = BytesType{Length: l}
	default:
		return nil, fmt.Errorf("unsupported type")
	}

	if isArray {
		return ArrayType{Member: typ}, nil
	}
	return typ, nil
}
