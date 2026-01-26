package data

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	DataAPIBase = "https://data-api.polymarket.com"
)

type DataSDK struct {
	baseURL    string
	proxyUrl   *string
	httpClient *http.Client
}

func NewDataSDK(proxyUrl *string) (*DataSDK, error) {

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	client := &DataSDK{
		baseURL:    DataAPIBase,
		proxyUrl:   proxyUrl,
		httpClient: httpClient,
	}
	if client.proxyUrl != nil && *client.proxyUrl != "" {
		proxy, err := url.Parse(*proxyUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy url: %w", err)
		}
		client.httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	}

	return client, nil
}

func (d *DataSDK) GetHttpClient() *http.Client {
	return d.httpClient
}

func (d *DataSDK) buildURL(endpoint string, query interface{}) (string, error) {
	u, err := url.Parse(d.baseURL + endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	if query != nil {
		values := url.Values{}
		v := reflect.ValueOf(query)

		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return u.String(), nil
			}
			v = v.Elem()
		}

		t := v.Type()

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldValue := v.Field(i)

			if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
				continue
			}

			jsonTag := field.Tag.Get("json")
			if jsonTag == "" {
				continue
			}

			if strings.Contains(jsonTag, "omitempty") && fieldValue.IsZero() {
				continue
			}

			fieldName := strings.Split(jsonTag, ",")[0]
			if fieldName == "" {
				continue
			}

			var strValue string
			if fieldValue.Kind() == reflect.Ptr {
				strValue = fmt.Sprintf("%v", fieldValue.Elem().Interface())
			} else {
				strValue = fmt.Sprintf("%v", fieldValue.Interface())
			}

			if fieldValue.Kind() == reflect.Slice {
				slice := fieldValue.Interface()
				if sliceValue, ok := slice.([]string); ok {
					for _, item := range sliceValue {
						values.Add(fieldName, item)
					}
				}
			} else {
				values.Add(fieldName, strValue)
			}
		}

		u.RawQuery = values.Encode()
	}

	return u.String(), nil
}

func (d *DataSDK) createRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "data-go-sdk/1.0")

	return req, nil
}

func (d *DataSDK) makeRequest(method, endpoint string, query interface{}) (*APIResponse, error) {

	fullURL, err := d.buildURL(endpoint, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	req, err := d.createRequest(method, fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	apiResp := &APIResponse{
		Status: resp.StatusCode,
		OK:     resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	if resp.StatusCode == 204 {
		return apiResp, nil
	}

	if len(body) > 0 {
		if resp.StatusCode >= 400 {

			var errData map[string]interface{}
			if err := sonic.Unmarshal(body, &errData); err == nil {
				apiResp.ErrorData = errData
			} else {
				apiResp.ErrorData = string(body)
			}
		} else {
			apiResp.Data = json.RawMessage(body)
		}
	}

	return apiResp, nil
}

func (d *DataSDK) extractResponseData(resp *APIResponse, operation string) ([]byte, error) {
	if !resp.OK {
		return nil, fmt.Errorf("[DataSDK] %s failed: status %d", operation, resp.Status)
	}

	if resp.Data == nil {
		return nil, fmt.Errorf("[DataSDK] %s returned null data despite successful response", operation)
	}

	return resp.Data, nil
}

func (d *DataSDK) GetHealth() (*DataHealthResponse, error) {
	resp, err := d.makeRequest("GET", "/", nil)
	if err != nil {
		return nil, err
	}

	var result DataHealthResponse
	if resp.Data != nil {
		if err := json.Unmarshal(resp.Data, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal health response: %w", err)
		}
	}

	return &result, nil
}

func (d *DataSDK) GetCurrentPositions(query *PositionsQuery) ([]Position, error) {
	if query == nil {
		query = &PositionsQuery{}
	}

	resp, err := d.makeRequest("GET", "/positions", query)
	if err != nil {
		return nil, err
	}

	return d.unmarshalPositionsResponse(resp, "Get current positions")
}

func (d *DataSDK) GetClosedPositions(query *ClosedPositionsQuery) ([]ClosedPosition, error) {
	if query == nil {
		query = &ClosedPositionsQuery{}
	}

	resp, err := d.makeRequest("GET", "/closed-positions", query)
	if err != nil {
		return nil, err
	}

	return d.unmarshalClosedPositionsResponse(resp, "Get closed positions")
}

func (d *DataSDK) GetTrades(query *TradesQuery) ([]DataTrade, error) {
	if query == nil {
		query = &TradesQuery{}
	}

	resp, err := d.makeRequest("GET", "/trades", query)
	if err != nil {
		return nil, err
	}

	return d.unmarshalTradesResponse(resp, "Get trades")
}

func (d *DataSDK) GetUserActivity(query *UserActivityQuery) ([]Activity, error) {
	if query == nil {
		query = &UserActivityQuery{}
	}

	resp, err := d.makeRequest("GET", "/activity", query)
	if err != nil {
		return nil, err
	}

	return d.unmarshalActivityResponse(resp, "Get user activity")
}

func (d *DataSDK) GetTopHolders(query *TopHoldersQuery) ([]MetaHolder, error) {
	if query == nil {
		query = &TopHoldersQuery{}
	}

	resp, err := d.makeRequest("GET", "/holders", query)
	if err != nil {
		return nil, err
	}

	return d.unmarshalMetaHoldersResponse(resp, "Get top holders")
}

func (d *DataSDK) GetTotalValue(query *TotalValueQuery) ([]TotalValue, error) {
	if query == nil {
		query = &TotalValueQuery{}
	}

	resp, err := d.makeRequest("GET", "/value", query)
	if err != nil {
		return nil, err
	}

	return d.unmarshalTotalValueResponse(resp, "Get total value")
}

func (d *DataSDK) GetTotalMarketsTraded(query *TotalMarketsTradedQuery) (*TotalMarketsTraded, error) {
	if query == nil {
		query = &TotalMarketsTradedQuery{}
	}

	resp, err := d.makeRequest("GET", "/traded", query)
	if err != nil {
		return nil, err
	}

	return d.unmarshalTotalMarketsTradedResponse(resp, "Get total markets traded")
}

func (d *DataSDK) GetOpenInterest(query *OpenInterestQuery) ([]OpenInterest, error) {
	if query == nil {
		query = &OpenInterestQuery{}
	}

	resp, err := d.makeRequest("GET", "/oi", query)
	if err != nil {
		return nil, err
	}

	return d.unmarshalOpenInterestResponse(resp, "Get open interest")
}

func (d *DataSDK) GetLiveVolume(query *LiveVolumeQuery) (*LiveVolumeResponse, error) {
	if query == nil {
		query = &LiveVolumeQuery{}
	}

	resp, err := d.makeRequest("GET", "/live-volume", query)
	if err != nil {
		return nil, err
	}

	return d.unmarshalLiveVolumeResponse(resp, "Get live volume")
}

func (d *DataSDK) GetAllPositions(user string, options *struct {
	Limit         *int
	Offset        *int
	SortBy        *string
	SortDirection *string
}) (*struct {
	Current []Position
	Closed  []ClosedPosition
}, error) {

	currentQuery := &PositionsQuery{
		User:          &user,
		Limit:         options.Limit,
		Offset:        options.Offset,
		SortBy:        options.SortBy,
		SortDirection: options.SortDirection,
	}

	closedQuery := &ClosedPositionsQuery{
		User:          &user,
		Limit:         options.Limit,
		Offset:        options.Offset,
		SortBy:        options.SortBy,
		SortDirection: options.SortDirection,
	}

	currentChan := make(chan []Position, 1)
	closedChan := make(chan []ClosedPosition, 1)
	currentErrChan := make(chan error, 1)
	closedErrChan := make(chan error, 1)

	go func() {
		positions, err := d.GetCurrentPositions(currentQuery)
		currentChan <- positions
		currentErrChan <- err
	}()

	go func() {
		positions, err := d.GetClosedPositions(closedQuery)
		closedChan <- positions
		closedErrChan <- err
	}()

	currentPositions := <-currentChan
	closedPositions := <-closedChan
	currentErr := <-currentErrChan
	closedErr := <-closedErrChan

	if currentErr != nil {
		return nil, fmt.Errorf("failed to get current positions: %w", currentErr)
	}
	if closedErr != nil {
		return nil, fmt.Errorf("failed to get closed positions: %w", closedErr)
	}

	return &struct {
		Current []Position
		Closed  []ClosedPosition
	}{
		Current: currentPositions,
		Closed:  closedPositions,
	}, nil
}

func (d *DataSDK) GetPortfolioSummary(user string) (*struct {
	TotalValue       []TotalValue
	MarketsTraded    *TotalMarketsTraded
	CurrentPositions []Position
}, error) {

	totalValueChan := make(chan []TotalValue, 1)
	marketsTradedChan := make(chan *TotalMarketsTraded, 1)
	positionsChan := make(chan []Position, 1)
	totalValueErrChan := make(chan error, 1)
	marketsTradedErrChan := make(chan error, 1)
	positionsErrChan := make(chan error, 1)

	go func() {
		value, err := d.GetTotalValue(&TotalValueQuery{User: &user})
		totalValueChan <- value
		totalValueErrChan <- err
	}()

	go func() {
		traded, err := d.GetTotalMarketsTraded(&TotalMarketsTradedQuery{User: &user})
		marketsTradedChan <- traded
		marketsTradedErrChan <- err
	}()

	go func() {
		positions, err := d.GetCurrentPositions(&PositionsQuery{User: &user})
		positionsChan <- positions
		positionsErrChan <- err
	}()

	totalValue := <-totalValueChan
	marketsTraded := <-marketsTradedChan
	positions := <-positionsChan
	totalValueErr := <-totalValueErrChan
	marketsTradedErr := <-marketsTradedErrChan
	positionsErr := <-positionsErrChan

	if totalValueErr != nil {
		return nil, fmt.Errorf("failed to get total value: %w", totalValueErr)
	}
	if marketsTradedErr != nil {
		return nil, fmt.Errorf("failed to get markets traded: %w", marketsTradedErr)
	}
	if positionsErr != nil {
		return nil, fmt.Errorf("failed to get current positions: %w", positionsErr)
	}

	return &struct {
		TotalValue       []TotalValue
		MarketsTraded    *TotalMarketsTraded
		CurrentPositions []Position
	}{
		TotalValue:       totalValue,
		MarketsTraded:    marketsTraded,
		CurrentPositions: positions,
	}, nil
}

func (d *DataSDK) unmarshalPositionsResponse(resp *APIResponse, operation string) ([]Position, error) {
	data, err := d.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []Position
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (d *DataSDK) unmarshalClosedPositionsResponse(resp *APIResponse, operation string) ([]ClosedPosition, error) {
	data, err := d.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []ClosedPosition
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (d *DataSDK) unmarshalTradesResponse(resp *APIResponse, operation string) ([]DataTrade, error) {
	data, err := d.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []DataTrade
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (d *DataSDK) unmarshalActivityResponse(resp *APIResponse, operation string) ([]Activity, error) {
	data, err := d.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []Activity
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (d *DataSDK) unmarshalMetaHoldersResponse(resp *APIResponse, operation string) ([]MetaHolder, error) {
	data, err := d.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []MetaHolder
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (d *DataSDK) unmarshalTotalValueResponse(resp *APIResponse, operation string) ([]TotalValue, error) {
	data, err := d.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []TotalValue
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (d *DataSDK) unmarshalTotalMarketsTradedResponse(resp *APIResponse, operation string) (*TotalMarketsTraded, error) {
	data, err := d.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result TotalMarketsTraded
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return &result, nil
}

func (d *DataSDK) unmarshalOpenInterestResponse(resp *APIResponse, operation string) ([]OpenInterest, error) {
	data, err := d.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []OpenInterest
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (d *DataSDK) unmarshalLiveVolumeResponse(resp *APIResponse, operation string) (*LiveVolumeResponse, error) {
	data, err := d.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result LiveVolumeResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return &result, nil
}

type APIResponse struct {
	Status    int             `json:"status"`
	OK        bool            `json:"ok"`
	Data      json.RawMessage `json:"data,omitempty"`
	ErrorData interface{}     `json:"errorData,omitempty"`
}
