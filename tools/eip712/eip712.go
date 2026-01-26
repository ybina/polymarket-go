package eip712

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ybina/polymarket-go/client/clob/clob_types"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/client/signer"
)

const (
	MSG_TO_SIGN = "This message attests that I control the given wallet"
)

type EIP712Domain struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	ChainID           string `json:"chainId"`
	Salt              string `json:"salt,omitempty"`
	VerifyingContract string `json:"verifyingContract,omitempty"`
}

type EIP712Type struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ClobAuthData struct {
	Address   string `json:"address"`
	Timestamp string `json:"timestamp"`
	Nonce     uint64 `json:"nonce"`
	Message   string `json:"message"`
}

type TypedData struct {
	Types       map[string][]EIP712Type `json:"types"`
	PrimaryType string                  `json:"primaryType"`
	Domain      EIP712Domain            `json:"domain"`
	Message     interface{}             `json:"message"`
}

func BuildClobEip712Signature(signerHandler *signer.Signer, option clob_types.ClobOption, chainID string, timestamp int64, nonce uint64) (string, error) {
	var address string
	if signerHandler.SignerType() == signer.Turnkey {
		if option.TurnkeyAccount == constants.ZERO_ADDRESS {
			return address, errors.New("turnkey account is required")
		}
		address = option.TurnkeyAccount.Hex()
	} else if signerHandler.SignerType() == signer.PrivateKey {
		address = signerHandler.Address()
	} else {
		return "", errors.New("invalid signer type")
	}

	domain := EIP712Domain{
		Name:    "ClobAuthDomain",
		Version: "1",
		ChainID: chainID,
	}

	types := map[string][]EIP712Type{
		"ClobAuth": {
			{Name: "address", Type: "address"},
			{Name: "timestamp", Type: "string"},
			{Name: "nonce", Type: "uint256"},
			{Name: "message", Type: "string"},
		},
	}

	message := ClobAuthData{
		Address:   address,
		Timestamp: fmt.Sprintf("%d", timestamp),
		Nonce:     nonce,
		Message:   MSG_TO_SIGN,
	}

	domainSeparator, err := getDomainSeparator(domain)
	if err != nil {
		return "", fmt.Errorf("failed to get domain separator: %w", err)
	}

	typeHash, err := getTypeHash(types["ClobAuth"])
	if err != nil {
		return "", fmt.Errorf("failed to get type hash: %w", err)
	}

	encodeData, err := encodeClobAuthData(message)
	if err != nil {
		return "", fmt.Errorf("failed to encode data: %w", err)
	}

	structHash := crypto.Keccak256Hash(append(typeHash.Bytes(), encodeData...))

	hash := crypto.Keccak256Hash(
		append(append([]byte("\x19\x01"), domainSeparator.Bytes()...), structHash.Bytes()...),
	)

	if signerHandler.SignerType() == signer.Turnkey {
		return signerHandler.SignHashWithTurnkey(hash.String(), option.TurnkeyAccount)
	} else if signerHandler.SignerType() == signer.PrivateKey {
		return signerHandler.SignHash(hash.String())
	} else {
		return "", errors.New("invalid signer type")
	}

}

func getDomainSeparator(domain EIP712Domain) (common.Hash, error) {
	typeHash := crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId)"))

	nameHash := crypto.Keccak256Hash([]byte(domain.Name))
	versionHash := crypto.Keccak256Hash([]byte(domain.Version))
	chainId := new(big.Int)
	chainId.SetString(domain.ChainID, 10)
	chainIdBytes := make([]byte, 32)
	chainId.FillBytes(chainIdBytes)
	data := append(typeHash.Bytes(), nameHash.Bytes()...)
	data = append(data, versionHash.Bytes()...)
	data = append(data, chainIdBytes...)

	return crypto.Keccak256Hash(data), nil
}

func getTypeHash(types []EIP712Type) (common.Hash, error) {
	typeString := "ClobAuth(address address,string timestamp,uint256 nonce,string message)"
	return crypto.Keccak256Hash([]byte(typeString)), nil
}

func encodeClobAuthData(data ClobAuthData) ([]byte, error) {
	address := common.HexToAddress(data.Address)
	nonce := new(big.Int).SetUint64(data.Nonce)

	addressBytes := make([]byte, 32)
	copy(addressBytes[12:], address.Bytes()) // address is 20 bytes, so left-pad with 12 zeros

	timestampHash := crypto.Keccak256Hash([]byte(data.Timestamp))

	nonceBytes := make([]byte, 32)
	nonce.FillBytes(nonceBytes)

	messageHash := crypto.Keccak256Hash([]byte(data.Message))

	encodedData := append(addressBytes, timestampHash.Bytes()...)
	encodedData = append(encodedData, nonceBytes...)
	encodedData = append(encodedData, messageHash.Bytes()...)

	return encodedData, nil
}

func GetTypedDataHash(typedData TypedData) (common.Hash, error) {

	domainSeparator, err := getDomainSeparator(typedData.Domain)
	if err != nil {
		return common.Hash{}, err
	}

	messageHash, err := getMessageHash(typedData)
	if err != nil {
		return common.Hash{}, err
	}

	prefix := []byte("\x19\x01")

	data := append(prefix, domainSeparator.Bytes()...)
	data = append(data, messageHash.Bytes()...)

	return crypto.Keccak256Hash(data), nil
}

func getMessageHash(typedData TypedData) (common.Hash, error) {
	messageBytes, err := json.Marshal(typedData.Message)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to marshal message: %w", err)
	}

	return crypto.Keccak256Hash(messageBytes), nil
}
