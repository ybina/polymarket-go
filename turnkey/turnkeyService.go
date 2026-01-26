package turnkey

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/tkhq/go-sdk"
	"github.com/tkhq/go-sdk/pkg/api/client/signing"
	"github.com/tkhq/go-sdk/pkg/api/client/wallets"
	"github.com/tkhq/go-sdk/pkg/api/models"
	"github.com/tkhq/go-sdk/pkg/apikey"
	"github.com/ybina/polymarket-go/tools/utils"
)

type Config struct {
	PubKey       string `json:"publicKey"`
	PrivateKey   string `json:"privateKey"`
	Organization string `json:"organization"`
	WalletName   string `json:"masterWalletName"`
}

type WalletInfo struct {
	WalletID  string
	Addresses []string
}

type Client struct {
	client     *sdk.Client
	WalletName string
	WalletId   string
}

func NewTurnKeyService(config Config) (turnkeyClient Client, err error) {

	scheme := apikey.SchemeP256
	apiKey, err := apikey.FromTurnkeyPrivateKey(config.PrivateKey, scheme)
	if err != nil {
		panic(err)
	}
	apiKey.Metadata.Organizations = []string{config.Organization}
	var turnKeyClient Client
	client, err := sdk.New(sdk.WithAPIKey(apiKey))

	if err != nil {
		return turnKeyClient, err
	}
	if config.WalletName == "" {
		return turnKeyClient, fmt.Errorf("wallet name is required")
	}
	turnKeyClient.WalletName = config.WalletName
	turnKeyClient.client = client
	walletId, err := turnKeyClient.TryCreateNewWallet(config.WalletName)
	if err != nil {
		return turnKeyClient, err
	}
	turnKeyClient.WalletId = walletId

	return turnKeyClient, nil
}

func (c *Client) TryCreateNewWallet(walletName string) (walletId string, err error) {

	list, err := c.GetWalletList()
	if err != nil {
		return "", fmt.Errorf("failed to get existing master wallet list: %w", err)
	}

	for _, w := range list {
		if w.WalletName != nil && *w.WalletName == walletName {
			if w.WalletID != nil {
				return *w.WalletID, nil
			}
			return "", fmt.Errorf("got nil walletID for %s", walletName)
		}
	}
	walletId, _, err = c.CreateNewWallet(walletName)
	return
}

func (c *Client) GetWalletList() ([]*models.Wallet, error) {
	if c.client == nil {
		return nil, errors.New("client is nil")
	}
	params := wallets.NewGetWalletsParams().WithBody(&models.GetWalletsRequest{
		OrganizationID: c.client.DefaultOrganization(),
	})

	resp, err := c.client.V0().Wallets.GetWallets(params, c.client.Authenticator)
	if err != nil {
		return nil, err
	}

	if resp == nil ||
		resp.Payload == nil ||
		resp.Payload.Wallets == nil || len(resp.Payload.Wallets) <= 0 {
		return nil, errors.New("unexpected empty response from GetWallet")
	}

	return resp.Payload.Wallets, nil
}

func (c *Client) CreateNewWallet(walletName string) (walletId string, depositAddr string, err error) {
	if walletName == "" {
		return "", "", errors.New("walletName is empty")
	}
	path := "m/44'/60'/0'/0/0"
	timestamp := time.Now().UnixMilli()
	timestampString := strconv.FormatInt(timestamp, 10)
	params := wallets.NewCreateWalletParams().WithBody(&models.CreateWalletRequest{
		OrganizationID: c.client.DefaultOrganization(),
		Parameters: &models.CreateWalletIntent{
			Accounts: []*models.WalletAccountParams{
				{
					AddressFormat: models.AddressFormatEthereum.Pointer(),
					Curve:         models.CurveSecp256k1.Pointer(),
					Path:          &path,
					PathFormat:    models.PathFormatBip32.Pointer(),
				},
			},
			WalletName: &walletName,
		},
		TimestampMs: &timestampString,
		Type:        (*string)(models.ActivityTypeCreateWallet.Pointer()),
	})

	resp, err := c.client.V0().Wallets.CreateWallet(params, c.client.Authenticator)
	if err != nil {
		return "", "", err
	}
	if resp.Payload.Activity.Result.CreateWalletResult.WalletID == nil || *resp.Payload.Activity.Result.CreateWalletResult.WalletID == "" {
		return "", "", fmt.Errorf("got nil walletID for %s", walletName)
	}
	if len(resp.Payload.Activity.Result.CreateWalletResult.Addresses) <= 0 || resp.Payload.Activity.Result.CreateWalletResult.Addresses[0] == "" {
		return "", "", errors.New("unexpected empty addresses")
	}
	return *resp.Payload.Activity.Result.CreateWalletResult.WalletID, resp.Payload.Activity.Result.CreateWalletResult.Addresses[0], nil
}

func (c *Client) CreateAccount(idx uint64) (string, error) {
	if c.WalletId == "" {
		return "", errors.New("walletId is empty")
	}
	timestamp := time.Now().UnixMilli()
	timestampString := strconv.FormatInt(timestamp, 10)
	path := fmt.Sprintf("m/44'/60'/0'/0/%d", idx)
	params := wallets.NewCreateWalletAccountsParams().WithBody(&models.CreateWalletAccountsRequest{
		OrganizationID: c.client.DefaultOrganization(),
		TimestampMs:    &timestampString,
		Parameters: &models.CreateWalletAccountsIntent{
			WalletID: &c.WalletId,
			Accounts: []*models.WalletAccountParams{
				&models.WalletAccountParams{
					AddressFormat: models.AddressFormatEthereum.Pointer(),
					Curve:         models.CurveSecp256k1.Pointer(),
					PathFormat:    models.PathFormatBip32.Pointer(),
					Path:          &path,
				},
			},
		},
		Type: (*string)(models.ActivityTypeCreateWalletAccounts.Pointer()),
	})
	resp, err := c.client.V0().Wallets.CreateWalletAccounts(params, c.client.Authenticator)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New("unexpected resp nil")
	}
	if resp.Payload == nil {
		return "", errors.New("unexpected nil response payload")
	}
	if resp.Payload.Activity == nil {
		return "", errors.New("unexpected nil response activity")
	}
	if resp.Payload.Activity.Result == nil {
		return "", errors.New("unexpected nil response activity Result")
	}

	if resp.Payload.Activity.Result.CreateWalletAccountsResult == nil {
		return "", errors.New("unexpected nil response CreateWalletAccountsResult")
	}
	if resp.Payload.Activity.Result.CreateWalletAccountsResult.Addresses == nil {
		return "", errors.New("unexpected nil response CreateWalletAccountsResult.Addresses")
	}
	if len(resp.Payload.Activity.Result.CreateWalletAccountsResult.Addresses) <= 0 ||
		resp.Payload.Activity.Result.CreateWalletAccountsResult.Addresses[0] == "" {
		return "", fmt.Errorf("unexpected empty addresses")
	}
	return resp.Payload.Activity.Result.CreateWalletAccountsResult.Addresses[0], nil
}

func (c *Client) Sign(userAccount string, unsignedMsg string) (signedMsg string, err error) {
	if c.client == nil {
		return "", errors.New("client is nil")
	}
	if userAccount == "" {
		return "", errors.New("userAccount is empty")
	}
	decoded, err := base64.StdEncoding.DecodeString(unsignedMsg)
	if err != nil {
		return "", err
	}
	payloadHexMsg := hex.EncodeToString(decoded)
	timestamp := time.Now().UnixMilli()
	timestampString := strconv.FormatInt(timestamp, 10)
	pkParams := signing.NewSignRawPayloadParams().WithBody(&models.SignRawPayloadRequest{
		OrganizationID: c.client.DefaultOrganization(),
		TimestampMs:    &timestampString,
		Parameters: &models.SignRawPayloadIntentV2{
			SignWith:     &userAccount,
			Payload:      &payloadHexMsg,
			HashFunction: models.HashFunctionNoOp.Pointer(),
			Encoding:     models.PayloadEncodingHexadecimal.Pointer(),
		},
		Type: (*string)(models.ActivityTypeSignRawPayloadV2.Pointer()),
	})
	signedResp, err := c.client.V0().Signing.SignRawPayload(pkParams, c.client.Authenticator)

	if err != nil {
		return "", err
	}
	if signedResp.Payload == nil ||
		signedResp.Payload.Activity == nil ||
		signedResp.Payload.Activity.Result == nil ||
		signedResp.Payload.Activity.Result.SignRawPayloadResult == nil ||
		signedResp.Payload.Activity.Result.SignRawPayloadResult.R == nil ||
		signedResp.Payload.Activity.Result.SignRawPayloadResult.S == nil ||
		signedResp.Payload.Activity.Result.SignRawPayloadResult.V == nil {
		return "", errors.New("unexpected nil fields in signed response")
	}
	result := signedResp.Payload.Activity.Result.SignRawPayloadResult
	r := utils.TrimHex(*result.R)
	s := utils.TrimHex(*result.S)
	vHex := utils.TrimHex(*result.V)
	vBytes, err := hex.DecodeString(vHex)
	if err != nil {
		return "", err
	}
	if len(vBytes) != 1 {
		return "", errors.New("invalid v length")
	}
	if vBytes[0] < 27 {
		vBytes[0] += 27
	}
	signedMsg = "0x" + r + s + hex.EncodeToString(vBytes)

	return signedMsg, nil
}

// GetAccounts if idx < 0, get all accounts of wallet
func (c *Client) GetAccounts(idx int64) ([]*models.WalletAccount, error) {
	params := wallets.NewGetWalletAccountsParams().WithBody(&models.GetWalletAccountsRequest{
		OrganizationID: c.client.DefaultOrganization(),
		WalletID:       &c.WalletId,
		PaginationOptions: &models.Pagination{
			Limit: "100",
		},
	})

	resp, err := c.client.V0().Wallets.GetWalletAccounts(params, c.client.Authenticator)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Payload == nil || len(resp.Payload.Accounts) == 0 {
		return nil, errors.New("unexpected empty response from GetWalletAccounts")
	}
	var ethAccounts []*models.WalletAccount

	for _, acc := range resp.Payload.Accounts {
		if acc == nil || acc.AddressFormat == nil || acc.Curve == nil || acc.Path == nil {
			continue
		}

		if *acc.AddressFormat != "ADDRESS_FORMAT_ETHEREUM" || *acc.Curve != "CURVE_SECP256K1" {
			continue
		}

		if idx < 0 {
			ethAccounts = append(ethAccounts, acc)
			continue
		}

		wantPath := fmt.Sprintf("m/44'/60'/0'/0/%d", idx)
		if *acc.Path == wantPath {
			return []*models.WalletAccount{acc}, nil
		}
	}

	if idx < 0 {
		if len(ethAccounts) == 0 {
			return nil, errors.New("no ethereum accounts found")
		}
		return ethAccounts, nil
	}

	return nil, fmt.Errorf("ethereum account index=%d not found", idx)
}
