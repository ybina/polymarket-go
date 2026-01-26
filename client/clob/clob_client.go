package clob

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/ybina/polymarket-go/client/clob/clob_types"
	"github.com/ybina/polymarket-go/client/clob/order_builder"
	"github.com/ybina/polymarket-go/client/clob/utils_order_builder"
	"github.com/ybina/polymarket-go/client/config"
	"github.com/ybina/polymarket-go/client/constants"
	"github.com/ybina/polymarket-go/client/endpoint"
	"github.com/ybina/polymarket-go/client/signer"
	"github.com/ybina/polymarket-go/client/types"
	"github.com/ybina/polymarket-go/tools/headers"
)

// ClobClient represents CLOB client
type ClobClient struct {
	host           string
	chainID        types.Chain
	signer         *signer.Signer
	creds          *types.ApiKeyCreds
	builderConfig  *headers.BuilderConfig
	geoBlockToken  string
	useServerTime  bool
	httpClient     *http.Client
	contractConfig config.ContractConfig
}

type ClientConfig struct {
	Host          string
	ChainID       types.Chain
	APIKey        *types.ApiKeyCreds
	BuilderConfig *headers.BuilderConfig
	GeoBlockToken string
	UseServerTime bool
	Timeout       time.Duration
	ProxyUrl      string
	Signer        *signer.Signer
}

func NewClobClient(config *ClientConfig) (*ClobClient, error) {

	host := config.Host
	if len(host) > 0 && host[len(host)-1] == '/' {
		host = host[:len(host)-1]
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &ClobClient{
		host:          host,
		chainID:       config.ChainID,
		signer:        config.Signer,
		creds:         config.APIKey,
		builderConfig: config.BuilderConfig,
		geoBlockToken: config.GeoBlockToken,
		useServerTime: config.UseServerTime,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
	if config.ProxyUrl != "" {
		proxyUrl, err := url.Parse(config.ProxyUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy url: %w", err)
		}
		client.httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		}
	}

	return client, nil
}

func (c *ClobClient) GetOK() (interface{}, error) {
	return c.get("/")
}

func (c *ClobClient) GetServerTime() (int64, error) {
	var result int64
	err := c.getJSON(endpoint.Time, &result)

	return result, err
}

func (c *ClobClient) GetSamplingSimplifiedMarkets(nextCursor string) (*types.PaginationPayload, error) {
	params := url.Values{}
	if nextCursor != "" {
		params.Add("next_cursor", nextCursor)
	}

	var result types.PaginationPayload
	err := c.getJSONWithParams(endpoint.GetSamplingSimplifiedMarkets, params, &result)
	return &result, err
}

func (c *ClobClient) GetMarkets(nextCursor string) (*types.PaginationPayload, error) {
	params := url.Values{}
	if nextCursor != "" {
		params.Add("next_cursor", nextCursor)
	}

	var result types.PaginationPayload
	err := c.getJSONWithParams(endpoint.GetMarkets, params, &result)
	return &result, err
}

func (c *ClobClient) GetMarket(conditionID string) (interface{}, error) {
	return c.get(endpoint.GetMarket + conditionID)
}

func (c *ClobClient) GetOrderBook(tokenID string) (*types.OrderBookSummary, error) {
	params := url.Values{}
	params.Add("token_id", tokenID)

	var result types.OrderBookSummary
	err := c.getJSONWithParams(endpoint.GetOrderBook, params, &result)
	return &result, err
}

func (c *ClobClient) GetOrderBooks(params []types.BookParams) ([]types.OrderBookSummary, error) {
	var result []types.OrderBookSummary
	err := c.postJSON(endpoint.GetOrderBooks, params, &result)
	return result, err
}

func (c *ClobClient) GetTickSize(tokenID string) (types.TickSize, error) {
	params := url.Values{}
	params.Add("token_id", tokenID)

	var result struct {
		MinimumTickSize decimal.Decimal `json:"minimum_tick_size"`
	}

	err := c.getJSONWithParams(endpoint.GetTickSize, params, &result)
	if err != nil {
		return "", err
	}
	return types.TickSize(result.MinimumTickSize.String()), nil
}

func (c *ClobClient) GetNegRisk(tokenID string) (bool, error) {
	params := url.Values{}
	params.Add("token_id", tokenID)

	var result struct {
		NegRisk bool `json:"neg_risk"`
	}

	err := c.getJSONWithParams(endpoint.GetNegRisk, params, &result)
	return result.NegRisk, err
}

func (c *ClobClient) GetFeeRateBps(tokenID string) (int, error) {
	params := url.Values{}
	params.Add("token_id", tokenID)

	var result struct {
		BaseFee int `json:"base_fee"`
	}

	err := c.getJSONWithParams(endpoint.GetFeeRate, params, &result)
	return result.BaseFee, err
}

func (c *ClobClient) ResolveFeeRateBps(tokenID string, userFeeRate int) (int, error) {
	marketFeeRateBps, err := c.GetFeeRateBps(tokenID)
	if err != nil {
		return 0, err
	}
	if marketFeeRateBps > 0 && userFeeRate > 0 && marketFeeRateBps != userFeeRate {
		return 0, fmt.Errorf("invalid user provided fee rate: (%v), fee rate for the market must be %v", userFeeRate, marketFeeRateBps)
	}
	return marketFeeRateBps, nil
}

func (c *ClobClient) GetMidpoint(tokenID string) (interface{}, error) {
	params := url.Values{}
	params.Add("token_id", tokenID)
	return c.getWithParams(endpoint.GetMidpoint, params)
}

func (c *ClobClient) GetMidpoints(params []types.BookParams) (interface{}, error) {
	var result interface{}
	err := c.postJSON(endpoint.GetMidpoints, params, &result)
	return result, err
}

func (c *ClobClient) GetPrice(tokenID string, side types.Side) (decimal.Decimal, error) {
	params := url.Values{}
	params.Add("token_id", tokenID)
	params.Add("side", string(side))
	res, err := c.getWithParams(endpoint.GetPrice, params)
	if err != nil {
		return decimal.Decimal{}, err
	}
	j := make(map[string]decimal.Decimal)
	err = sonic.Unmarshal(res, &j)
	if err != nil {
		return decimal.Decimal{}, err
	}
	p, ok := j["price"]
	if !ok {
		return decimal.Decimal{}, fmt.Errorf("invalid response")
	}
	return p, nil
}

func (c *ClobClient) GetPrices(params []types.BookParams) (interface{}, error) {
	var result interface{}
	err := c.postJSON(endpoint.GetPrices, params, &result)
	return result, err
}

func (c *ClobClient) GetLastTradePrice(tokenID string) (interface{}, error) {
	params := url.Values{}
	params.Add("token_id", tokenID)
	return c.getWithParams(endpoint.GetLastTradePrice, params)
}

func (c *ClobClient) GetLastTradesPrices(params []types.BookParams) (interface{}, error) {
	var result interface{}
	err := c.postJSON(endpoint.GetLastTradesPrices, params, &result)
	return result, err
}

func (c *ClobClient) GetPricesHistory(params types.PriceHistoryFilterParams) (interface{}, error) {
	queryParams := url.Values{}
	if params.Market != nil {
		queryParams.Add("market", *params.Market)
	}
	if params.StartTs != nil {
		queryParams.Add("startTs", fmt.Sprintf("%d", *params.StartTs))
	}
	if params.EndTs != nil {
		queryParams.Add("endTs", fmt.Sprintf("%d", *params.EndTs))
	}
	if params.Fidelity != nil {
		queryParams.Add("fidelity", fmt.Sprintf("%d", *params.Fidelity))
	}
	if params.Interval != nil {
		queryParams.Add("interval", string(*params.Interval))
	}

	return c.getWithParams(endpoint.GetPricesHistory, queryParams)
}

func (c *ClobClient) CreateApiKey(nonce *uint64, option clob_types.ClobOption) (*types.ApiKeyCreds, error) {
	if c.signer == nil {
		return nil, fmt.Errorf("signer is required to create API key")
	}

	var timestamp *int64
	if c.useServerTime {
		serverTime, err := c.GetServerTime()
		if err != nil {
			return nil, fmt.Errorf("failed to get server time: %w", err)
		}
		timestamp = &serverTime
	}

	l1Headers, err := headers.CreateL1Headers(c.signer, option, c.chainID, nonce, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 headers: %w", err)
	}

	var apiKeyRaw types.ApiKeyRaw
	err = c.postJSONWithHeaders(endpoint.CreateApiKey, l1Headers, nil, &apiKeyRaw)
	if err != nil {
		return nil, err
	}

	apiKey := &types.ApiKeyCreds{
		Key:        apiKeyRaw.APIKey,
		Secret:     apiKeyRaw.Secret,
		Passphrase: apiKeyRaw.Passphrase,
	}

	return apiKey, nil
}

func (c *ClobClient) DeriveApiKey(nonce *uint64, option clob_types.ClobOption) (*types.ApiKeyCreds, error) {
	if c.signer == nil {
		return nil, fmt.Errorf("signer is required to derive API key")
	}

	var timestamp *int64
	if c.useServerTime {
		serverTime, err := c.GetServerTime()
		if err != nil {
			return nil, fmt.Errorf("failed to get server time: %w", err)
		}
		timestamp = &serverTime
	}

	l1Headers, err := headers.CreateL1Headers(c.signer, option, c.chainID, nonce, timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 headers: %w", err)
	}

	var apiKeyRaw types.ApiKeyRaw
	err = c.getJSONWithHeaders(endpoint.DeriveApiKey, l1Headers, &apiKeyRaw)
	if err != nil {
		return nil, err
	}

	apiKey := &types.ApiKeyCreds{
		Key:        apiKeyRaw.APIKey,
		Secret:     apiKeyRaw.Secret,
		Passphrase: apiKeyRaw.Passphrase,
	}

	return apiKey, nil
}

// GetApiKeys gets API keys
func (c *ClobClient) GetApiKeys(addr common.Address) (*types.ApiKeysResponse, error) {
	if c.creds == nil {
		return nil, fmt.Errorf("API credentials are required")
	}

	headerArgs := &types.L2HeaderArgs{
		Method:      "GET",
		RequestPath: endpoint.GetApiKeys,
	}

	l2Headers, err := c.createL2Headers(addr, headerArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 headers: %w", err)
	}

	var result types.ApiKeysResponse
	err = c.getJSONWithHeaders(endpoint.GetApiKeys, l2Headers, &result)
	return &result, err
}

// GetClosedOnlyMode gets closed only mode status
func (c *ClobClient) GetClosedOnlyMode(funder common.Address) (*types.BanStatus, error) {
	if c.creds == nil {
		return nil, fmt.Errorf("API credentials are required")
	}

	headerArgs := &types.L2HeaderArgs{
		Method:      "GET",
		RequestPath: endpoint.ClosedOnly,
	}

	l2Headers, err := c.createL2Headers(funder, headerArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 headers: %w", err)
	}

	var result types.BanStatus
	err = c.getJSONWithHeaders(endpoint.ClosedOnly, l2Headers, &result)
	return &result, err
}

// DeleteApiKey deletes API key
func (c *ClobClient) DeleteApiKey(funder common.Address) error {
	if c.creds == nil {
		return fmt.Errorf("API credentials are required")
	}

	headerArgs := &types.L2HeaderArgs{
		Method:      "DELETE",
		RequestPath: endpoint.DeleteApiKey,
	}

	l2Headers, err := c.createL2Headers(funder, headerArgs)
	if err != nil {
		return fmt.Errorf("failed to create L2 headers: %w", err)
	}

	return c.deleteWithHeaders(endpoint.DeleteApiKey, l2Headers, nil, nil)
}

// GetOrder gets an order by ID
func (c *ClobClient) GetOrder(funder common.Address, orderID string) (*types.OpenOrder, error) {
	if c.creds == nil {
		return nil, fmt.Errorf("API credentials are required")
	}

	endpoint := endpoint.GetOrder + orderID
	headerArgs := &types.L2HeaderArgs{
		Method:      "GET",
		RequestPath: endpoint,
	}

	headers, err := c.createL2Headers(funder, headerArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 headers: %w", err)
	}

	var result types.OpenOrder
	err = c.getJSONWithHeaders(endpoint, headers, &result)
	return &result, err
}

// GetTrades gets trades
func (c *ClobClient) GetTrades(funder common.Address, params *types.TradeParams, onlyFirstPage bool, nextCursor string) ([]types.Trade, error) {
	if c.creds == nil {
		return nil, fmt.Errorf("API credentials are required")
	}

	headerArgs := &types.L2HeaderArgs{
		Method:      "GET",
		RequestPath: endpoint.GetTrades,
	}

	headers, err := c.createL2Headers(funder, headerArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 headers: %w", err)
	}

	queryParams := url.Values{}
	if nextCursor == "" {
		nextCursor = types.INITIAL_CURSOR
	}
	queryParams.Add("next_cursor", nextCursor)

	if params != nil {
		if params.ID != nil {
			queryParams.Add("id", *params.ID)
		}
		if params.MakerAddress != nil {
			queryParams.Add("maker_address", *params.MakerAddress)
		}
		if params.Market != nil {
			queryParams.Add("market", *params.Market)
		}
		if params.AssetID != nil {
			queryParams.Add("asset_id", *params.AssetID)
		}
		if params.Before != nil {
			queryParams.Add("before", *params.Before)
		}
		if params.After != nil {
			queryParams.Add("after", *params.After)
		}
	}

	var result struct {
		Data       []types.Trade `json:"data"`
		NextCursor string        `json:"next_cursor"`
	}

	err = c.getJSONWithHeadersAndParams(endpoint.GetTrades, headers, queryParams, &result)
	if err != nil {
		return nil, err
	}

	if onlyFirstPage || result.NextCursor == "-1" {
		return result.Data, nil
	}

	// Recursively get all pages
	moreTrades, err := c.GetTrades(funder, params, onlyFirstPage, result.NextCursor)
	if err != nil {
		return result.Data, nil // Return what we have so far
	}

	return append(result.Data, moreTrades...), nil
}

// Helper methods for HTTP requests

func (c *ClobClient) get(endpoint string) (interface{}, error) {
	return c.getWithParams(endpoint, url.Values{})
}

func (c *ClobClient) getWithParams(endpoint string, params url.Values) ([]byte, error) {
	fullURL := c.host + endpoint
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}
	// log.Printf("GET full %s\n", fullURL)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add geo block token if present
	if c.geoBlockToken != "" {
		q := req.URL.Query()
		q.Add("geo_block_token", c.geoBlockToken)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	resBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return resBytes, nil
}

func (c *ClobClient) getJSON(endpoint string, result interface{}) error {
	return c.getJSONWithParams(endpoint, url.Values{}, result)
}

func (c *ClobClient) getJSONWithParams(endpoint string, params url.Values, result interface{}) error {
	data, err := c.getWithParams(endpoint, params)
	if err != nil {
		return err
	}
	//log.Printf("data: %s", string(data))
	return sonic.Unmarshal(data, result)
}

func (c *ClobClient) getJSONWithHeaders(endpoint string, headers interface{}, result interface{}) error {
	return c.getJSONWithHeadersAndParams(endpoint, headers, url.Values{}, result)
}

func (c *ClobClient) getJSONWithHeadersAndParams(endpoint string, headers interface{}, params url.Values, result interface{}) error {
	fullURL := c.host + endpoint
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	c.addHeadersToRequest(req, headers)

	// Add geo block token if present
	if c.geoBlockToken != "" {
		q := req.URL.Query()
		q.Add("geo_block_token", c.geoBlockToken)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	//log.Printf("getJSONWithHeadersAndParams body: %s\n", string(body))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return sonic.Unmarshal(body, result)
}

func (c *ClobClient) postJSON(endpoint string, data interface{}, result interface{}) error {
	return c.postJSONWithHeaders(endpoint, nil, data, result)
}

type ErrResp struct {
	Error string `json:"error"`
}

func (c *ClobClient) postJSONWithHeaders(endpoint string, headers interface{}, data interface{}, result interface{}) error {
	var bodyReader io.Reader
	if data != nil {
		switch v := data.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		case json.RawMessage:
			bodyReader = bytes.NewReader(v)
		default:
			jsonData, err := sonic.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal request data: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	req, err := http.NewRequest("POST", c.host+endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add headers
	c.addHeadersToRequest(req, headers)

	// Add geo block token if present
	if c.geoBlockToken != "" {
		q := req.URL.Query()
		q.Add("geo_block_token", c.geoBlockToken)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode >= 400 {
		respErr := &ErrResp{}
		err = sonic.Unmarshal(body, respErr)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response of code 400: %w, body:%s\n", err, string(body))
		}
		return fmt.Errorf("%s", respErr.Error)
	}

	if result != nil {
		err = sonic.Unmarshal(body, result)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response: %w, body: %s", err, string(body))
		}
		return nil
	}
	return errors.New("result should not be nil")
}

func (c *ClobClient) deleteWithHeaders(endpoint string, headers interface{}, data interface{}, result interface{}) error {
	var bodyReader io.Reader
	if data != nil {
		switch v := data.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		case json.RawMessage:
			bodyReader = bytes.NewReader(v)
		default:
			j, err := sonic.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal delete body: %w", err)
			}
			bodyReader = bytes.NewReader(j)
		}
	}

	req, err := http.NewRequest("DELETE", c.host+endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.addHeadersToRequest(req, headers)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := sonic.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return nil
}

func (c *ClobClient) createL2Headers(addr common.Address, args *types.L2HeaderArgs) (interface{}, error) {
	if c.signer == nil {
		return nil, fmt.Errorf("signer is required for authenticated requests")
	}

	var timestamp *int64
	var tsStr string
	if c.useServerTime {
		serverTime, err := c.GetServerTime()
		if err != nil {
			return nil, fmt.Errorf("failed to get server time: %w", err)
		}
		timestamp = &serverTime
		tsStr = strconv.FormatInt(*timestamp, 10)
	} else {
		tsStr = strconv.FormatInt(time.Now().Unix(), 10)
	}

	return headers.CreateL2Headers(addr, c.creds, args, tsStr)
}

func (c *ClobClient) addHeadersToRequest(req *http.Request, requestHeaders interface{}) {
	switch h := requestHeaders.(type) {
	case *types.L1PolyHeader:
		req.Header.Set("POLY_ADDRESS", h.POLYAddress)
		req.Header.Set("POLY_SIGNATURE", h.POLYSignature)
		req.Header.Set("POLY_TIMESTAMP", h.POLYTimestamp)
		req.Header.Set("POLY_NONCE", h.POLYNonce)
	case *types.L2PolyHeader:
		req.Header.Set("POLY_ADDRESS", h.POLYAddress)
		req.Header.Set("POLY_SIGNATURE", h.POLYSignature)
		req.Header.Set("POLY_TIMESTAMP", h.POLYTimestamp)
		req.Header.Set("POLY_API_KEY", h.POLYAPIKey)
		req.Header.Set("POLY_PASSPHRASE", h.POLYPassphrase)
	case *headers.L2WithBuilderHeader:
		req.Header.Set("POLY_ADDRESS", h.POLYAddress)
		req.Header.Set("POLY_SIGNATURE", h.POLYSignature)
		req.Header.Set("POLY_TIMESTAMP", h.POLYTimestamp)
		req.Header.Set("POLY_API_KEY", h.POLYAPIKey)
		req.Header.Set("POLY_PASSPHRASE", h.POLYPassphrase)
		req.Header.Set("POLY_BUILDER_API_KEY", h.POLYBuilderAPIKey)
		req.Header.Set("POLY_BUILDER_TIMESTAMP", h.POLYBuilderTimestamp)
		req.Header.Set("POLY_BUILDER_PASSPHRASE", h.POLYBuilderPassphrase)
		req.Header.Set("POLY_BUILDER_SIGNATURE", h.POLYBuilderSignature)
	}
}

func (c *ClobClient) GetClientMode() constants.AccessLevel {
	if c.signer != nil && c.creds != nil {
		return constants.L2
	}
	if c.signer != nil {
		return constants.L1
	}
	return constants.L0
}

func (c *ClobClient) AssertL1Auth() error {
	if c.GetClientMode() < constants.L1 {
		return errors.New(constants.L1_AUTH_UNAVAILABLE)
	}
	return nil
}

func (c *ClobClient) AssertL2Auth() error {
	if c.GetClientMode() < constants.L2 {
		return errors.New(constants.L2_AUTH_UNAVAILABLE)
	}
	return nil
}

func (c *ClobClient) canBuilderAuth() bool {
	if c.builderConfig != nil {
		return c.builderConfig.IsValid()
	}
	return false
}

func (c *ClobClient) CreateAndPostOrder(args clob_types.OrderArgs, option clob_types.PartialCreateOrderOptions) (*types.OrderResponse, error) {
	if c.signer.SignerType() == signer.Turnkey {
		if option.TurnkeyAccount == constants.ZERO_ADDRESS {
			return nil, fmt.Errorf("turnkeyAccount is required")
		}
		if option.SafeAccount == constants.ZERO_ADDRESS {
			return nil, fmt.Errorf("safe account is required")
		}
	}
	signedOrder, err := c.createOrder(args, option)
	if err != nil {
		return nil, err
	}
	return c.postOrder(signedOrder, option)
}

func (c *ClobClient) createOrder(args clob_types.OrderArgs, option clob_types.PartialCreateOrderOptions) (utils_order_builder.SignedOrder, error) {
	err := c.AssertL1Auth()
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	if option.TickSize == nil {
		tickSize, err := c.GetTickSize(args.TokenID)
		if err != nil {
			return utils_order_builder.SignedOrder{}, err
		}
		option.TickSize = &tickSize
	}

	err = c.priceValid(args.Price, *option.TickSize)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	if option.NegRisk == nil {
		isNegRisk, err := c.GetNegRisk(args.TokenID)
		if err != nil {
			return utils_order_builder.SignedOrder{}, err
		}
		option.NegRisk = &isNegRisk
	}

	feeRateBps, err := c.ResolveFeeRateBps(args.TokenID, args.FeeRateBps)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	args.FeeRateBps = feeRateBps
	var funder common.Address
	var signatureType constants.SigType
	if c.signer.SignerType() == signer.Turnkey {
		signatureType = constants.POLY_GNOSIS_SAFE
		funder = option.SafeAccount

	} else if c.signer.SignerType() == signer.PrivateKey {
		signatureType = constants.EOA
		funder = common.HexToAddress(c.signer.Address())
	}

	orderBuilder, err := order_builder.NewOrderBuilder(c.signer, signatureType, funder)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	return orderBuilder.CreateOrder(c.signer, args, option)

}

type RequestArgs struct {
	Method      string     `json:"method"`
	RequestPath string     `json:"requestPath"`
	Body        *FinalBody `json:"body"`
}

// postOrder:
func (c *ClobClient) postOrder(order utils_order_builder.SignedOrder, option clob_types.PartialCreateOrderOptions) (*types.OrderResponse, error) {
	err := c.AssertL2Auth()
	if err != nil {
		return nil, err
	}
	if c.creds == nil {
		return nil, fmt.Errorf("API credentials required")
	}
	if option.OrderType == "" {
		option.OrderType = types.OrderTypeGTC
	}
	body, err := c.orderToBody(order, c.creds, string(option.OrderType))
	if err != nil {
		return nil, err
	}
	bodyStr, err := sonic.MarshalString(body)
	if err != nil {
		return nil, err
	}

	serializedBody, err := serializeJsonBody(body)
	if err != nil {
		return nil, err
	}

	requestArgs := &types.L2HeaderArgs{
		Method:         "POST",
		RequestPath:    endpoint.PostOrder,
		Body:           bodyStr,
		SerializedBody: serializedBody,
	}
	timestamp, err := c.GetServerTime()
	if err != nil {
		return nil, err
	}
	tsStr := strconv.FormatInt(timestamp, 10)

	l2headers := &types.L2PolyHeader{}
	if c.signer.SignerType() == signer.Turnkey {

		l2headers, err = headers.CreateL2Headers(option.TurnkeyAccount, c.creds, requestArgs, tsStr)
		if err != nil {
			return nil, err
		}
	} else {
		pub, err := c.signer.GetPubkeyOfPrivateKey()
		if err != nil {
			return nil, err
		}
		l2headers, err = headers.CreateL2Headers(pub, c.creds, requestArgs, strconv.FormatInt(timestamp, 10))
		if err != nil {
			return nil, err
		}
	}

	canBuilderAuth := c.canBuilderAuth()
	if canBuilderAuth {
		builderHeaders, err := c.builderConfig.GenerateBuilderHeaders(requestArgs.Method, requestArgs.RequestPath, &requestArgs.SerializedBody, tsStr)
		if err != nil {
			return nil, err
		}
		if builderHeaders == nil {
			return nil, fmt.Errorf("builder headers is nil")
		}
		enriched := headers.InsertBuilderHeaders(l2headers, builderHeaders)
		result := types.OrderResponse{}
		err = c.postJSONWithHeaders(endpoint.PostOrder, enriched, serializedBody, &result)
		if err != nil {
			log.Printf("postJSONWithHeaders: %s\n", err)
			return nil, err
		}
		return &result, nil
	} else {
		result := types.OrderResponse{}
		l2h, err := sonic.MarshalString(l2headers)
		if err != nil {
			return nil, err
		}
		log.Printf("before post l2headers:%v\n", l2h)
		err = c.postJSONWithHeaders(endpoint.PostOrder, l2headers, serializedBody, &result)
		if err != nil {
			return nil, err
		}
		return &result, nil
	}

}

func serializeJsonBody(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "") // no indent
	if err := enc.Encode(v); err != nil {
		return "", err
	}

	out := strings.TrimSuffix(buf.String(), "\n")
	return out, nil
}

type FinalBody struct {
	Order     types.SignedOrder `json:"order"`
	Owner     string            `json:"owner"`
	OrderType string            `json:"orderType"`
}

func (c *ClobClient) orderToBody(order utils_order_builder.SignedOrder, creds *types.ApiKeyCreds, orderType string) (*FinalBody, error) {
	if creds.Key == "" {
		return nil, fmt.Errorf("API credentials required")
	}
	sigType := int(order.SignatureType)

	fOrder := types.SignedOrder{
		Salt:          order.Salt,
		Maker:         order.Maker.Hex(),
		Signer:        order.Signer.Hex(),
		Taker:         order.Taker.Hex(),
		TokenID:       order.TokenID,
		MakerAmount:   order.MakerAmount,
		TakerAmount:   order.TakerAmount,
		Expiration:    order.Expiration,
		Nonce:         order.Nonce,
		FeeRateBps:    order.FeeRateBps,
		Side:          order.Side,
		SignatureType: types.SignatureType(sigType),
		Signature:     order.Signature,
	}
	body := &FinalBody{
		Order:     fOrder,
		Owner:     creds.Key,
		OrderType: orderType,
	}
	return body, nil
}

func (c *ClobClient) generateBuilderHeaders(headerArgs types.L2HeaderArgs, headers types.L2PolyHeader) {

}
func (c *ClobClient) priceValid(price decimal.Decimal, tickSize types.TickSize) error {
	ts, err := decimal.NewFromString(string(tickSize))
	if err != nil {
		return err
	}

	minPara := ts
	maxPara := decimal.NewFromFloat(1).Sub(ts)

	if price.LessThan(minPara) || price.GreaterThan(maxPara) {
		return fmt.Errorf(
			"price (%v), min: %v - max: %v",
			price,
			minPara,
			maxPara,
		)
	}

	return nil
}

func (c *ClobClient) CreateOrDeriveApiKey() {

}

func (c *ClobClient) GetOrders(orderIds []string) {

}

func (c *ClobClient) CancelOrder(orderId string, signerAddr common.Address) (*types.OrderResponse, error) {
	err := c.AssertL2Auth()
	if err != nil {
		return nil, err
	}
	body := make(map[string]string)
	body["orderId"] = orderId
	bodyJs, err := sonic.MarshalString(body)
	if err != nil {
		return nil, err
	}
	args := &types.L2HeaderArgs{
		Method:         "DELETE",
		RequestPath:    endpoint.CancelOrder,
		Body:           bodyJs,
		SerializedBody: bodyJs,
	}

	timestamp, err := c.GetServerTime()
	if err != nil {
		return nil, err
	}
	tsStr := strconv.FormatInt(timestamp, 10)
	l2Headers, err := headers.CreateL2Headers(signerAddr, c.creds, args, tsStr)
	if err != nil {
		return nil, err
	}
	var result types.OrderResponse
	err = c.deleteWithHeaders(endpoint.CancelOrder, l2Headers, bodyJs, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *ClobClient) CancelOrders() {

}

func (c *ClobClient) CancelAllOrders() {

}

func (c *ClobClient) CreateAndPostMarketOrder(args clob_types.MarketOrderArgs, option clob_types.PartialCreateOrderOptions) (*types.OrderResponse, error) {
	argStr, err := sonic.MarshalString(args)
	if err != nil {
		return nil, err
	}
	log.Printf("CreateAndPostMarketOrder arg:%v\n", argStr)
	signedOrder, err := c.createMarketOrder(args, option)
	if err != nil {
		return nil, err
	}

	return c.postOrder(signedOrder, option)
}

func (c *ClobClient) createMarketOrder(args clob_types.MarketOrderArgs, option clob_types.PartialCreateOrderOptions) (utils_order_builder.SignedOrder, error) {
	err := c.AssertL1Auth()
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	if option.TickSize == nil {
		tickSize, err := c.GetTickSize(args.TokenID)
		if err != nil {
			return utils_order_builder.SignedOrder{}, err
		}
		option.TickSize = &tickSize
	}
	err = c.priceValid(args.Price, *option.TickSize)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	if option.NegRisk == nil {
		isNegRisk, err := c.GetNegRisk(args.TokenID)
		if err != nil {
			return utils_order_builder.SignedOrder{}, err
		}
		option.NegRisk = &isNegRisk
	}

	feeRateBps, err := c.ResolveFeeRateBps(args.TokenID, args.FeeRateBps)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	args.FeeRateBps = feeRateBps
	if args.OrderType == "" {
		args.OrderType = types.OrderTypeFOK
	}
	var funder common.Address
	var signatureType constants.SigType
	if c.signer.SignerType() == signer.Turnkey {
		signatureType = constants.POLY_GNOSIS_SAFE
		funder = option.SafeAccount

	} else if c.signer.SignerType() == signer.PrivateKey {
		signatureType = constants.EOA
		funder = common.HexToAddress(c.signer.Address())
	}

	orderBuilder, err := order_builder.NewOrderBuilder(c.signer, signatureType, funder)
	if err != nil {
		return utils_order_builder.SignedOrder{}, err
	}
	return orderBuilder.CreateMarketOrder(c.signer, args, option)
}

func (c *ClobClient) CancelMarketOrders() {}

func (c *ClobClient) GetBuilderTrades() {
}
