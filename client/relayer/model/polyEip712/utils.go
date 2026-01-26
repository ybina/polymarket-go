package polyEip712

import "github.com/ethereum/go-ethereum/crypto"

func keccak(data []byte) [32]byte {
	return crypto.Keccak256Hash(data)
}

func leftPad32(b []byte) [32]byte {
	var out [32]byte
	copy(out[32-len(b):], b)
	return out
}

func rightPad32(b []byte) [32]byte {
	var out [32]byte
	copy(out[:], b)
	return out
}
