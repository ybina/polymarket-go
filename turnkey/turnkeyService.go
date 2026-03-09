package turnkey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tkhq/go-sdk"
)

type Config struct {
	PubKey       string `json:"publicKey"`
	PrivateKey   string `json:"privateKey"`
	Organization string `json:"organization"`
	WalletName   string `json:"masterWalletName"`
	FcUrl        string `json:"fcUrl"`
	BearerToken  string `json:"bearerToken"`
}

type WalletInfo struct {
	WalletID  string
	Addresses []string
}

type Client struct {
	client      *sdk.Client
	WalletName  string
	WalletId    string
	baseURL     string
	bearerToken string
	httpClient  *http.Client
}

func NewTurnKeyService(config Config) (turnkeyClient Client, err error) {

	return Client{
		baseURL:     config.FcUrl,
		bearerToken: config.BearerToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

type remoteResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type AccountInfo struct {
	Address string `json:"address"`
	Path    string `json:"path"`
}

func (c *Client) post(path string, payload interface{}) (*remoteResponse, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result remoteResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response (status=%d, body=%s): %w", resp.StatusCode, string(respBytes), err)
	}

	if !result.Success {
		return nil, fmt.Errorf("remote error: %s", result.Error)
	}

	return &result, nil
}

// Sign 对原始字节进行签名，返回十六进制签名字符串（0x r+s+v 格式）
func (c *Client) Sign(account string, b64Payload string) (string, error) {
	// 云端 from.go 中 Sign() 接受 base64 编码的 payload

	resp, err := c.post("/sign", map[string]string{
		"account": account,
		"payload": b64Payload,
	})
	if err != nil {
		return "", err
	}

	var result map[string]string
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return "", fmt.Errorf("parse sign result: %w", err)
	}
	return result["signature"], nil
}

// GetAccounts 获取钱包账户列表。idx=-1 获取全部；idx>=0 获取指定派生路径索引的账户。
func (c *Client) GetAccounts(idx int64) ([]AccountInfo, error) {
	resp, err := c.post("/accounts", map[string]int64{"idx": idx})
	if err != nil {
		return nil, err
	}

	var accounts []AccountInfo
	if err := json.Unmarshal(resp.Data, &accounts); err != nil {
		return nil, fmt.Errorf("parse accounts result: %w", err)
	}
	return accounts, nil
}

// CreateAccount 在指定派生索引创建新账户，返回以太坊地址
func (c *Client) CreateAccount(idx uint64) (string, error) {
	resp, err := c.post("/create-account", map[string]uint64{"idx": idx})
	if err != nil {
		return "", err
	}

	var result map[string]string
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return "", fmt.Errorf("parse create account result: %w", err)
	}
	return result["address"], nil
}
