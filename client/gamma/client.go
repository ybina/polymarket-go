package gamma

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	GammaAPIBase = "https://gamma-api.polymarket.com"
)

type GammaSDK struct {
	baseURL    string
	proxyUrl   *string
	httpClient *http.Client
}

func NewGammaSDK(proxyUrl *string) (*GammaSDK, error) {

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	client := &GammaSDK{
		baseURL:    GammaAPIBase,
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

func (g *GammaSDK) GetHttpClient() *http.Client {
	return g.httpClient
}

func (g *GammaSDK) buildURL(endpoint string, query interface{}) (string, error) {
	u, err := url.Parse(g.baseURL + endpoint)
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

			values.Add(fieldName, strValue)
		}

		u.RawQuery = values.Encode()
	}

	return u.String(), nil
}

func (g *GammaSDK) createRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "gamma-go-sdk/1.0")

	return req, nil
}

func (g *GammaSDK) makeRequest(method, endpoint string, query interface{}) (*APIResponse, error) {

	fullURL, err := g.buildURL(endpoint, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	req, err := g.createRequest(method, fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
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
			var errData GammaError
			if err := sonic.Unmarshal(body, &errData); err == nil {
				apiResp.ErrorData = errData
			} else {
				apiResp.ErrorData = string(body)
			}
		} else {
			apiResp.Data = body
		}
	}

	return apiResp, nil
}

func (g *GammaSDK) extractResponseData(resp *APIResponse, operation string) ([]byte, error) {
	if !resp.OK {
		return nil, fmt.Errorf("[GammaSDK] %s failed: status %d", operation, resp.Status)
	}

	if resp.Data == nil {
		return nil, fmt.Errorf("[GammaSDK] %s returned null data despite successful response", operation)
	}

	return resp.Data, nil
}

func (g *GammaSDK) unmarshalTeamsResponse(resp *APIResponse, operation string) ([]Team, error) {
	data, err := g.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []Team
	if err := sonic.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (g *GammaSDK) unmarshalTagsResponse(resp *APIResponse, operation string) ([]UpdatedTag, error) {
	data, err := g.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []UpdatedTag
	if err := sonic.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (g *GammaSDK) unmarshalTagResponse(resp *APIResponse, operation string) (*UpdatedTag, error) {
	if resp.Status == 404 {
		return nil, nil
	}

	data, err := g.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result UpdatedTag
	if err := sonic.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return &result, nil
}

func (g *GammaSDK) unmarshalRelatedTagRelationshipsResponse(resp *APIResponse, operation string) ([]RelatedTagRelationship, error) {
	data, err := g.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []RelatedTagRelationship
	if err := sonic.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (g *GammaSDK) unmarshalSeriesResponse(resp *APIResponse, operation string) ([]Series, error) {
	data, err := g.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []Series
	if err := sonic.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (g *GammaSDK) unmarshalSeriesSingleResponse(resp *APIResponse, operation string) (*Series, error) {
	if resp.Status == 404 {
		return nil, nil
	}

	data, err := g.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result Series
	if err := sonic.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return &result, nil
}

func (g *GammaSDK) unmarshalCommentsResponse(resp *APIResponse, operation string) ([]Comment, error) {
	data, err := g.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result []Comment
	if err := sonic.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return result, nil
}

func (g *GammaSDK) unmarshalEventsResponse(resp *APIResponse, operation string) ([]Event, error) {
	data, err := g.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var rawItems []map[string]interface{}
	if err := sonic.Unmarshal(data, &rawItems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s data: %w", operation, err)
	}

	events := make([]Event, len(rawItems))
	for i, item := range rawItems {
		events[i] = g.transformEventData(item)
	}

	return events, nil
}

func (g *GammaSDK) unmarshalMarketsResponse(resp *APIResponse, operation string) ([]Market, error) {
	data, err := g.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var rawItems []map[string]interface{}
	if err := sonic.Unmarshal(data, &rawItems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s data: %w", operation, err)
	}

	markets := make([]Market, len(rawItems))
	for i, item := range rawItems {
		markets[i] = g.transformMarketData(item)
	}

	return markets, nil
}

func (g *GammaSDK) unmarshalSearchResponse(resp *APIResponse, operation string) (*SearchResponse, error) {
	data, err := g.extractResponseData(resp, operation)
	if err != nil {
		return nil, err
	}

	var result SearchResponse
	if err := sonic.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", operation, err)
	}

	return &result, nil
}

func (g *GammaSDK) parseJSONArray(value interface{}) []string {
	if value == nil {
		return []string{}
	}

	switch v := value.(type) {
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	case []string:
		return v
	case string:
		var result []string
		if err := sonic.Unmarshal([]byte(v), &result); err != nil {
			// If parsing fails, return empty array
			return []string{}
		}
		return result
	default:
		return []string{fmt.Sprintf("%v", v)}
	}
}

func (g *GammaSDK) transformMarketData(item map[string]interface{}) Market {
	market := Market{}

	itemBytes, _ := sonic.Marshal(item)
	err := sonic.Unmarshal(itemBytes, &market)
	if err != nil {
		log.Printf("[GammaSDK] unmarshal Market failed: %v, raw=%s", err, string(itemBytes))
		return Market{}
	}

	if outcomes, ok := item["outcomes"]; ok {
		market.Outcomes = g.parseJSONArray(outcomes)
	}

	if outcomePrices, ok := item["outcomePrices"]; ok {
		market.OutcomePrices = g.parseJSONArray(outcomePrices)
	}

	if clobTokenIds, ok := item["clobTokenIds"]; ok {
		market.ClobTokenIDs = g.parseJSONArray(clobTokenIds)
	}

	return market
}

func (g *GammaSDK) transformEventData(item map[string]interface{}) Event {
	event := Event{}

	itemBytes, _ := sonic.Marshal(item)
	err := sonic.Unmarshal(itemBytes, &event)
	if err != nil {
		return Event{}
	}

	if marketsData, ok := item["markets"]; ok {
		if markets, ok := marketsData.([]interface{}); ok {
			event.Markets = make([]EventMarket, len(markets))
			for i, marketItem := range markets {
				if marketMap, ok := marketItem.(map[string]interface{}); ok {
					market := EventMarket{}
					marketBytes, _ := sonic.Marshal(marketMap)
					err = sonic.Unmarshal(marketBytes, &market)
					if err != nil {
						return Event{}
					}

					if outcomes, ok := marketMap["outcomes"]; ok {
						market.Outcomes = g.parseJSONArray(outcomes)
					}

					if outcomePrices, ok := marketMap["outcomePrices"]; ok {
						market.OutcomePrices = g.parseJSONArray(outcomePrices)
					}

					if clobTokenIds, ok := marketMap["clobTokenIds"]; ok {
						market.ClobTokenIDs = g.parseJSONArray(clobTokenIds)
					}

					event.Markets[i] = market
				}
			}
		}
	}
	return event
}

func (g *GammaSDK) GetHealth() (map[string]interface{}, error) {
	resp, err := g.makeRequest("GET", "/health", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if resp.Data != nil {
		if err := sonic.Unmarshal(resp.Data, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal health response: %w", err)
		}
	}

	return result, nil
}

func (g *GammaSDK) GetTeams(query *TeamQuery) ([]Team, error) {
	if query == nil {
		query = &TeamQuery{}
	}

	resp, err := g.makeRequest("GET", "/teams", query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalTeamsResponse(resp, "Get teams")
}

func (g *GammaSDK) GetTags(query TagQuery) ([]UpdatedTag, error) {
	resp, err := g.makeRequest("GET", "/tags", query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalTagsResponse(resp, "Get tags")
}

func (g *GammaSDK) GetTagById(id int, query *TagByIdQuery) (*UpdatedTag, error) {
	if query == nil {
		query = &TagByIdQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/tags/%d", id), query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalTagResponse(resp, "Get tag by ID")
}

func (g *GammaSDK) GetTagBySlug(slug string, query *TagByIdQuery) (*UpdatedTag, error) {
	if query == nil {
		query = &TagByIdQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/tags/slug/%s", slug), query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalTagResponse(resp, "Get tag by slug")
}

func (g *GammaSDK) GetRelatedTagsRelationshipsByTagId(id int, query *RelatedTagsQuery) ([]RelatedTagRelationship, error) {
	if query == nil {
		query = &RelatedTagsQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/tags/%d/related-tags", id), query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalRelatedTagRelationshipsResponse(resp, "Get related tags relationships")
}

func (g *GammaSDK) GetRelatedTagsRelationshipsByTagSlug(slug string, query *RelatedTagsQuery) ([]RelatedTagRelationship, error) {
	if query == nil {
		query = &RelatedTagsQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/tags/slug/%s/related-tags", slug), query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalRelatedTagRelationshipsResponse(resp, "Get related tags relationships")
}

func (g *GammaSDK) GetTagsRelatedToTagId(id int, query *RelatedTagsQuery) ([]UpdatedTag, error) {
	if query == nil {
		query = &RelatedTagsQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/tags/%d/related-tags/tags", id), query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalTagsResponse(resp, "Get related tags")
}

func (g *GammaSDK) GetTagsRelatedToTagSlug(slug string, query *RelatedTagsQuery) ([]UpdatedTag, error) {
	if query == nil {
		query = &RelatedTagsQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/tags/slug/%s/related-tags/tags", slug), query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalTagsResponse(resp, "Get related tags")
}

func (g *GammaSDK) GetEvents(query *UpdatedEventQuery) ([]Event, error) {
	if query == nil {
		query = &UpdatedEventQuery{}
	}

	resp, err := g.makeRequest("GET", "/events", query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalEventsResponse(resp, "Get events")
}

func (g *GammaSDK) GetEventsPaginated(query PaginatedEventQuery) (*PaginatedEventsResponse, error) {
	resp, err := g.makeRequest("GET", "/events/pagination", query)
	if err != nil {
		return nil, err
	}

	data, err := g.extractResponseData(resp, "Get paginated events")
	if err != nil {
		return nil, err
	}

	var rawResponse struct {
		Data       []map[string]interface{} `json:"data"`
		Pagination Pagination               `json:"pagination"`
	}

	if err := sonic.Unmarshal(data, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal paginated events response: %w", err)
	}

	events := make([]Event, len(rawResponse.Data))
	for i, item := range rawResponse.Data {
		events[i] = g.transformEventData(item)
	}

	return &PaginatedEventsResponse{
		Data:       events,
		Pagination: rawResponse.Pagination,
	}, nil
}

func (g *GammaSDK) GetEventById(id int, query *EventByIdQuery) (*Event, error) {
	if query == nil {
		query = &EventByIdQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/events/%d", id), query)
	if err != nil {
		return nil, err
	}

	if resp.Status == 404 {
		return nil, nil
	}

	data, err := g.extractResponseData(resp, "Get event by ID")
	if err != nil {
		return nil, err
	}

	var rawEvent map[string]interface{}
	if err := sonic.Unmarshal(data, &rawEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	event := g.transformEventData(rawEvent)
	return &event, nil
}

func (g *GammaSDK) GetEventTags(id int) ([]UpdatedTag, error) {
	resp, err := g.makeRequest("GET", fmt.Sprintf("/events/%d/tags", id), nil)
	if err != nil {
		return nil, err
	}

	return g.unmarshalTagsResponse(resp, "Get event tags")
}

func (g *GammaSDK) GetEventBySlug(slug string, query *EventByIdQuery) (*Event, error) {
	if query == nil {
		query = &EventByIdQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/events/slug/%s", slug), query)
	if err != nil {
		return nil, err
	}

	if resp.Status == 404 {
		return nil, nil
	}

	data, err := g.extractResponseData(resp, "Get event by slug")
	if err != nil {
		return nil, err
	}

	var rawEvent map[string]interface{}
	if err := sonic.Unmarshal(data, &rawEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	event := g.transformEventData(rawEvent)
	return &event, nil
}

func (g *GammaSDK) GetMarkets(query *UpdatedMarketQuery) ([]Market, error) {
	if query == nil {
		query = &UpdatedMarketQuery{}
	}

	resp, err := g.makeRequest("GET", "/markets", query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalMarketsResponse(resp, "Get markets")
}

func (g *GammaSDK) GetMarketById(id int, query *MarketByIdQuery) (*Market, error) {
	if query == nil {
		query = &MarketByIdQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/markets/%d", id), query)
	if err != nil {
		return nil, err
	}

	if resp.Status == 404 {
		return nil, nil
	}

	data, err := g.extractResponseData(resp, "Get market by ID")
	if err != nil {
		return nil, err
	}

	// Parse and transform the market
	var rawMarket map[string]interface{}
	if err := sonic.Unmarshal(data, &rawMarket); err != nil {
		return nil, fmt.Errorf("failed to unmarshal market data: %w", err)
	}

	market := g.transformMarketData(rawMarket)
	return &market, nil
}

func (g *GammaSDK) GetMarketTags(id int) ([]UpdatedTag, error) {
	resp, err := g.makeRequest("GET", fmt.Sprintf("/markets/%d/tags", id), nil)
	if err != nil {
		return nil, err
	}

	return g.unmarshalTagsResponse(resp, "Get market tags")
}

func (g *GammaSDK) GetMarketBySlug(slug string, query *MarketByIdQuery) (*Market, error) {
	if query == nil {
		query = &MarketByIdQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/markets/slug/%s", slug), query)
	if err != nil {
		return nil, err
	}

	if resp.Status == 404 {
		return nil, nil
	}

	data, err := g.extractResponseData(resp, "Get market by slug")
	if err != nil {
		return nil, err
	}

	// Parse and transform the market
	var rawMarket map[string]interface{}
	if err := sonic.Unmarshal(data, &rawMarket); err != nil {
		return nil, fmt.Errorf("failed to unmarshal market data: %w", err)
	}

	market := g.transformMarketData(rawMarket)
	return &market, nil
}

func (g *GammaSDK) GetSeries(query SeriesQuery) ([]Series, error) {
	resp, err := g.makeRequest("GET", "/series", query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalSeriesResponse(resp, "Get series")
}

func (g *GammaSDK) GetSeriesById(id int, query *SeriesByIdQuery) (*Series, error) {
	if query == nil {
		query = &SeriesByIdQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/series/%d", id), query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalSeriesSingleResponse(resp, "Get series by ID")
}

func (g *GammaSDK) GetComments(query *CommentQuery) ([]Comment, error) {
	if query == nil {
		query = &CommentQuery{}
	}

	resp, err := g.makeRequest("GET", "/comments", query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalCommentsResponse(resp, "Get comments")
}

func (g *GammaSDK) GetCommentsByCommentId(id int, query *CommentByIdQuery) ([]Comment, error) {
	if query == nil {
		query = &CommentByIdQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/comments/%d", id), query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalCommentsResponse(resp, "Get comments by comment ID")
}

func (g *GammaSDK) GetCommentsByUserAddress(userAddress string, query *CommentsByUserQuery) ([]Comment, error) {
	if query == nil {
		query = &CommentsByUserQuery{}
	}

	resp, err := g.makeRequest("GET", fmt.Sprintf("/comments/user_address/%s", userAddress), query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalCommentsResponse(resp, "Get comments by user address")
}

func (g *GammaSDK) Search(query SearchQuery) (*SearchResponse, error) {
	resp, err := g.makeRequest("GET", "/public-search", query)
	if err != nil {
		return nil, err
	}

	return g.unmarshalSearchResponse(resp, "Search")
}

func (g *GammaSDK) GetActiveEvents(query *UpdatedEventQuery) ([]Event, error) {
	if query == nil {
		query = &UpdatedEventQuery{}
	}

	active := true
	query.Active = &active
	return g.GetEvents(query)
}

func (g *GammaSDK) GetClosedEvents(query *UpdatedEventQuery) ([]Event, error) {
	if query == nil {
		query = &UpdatedEventQuery{}
	}

	closed := true
	query.Closed = &closed
	return g.GetEvents(query)
}

func (g *GammaSDK) GetFeaturedEvents(query *UpdatedEventQuery) ([]Event, error) {
	if query == nil {
		query = &UpdatedEventQuery{}
	}

	featured := true
	query.Featured = &featured
	return g.GetEvents(query)
}

func (g *GammaSDK) GetActiveMarkets(query *UpdatedMarketQuery) ([]Market, error) {
	if query == nil {
		query = &UpdatedMarketQuery{}
	}

	return g.GetMarkets(query)
}

func (g *GammaSDK) GetClosedMarkets(query *UpdatedMarketQuery) ([]Market, error) {
	if query == nil {
		query = &UpdatedMarketQuery{}
	}

	closed := true
	query.Closed = &closed
	return g.GetMarkets(query)
}

func (g *GammaSDK) TestProxyIP() (*IPResponse, error) {
	// List of IP detection services to try (in order of preference)
	services := []string{
		"https://ipinfo.io/json",
		"https://api.ipify.org?format=json",
		"https://api.my-ip.io/v1/ip",
		"https://checkip.amazonaws.com",
	}

	for _, service := range services {
		req, err := http.NewRequest("GET", service, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", "gamma-go-sdk/1.0")
		req.Header.Set("Accept", "application/json")

		resp, err := g.httpClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		var ipResp IPResponse
		if err := sonic.Unmarshal(body, &ipResp); err != nil {

			ipResp.IP = strings.TrimSpace(string(body))
		}

		if ipResp.IP != "" && ipResp.IP != "0.0.0.0" {
			return &ipResp, nil
		}
	}

	return nil, fmt.Errorf("failed to get IP address from any detection service")
}

func (g *GammaSDK) TestProxyIPComparison() (*struct {
	DirectIP   *IPResponse `json:"direct_ip"`
	ProxyIP    *IPResponse `json:"proxy_ip"`
	UsingProxy bool        `json:"using_proxy"`
}, error) {
	directClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	var directIP *IPResponse
	services := []string{
		"https://ipinfo.io/json",
		"https://api.ipify.org?format=json",
		"https://api.my-ip.io/v1/ip",
	}

	for _, service := range services {
		req, err := http.NewRequest("GET", service, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", "gamma-go-sdk/1.0")
		resp, err := directClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		var ipResp IPResponse
		if err := sonic.Unmarshal(body, &ipResp); err != nil {
			ipResp.IP = strings.TrimSpace(string(body))
		}

		if ipResp.IP != "" && ipResp.IP != "0.0.0.0" {
			directIP = &ipResp
			break
		}
	}

	proxyIP, err := g.TestProxyIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy IP: %w", err)
	}

	usingProxy := false
	if directIP != nil && proxyIP != nil {
		usingProxy = directIP.IP != proxyIP.IP
	}

	return &struct {
		DirectIP   *IPResponse `json:"direct_ip"`
		ProxyIP    *IPResponse `json:"proxy_ip"`
		UsingProxy bool        `json:"using_proxy"`
	}{
		DirectIP:   directIP,
		ProxyIP:    proxyIP,
		UsingProxy: usingProxy,
	}, nil
}
