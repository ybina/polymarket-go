package builder

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ybina/polymarket-go/client/constants"
)

func getCreate2Address(
	bytecodeHashHex string,
	fromAddress string,
	salt []byte,
) (common.Address, error) {

	bytecodeHash := common.HexToHash(bytecodeHashHex)

	factoryAddr := common.HexToAddress(fromAddress)

	buf := []byte{0xff}
	buf = append(buf, factoryAddr.Bytes()...)
	buf = append(buf, salt...)
	buf = append(buf, bytecodeHash.Bytes()...)

	addressHash := crypto.Keccak256(buf)
	address := common.BytesToAddress(addressHash[12:])

	return address, nil
}

func Derive(addr common.Address, safeFactory common.Address) common.Address {
	if addr == constants.ZERO_ADDRESS {
		return constants.ZERO_ADDRESS
	}

	addressType, _ := abi.NewType("address", "", nil)
	args := abi.Arguments{
		{Type: addressType},
	}

	encoded, err := args.Pack(addr)
	if err != nil {
		panic(fmt.Errorf("abi encode address failed: %w", err))
	}

	salt := crypto.Keccak256(encoded)

	safeAddr, err := getCreate2Address(
		constants.SAFE_INIT_CODE_HASH,
		safeFactory.Hex(),
		salt,
	)
	if err != nil {
		panic(err)
	}

	return safeAddr
}
