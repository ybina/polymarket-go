package signer

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/tools/utils"
	"github.com/ybina/polymarket-go/turnkey"
)

type SignerType int

const (
	PrivateKey SignerType = iota
	Turnkey
)

type SignerConfig struct {
	SignerType       SignerType
	PrivateKeyConfig *PrivateKeyClient
	TurnkeyConfig    *turnkey.Config
	ChainID          int64
}
type PrivateKeyClient struct {
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
}

type Signer struct {
	signerType       SignerType
	privateKeyClient PrivateKeyClient
	turnkeyClient    turnkey.Client
	chainID          int64
}

func NewSigner(config SignerConfig) (*Signer, error) {

	if config.SignerType == PrivateKey {

		pub := crypto.PubkeyToAddress(config.PrivateKeyConfig.PrivateKey.PublicKey)

		return &Signer{
			signerType: config.SignerType,
			privateKeyClient: PrivateKeyClient{
				PrivateKey: config.PrivateKeyConfig.PrivateKey,
				Address:    pub,
			},
			chainID: config.ChainID,
		}, nil
	}
	if config.SignerType == Turnkey {
		if config.TurnkeyConfig == nil {
			return nil, errors.New("turnkey config is empty")
		}
		tkClient, err := turnkey.NewTurnKeyService(*config.TurnkeyConfig)
		if err != nil {
			return nil, err
		}
		return &Signer{
			signerType:    config.SignerType,
			turnkeyClient: tkClient,
			chainID:       config.ChainID,
		}, nil
	}
	return nil, errors.New("invalid signerType")

}

func (s *Signer) Address() string {
	if s.signerType == PrivateKey {
		return s.privateKeyClient.Address.Hex()
	}
	if s.signerType == Turnkey {
		acc, err := s.turnkeyClient.GetAccounts(0)
		if err != nil {
			return ""
		}
		return acc[0].Address
	}
	return ""
}

func (s *Signer) GetPubkeyOfPrivateKey() (common.Address, error) {
	if s.signerType == PrivateKey {
		return s.privateKeyClient.Address, nil
	}
	return common.Address{}, errors.New("invalid signerType")
}

func (s *Signer) SignerType() SignerType {
	return s.signerType
}

func (s *Signer) ChainID() int64 {
	return s.chainID
}

// decodeHashHex decodes a hex string (with or without 0x prefix) into exactly 32 bytes.
func decodeHashHex(hashHex string) ([]byte, error) {
	hashBytes, err := hex.DecodeString(utils.TrimHex(hashHex))
	if err != nil {
		return nil, err
	}
	if len(hashBytes) != 32 {
		return nil, errors.New("hash must be 32 bytes")
	}
	return hashBytes, nil
}

// SignHash signs a raw 32-byte hash with the private key.
func (s *Signer) SignHash(hashHex string) (string, error) {
	if s.signerType != PrivateKey {
		return "", errors.New("signer type not match")
	}
	hashBytes, err := decodeHashHex(hashHex)
	if err != nil {
		return "", err
	}
	sig, err := crypto.Sign(hashBytes, s.privateKeyClient.PrivateKey)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(sig), nil
}

// SignHashWithTurnkey signs a raw 32-byte hash via the Turnkey service.
func (s *Signer) SignHashWithTurnkey(hashHex string, turnkeyAccount common.Address) (string, error) {
	if s.signerType != Turnkey {
		return "", errors.New("signer type not match")
	}
	if turnkeyAccount == constants.ZERO_ADDRESS {
		return "", errors.New("turnkey account is empty")
	}
	hashBytes, err := decodeHashHex(hashHex)
	if err != nil {
		return "", err
	}
	sig, err := s.turnkeyClient.Sign(turnkeyAccount.Hex(), base64.StdEncoding.EncodeToString(hashBytes))
	if err != nil {
		return "", err
	}
	sig = utils.TrimHex(sig)
	if len(sig) != 130 { // 65 bytes hex = 130 chars
		return "", fmt.Errorf("turnkey signature hex length=%d (want 130). sig=%s", len(sig), sig)
	}
	return utils.Prepend0x(sig), nil
}

// SignEIP712StructHash applies personal_sign (eth_sign) over a 32-byte struct hash.
func (s *Signer) SignEIP712StructHash(structHashHex string, turnkeyAccount common.Address) (string, error) {
	hashBytes, err := decodeHashHex(structHashHex)
	if err != nil {
		return "", err
	}
	msg := accounts.TextHash(hashBytes)
	switch s.signerType {
	case PrivateKey:
		sig, err := crypto.Sign(msg, s.privateKeyClient.PrivateKey)
		if err != nil {
			return "", err
		}
		return "0x" + hex.EncodeToString(sig), nil
	case Turnkey:
		if turnkeyAccount == constants.ZERO_ADDRESS {
			return "", errors.New("turnkey account is empty")
		}
		sig, err := s.turnkeyClient.Sign(turnkeyAccount.Hex(), base64.StdEncoding.EncodeToString(msg))
		if err != nil {
			return "", err
		}
		return utils.Prepend0x(sig), nil
	default:
		return "", errors.New("invalid signerType")
	}
}
