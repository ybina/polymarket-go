package headers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ybina/polymarket-go/client/clob/clob_types"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
	"github.com/ybina/polymarket-go/tools/eip712"
	"github.com/ybina/polymarket-go/tools/hmac"
)

func CreateL1Headers(signerHandler *signer.Signer, option clob_types.ClobOption, chainID types.Chain, nonce *uint64, timestamp *int64) (*types.L1PolyHeader, error) {

	ts := time.Now().Unix()
	if timestamp != nil {
		ts = *timestamp
	}

	var n uint64 = 0
	if nonce != nil {
		n = *nonce
	}
	chainIdStr := strconv.Itoa(int(chainID))
	sig, err := eip712.BuildClobEip712Signature(signerHandler, option, chainIdStr, ts, n)
	if err != nil {
		return nil, fmt.Errorf("failed to build EIP712 signature: %w", err)
	}

	var address string
	if signerHandler.SignerType() == signer.Turnkey {
		address = option.TurnkeyAccount.Hex()
	} else if signerHandler.SignerType() == signer.PrivateKey {
		pub, err := signerHandler.GetPubkeyOfPrivateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get public key of private key: %w", err)
		}
		address = pub.Hex()

	} else {
		return nil, fmt.Errorf("invalid signer type: %s", strconv.Itoa(int(signerHandler.SignerType())))
	}
	headers := &types.L1PolyHeader{
		POLYAddress:   address,
		POLYSignature: sig,
		POLYTimestamp: strconv.FormatInt(ts, 10),
		POLYNonce:     strconv.FormatUint(n, 10),
	}

	return headers, nil
}

func CreateL2Headers(signer common.Address, creds *types.ApiKeyCreds, l2HeaderArgs *types.L2HeaderArgs, timestamp string) (*types.L2PolyHeader, error) {
	address := signer.Hex()
	var body *string
	if l2HeaderArgs.SerializedBody != "" {
		body = &l2HeaderArgs.SerializedBody
	} else {
		body = &l2HeaderArgs.Body
	}

	hmacSig := hmac.BuildPolyHmacSignature(creds.Secret, timestamp, l2HeaderArgs.Method, l2HeaderArgs.RequestPath, body)

	headers := &types.L2PolyHeader{
		POLYAddress:    address,
		POLYSignature:  hmacSig,
		POLYTimestamp:  timestamp,
		POLYAPIKey:     creds.Key,
		POLYPassphrase: creds.Passphrase,
	}

	return headers, nil
}

type L2WithBuilderHeader struct {
	types.L2PolyHeader
	POLYBuilderAPIKey     string `json:"POLY_BUILDER_API_KEY"`
	POLYBuilderTimestamp  string `json:"POLY_BUILDER_TIMESTAMP"`
	POLYBuilderPassphrase string `json:"POLY_BUILDER_PASSPHRASE"`
	POLYBuilderSignature  string `json:"POLY_BUILDER_SIGNATURE"`
}

type BuilderConfig struct {
	APIKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

func (bc *BuilderConfig) IsValid() bool {
	return bc != nil && bc.APIKey != "" && bc.Secret != "" && bc.Passphrase != ""
}

func (bc *BuilderConfig) GenerateBuilderHeaders(method string, path string, body *string, tsStr string) (*L2WithBuilderHeader, error) {
	if !bc.IsValid() {
		return nil, fmt.Errorf("invalid builder config")
	}

	sig := hmac.BuildPolyHmacSignature(bc.Secret, tsStr, method, path, body)

	builderHeaders := &L2WithBuilderHeader{
		POLYBuilderAPIKey:     bc.APIKey,
		POLYBuilderTimestamp:  tsStr,
		POLYBuilderPassphrase: bc.Passphrase,
		POLYBuilderSignature:  sig,
	}

	return builderHeaders, nil
}

func InsertBuilderHeaders(l2Headers *types.L2PolyHeader, builderHeaders *L2WithBuilderHeader) *L2WithBuilderHeader {
	return &L2WithBuilderHeader{
		L2PolyHeader:          *l2Headers,
		POLYBuilderAPIKey:     builderHeaders.POLYBuilderAPIKey,
		POLYBuilderTimestamp:  builderHeaders.POLYBuilderTimestamp,
		POLYBuilderPassphrase: builderHeaders.POLYBuilderPassphrase,
		POLYBuilderSignature:  builderHeaders.POLYBuilderSignature,
	}
}
