package bridge

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"
)

const defaultBridgeBaseURL = "https://bridge.polymarket.com"

// BridgeClient represents Polymarket Bridge client
type BridgeClient struct {
	host       string
	httpClient *http.Client
}

type ClientConfig struct {
	Host     string
	Timeout  time.Duration
	ProxyUrl string
}

// NewBridgeClient creates a BridgeClient with optional proxy and timeout.
func NewBridgeClient(cfg *ClientConfig) (*BridgeClient, error) {
	host := defaultBridgeBaseURL
	if cfg != nil && strings.TrimSpace(cfg.Host) != "" {
		host = strings.TrimSpace(cfg.Host)
	}
	if len(host) > 0 && host[len(host)-1] == '/' {
		host = host[:len(host)-1]
	}

	timeout := 20 * time.Second
	if cfg != nil && cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}

	c := &BridgeClient{
		host: host,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}

	if cfg != nil && strings.TrimSpace(cfg.ProxyUrl) != "" {
		proxyURL, err := url.Parse(cfg.ProxyUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy url: %w", err)
		}
		c.httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	return c, nil
}

type CreateDepositAddressResponse struct {
	Address struct {
		EVM string `json:"evm"`
		SVM string `json:"svm"`
		BTC string `json:"btc"`
	} `json:"address"`
	Note string `json:"note"`
}

type TokenInfo struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
	Decimals int    `json:"decimals"`
}

type SupportedAsset struct {
	ChainID        string    `json:"chainId"`
	ChainName      string    `json:"chainName"`
	Token          TokenInfo `json:"token"`
	MinCheckoutUsd float64   `json:"minCheckoutUsd"`
}

type SupportedAssetsResponse struct {
	SupportedAssets []SupportedAsset `json:"supportedAssets"`
}

type DepositTransaction struct {
	FromChainID        string `json:"fromChainId"`
	FromTokenAddress   string `json:"fromTokenAddress"`
	FromAmountBaseUnit string `json:"fromAmountBaseUnit"`

	ToChainID      string `json:"toChainId"`
	ToTokenAddress string `json:"toTokenAddress"`

	TxHash        string `json:"txHash,omitempty"`
	CreatedTimeMs int64  `json:"createdTimeMs,omitempty"`
	Status        string `json:"status"` // e.g. DEPOSIT_DETECTED / PROCESSING / ... / COMPLETED / FAILED
}

type DepositStatusResponse struct {
	Transactions []DepositTransaction `json:"transactions"`
}

type QuoteRequest struct {
	FromAmountBaseUnit string `json:"fromAmountBaseUnit"` // required
	FromChainID        string `json:"fromChainId"`        // required
	FromTokenAddress   string `json:"fromTokenAddress"`   // required

	RecipientAddress string `json:"recipientAddress"` // required (your Polymarket wallet / funder)
	ToChainID        string `json:"toChainId"`        // required
	ToTokenAddress   string `json:"toTokenAddress"`   // required
}

type FeeBreakdown struct {
	AppFeeLabel     string  `json:"appFeeLabel"`
	AppFeePercent   float64 `json:"appFeePercent"`
	AppFeeUsd       float64 `json:"appFeeUsd"`
	FillCostPercent float64 `json:"fillCostPercent"`
	FillCostUsd     float64 `json:"fillCostUsd"`
	GasUsd          float64 `json:"gasUsd"`
	MaxSlippage     float64 `json:"maxSlippage"`
	MinReceived     float64 `json:"minReceived"`
	SwapImpact      float64 `json:"swapImpact"`
	SwapImpactUsd   float64 `json:"swapImpactUsd"`
	TotalImpact     float64 `json:"totalImpact"`
	TotalImpactUsd  float64 `json:"totalImpactUsd"`
}

type QuoteResponse struct {
	EstCheckoutTimeMs  int64        `json:"estCheckoutTimeMs"`
	EstFeeBreakdown    FeeBreakdown `json:"estFeeBreakdown"`
	EstInputUsd        float64      `json:"estInputUsd"`
	EstOutputUsd       float64      `json:"estOutputUsd"`
	EstToTokenBaseUnit string       `json:"estToTokenBaseUnit"`
	QuoteID            string       `json:"quoteId"`
}

type ErrResp struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (c *BridgeClient) doJSON(method, endpoint string, data interface{}, expectedStatus int, result interface{}) error {
	var bodyReader io.Reader
	if data != nil {
		switch v := data.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			b, err := sonic.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal request data: %w", err)
			}
			bodyReader = bytes.NewReader(b)
		}
	}

	req, err := http.NewRequest(method, c.host+endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != expectedStatus {
		er := &ErrResp{}
		if e := sonic.Unmarshal(raw, er); e == nil {
			if strings.TrimSpace(er.Error) != "" {
				return fmt.Errorf("%s", er.Error)
			}
			if strings.TrimSpace(er.Message) != "" {
				return fmt.Errorf("%s", er.Message)
			}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}

	if result == nil {
		return errors.New("result should not be nil")
	}
	if err := sonic.Unmarshal(raw, result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w, body: %s", err, string(raw))
	}
	return nil
}

// CreateDepositAddress creates unique deposit addresses for bridging assets to Polymarket.
//
// IMPORTANT:
// - For PrivateKey signer (EOA trading wallet): pass EOA address
// - For Turnkey+Safe flow: pass SAFE address (your Polymarket wallet / funder wallet)
func (c *BridgeClient) CreateDepositAddress(address common.Address) (*CreateDepositAddressResponse, error) {
	if (address == common.Address{}) {
		return nil, fmt.Errorf("address is empty")
	}

	reqBody := map[string]string{
		"address": address.Hex(),
	}

	var out CreateDepositAddressResponse
	if err := c.doJSON(http.MethodPost, "/deposit", reqBody, http.StatusCreated, &out); err != nil {
		return nil, fmt.Errorf("bridge /deposit failed: %w", err)
	}
	return &out, nil
}

// GetSupportedAssets fetches all supported chains/tokens and minimum deposit thresholds.
func (c *BridgeClient) GetSupportedAssets() (*SupportedAssetsResponse, error) {
	var out SupportedAssetsResponse
	if err := c.doJSON(http.MethodGet, "/supported-assets", nil, http.StatusOK, &out); err != nil {
		return nil, fmt.Errorf("GET /supported-assets failed: %w", err)
	}
	return &out, nil
}

// GetDepositStatus gets deposit status for a *deposit address* (EVM/SVM/BTC) returned by POST /deposit.
func (c *BridgeClient) GetDepositStatus(depositAddress string) (*DepositStatusResponse, error) {
	depositAddress = strings.TrimSpace(depositAddress)
	if depositAddress == "" {
		return nil, fmt.Errorf("depositAddress is empty")
	}

	ep := "/status/" + url.PathEscape(depositAddress)

	var out DepositStatusResponse
	if err := c.doJSON(http.MethodGet, ep, nil, http.StatusOK, &out); err != nil {
		return nil, fmt.Errorf("GET %s failed: %w", ep, err)
	}
	return &out, nil
}

// GetAQuote returns an estimated quote for a deposit/withdrawal.
func (c *BridgeClient) GetAQuote(req QuoteRequest) (*QuoteResponse, error) {
	if req.FromAmountBaseUnit == "" ||
		req.FromChainID == "" ||
		req.FromTokenAddress == "" ||
		req.RecipientAddress == "" ||
		req.ToChainID == "" ||
		req.ToTokenAddress == "" {
		return nil, fmt.Errorf("missing required fields in QuoteRequest")
	}

	var out QuoteResponse
	if err := c.doJSON(http.MethodPost, "/quote", req, http.StatusOK, &out); err != nil {
		return nil, fmt.Errorf("POST /quote failed: %w", err)
	}
	return &out, nil
}
