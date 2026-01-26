package model

import (
	"encoding/hex"

	"github.com/ethereum/go-ethereum/crypto"
)

type EIP712Hashable interface {
	StructHash(domainHash []byte) ([]byte, error)
}

func GenerateStructHash(
	domainHash []byte,
	structHash []byte,
) string {

	prefix := []byte{0x19, 0x01}
	data := append(prefix, append(domainHash, structHash...)...)

	hash := crypto.Keccak256(data)
	return "0x" + hex.EncodeToString(hash)
}
