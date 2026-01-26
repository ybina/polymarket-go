package gamma

import (
	"encoding/json"
	"time"
)

type Team struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	League       string    `json:"league"`
	Record       *string   `json:"record,omitempty"`
	Logo         string    `json:"logo"`
	Abbreviation string    `json:"abbreviation"`
	Alias        *string   `json:"alias,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type TeamQuery struct {
	Limit     *int    `json:"limit,omitempty"`
	Offset    *int    `json:"offset,omitempty"`
	Order     *string `json:"order,omitempty"`
	Ascending *bool   `json:"ascending,omitempty"`
	League    *string `json:"league,omitempty"`
}

type Tag struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Slug       string  `json:"slug"`
	ForceShow  *bool   `json:"forceShow,omitempty"`
	CreatedAt  *string `json:"createdAt,omitempty"`
	IsCarousel *bool   `json:"isCarousel,omitempty"`
}

type UpdatedTag struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Slug        string  `json:"slug"`
	ForceShow   *bool   `json:"forceShow,omitempty"`
	PublishedAt *string `json:"publishedAt,omitempty"`
	CreatedBy   *int    `json:"createdBy,omitempty"`
	UpdatedBy   *int    `json:"updatedBy,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	ForceHide   *bool   `json:"forceHide,omitempty"`
	IsCarousel  *bool   `json:"isCarousel,omitempty"`
}

type TagQuery struct {
	Limit      *int    `json:"limit,omitempty"`
	Offset     *int    `json:"offset,omitempty"`
	Order      *string `json:"order,omitempty"`
	Ascending  *bool   `json:"ascending,omitempty"`
	Search     *string `json:"search,omitempty"`
	IsCarousel *bool   `json:"is_carousel,omitempty"`
}

type TagByIdQuery struct {
	IncludeTemplate *bool `json:"include_template,omitempty"`
}

type RelatedTagRelationship struct {
	SourceTagID      int             `json:"sourceTagId"`
	TargetTagID      int             `json:"targetTagId"`
	RelationshipType string          `json:"relationshipType"`
	TargetTag        UpdatedTag      `json:"targetTag"`
	Relationship     TagRelationship `json:"relationship"`
}

type TagRelationship struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type RelatedTagsQuery struct {
	Limit     *int    `json:"limit,omitempty"`
	Offset    *int    `json:"offset,omitempty"`
	Order     *string `json:"order,omitempty"`
	Ascending *bool   `json:"ascending,omitempty"`
}

type EventMarket struct {
	ID                 string   `json:"id"`
	Question           string   `json:"question"`
	ConditionID        string   `json:"conditionId"`
	Slug               string   `json:"slug"`
	ResolutionSource   *string  `json:"resolutionSource,omitempty"`
	EndDate            *string  `json:"endDate,omitempty"`
	Liquidity          *string  `json:"liquidity,omitempty"`
	StartDate          *string  `json:"startDate,omitempty"`
	Image              string   `json:"image"`
	Icon               string   `json:"icon"`
	Description        string   `json:"description"`
	Outcomes           []string `json:"outcomes"`
	OutcomePrices      []string `json:"outcomePrices"`
	Volume             *string  `json:"volume,omitempty"`
	Active             bool     `json:"active"`
	Closed             bool     `json:"closed"`
	MarketMakerAddress *string  `json:"marketMakerAddress,omitempty"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
	New                *bool    `json:"new,omitempty"`
	ClobTokenIDs       []string `json:"clobTokenIds"`
}

type Event struct {
	ID                    string        `json:"id"`
	Ticker                string        `json:"ticker"`
	Slug                  string        `json:"slug"`
	Title                 string        `json:"title"`
	Description           *string       `json:"description,omitempty"`
	ResolutionSource      *string       `json:"resolutionSource,omitempty"`
	StartDate             *string       `json:"startDate,omitempty"`
	CreationDate          *string       `json:"creationDate,omitempty"`
	EndDate               *string       `json:"endDate,omitempty"`
	Image                 string        `json:"image"`
	Icon                  string        `json:"icon"`
	Active                bool          `json:"active"`
	Closed                bool          `json:"closed"`
	Archived              bool          `json:"archived"`
	New                   *bool         `json:"new,omitempty"`
	Featured              *bool         `json:"featured,omitempty"`
	Restricted            *bool         `json:"restricted,omitempty"`
	Liquidity             *float64      `json:"liquidity,omitempty"`
	Volume                *float64      `json:"volume,omitempty"`
	Volume24hr            *float64      `json:"volume24hr,omitempty"`
	VolumeNum             *float64      `json:"volumeNum,omitempty"`
	LastActiveAt          *string       `json:"lastActiveAt,omitempty"`
	LiquidityAmm          *float64      `json:"liquidityAmm,omitempty"`
	LiquidityNum          *float64      `json:"liquidityNum,omitempty"`
	Markets               []EventMarket `json:"markets"`
	Series                []Series      `json:"series,omitempty"`
	Tags                  []Tag         `json:"tags,omitempty"`
	Cyom                  *bool         `json:"cyom,omitempty"`
	ShowAllOutcomes       *bool         `json:"showAllOutcomes,omitempty"`
	ShowMarketImages      *bool         `json:"showMarketImages,omitempty"`
	EnableNegRisk         *bool         `json:"enableNegRisk,omitempty"`
	AutomaticallyActive   *bool         `json:"automaticallyActive,omitempty"`
	SeriesSlug            *string       `json:"seriesSlug,omitempty"`
	GmpChartMode          *string       `json:"gmpChartMode,omitempty"`
	NegRiskAugmented      *bool         `json:"negRiskAugmented,omitempty"`
	PendingDeployment     *bool         `json:"pendingDeployment,omitempty"`
	Deploying             *bool         `json:"deploying,omitempty"`
	SortBy                *string       `json:"sortBy,omitempty"`
	ClosedTime            *string       `json:"closedTime,omitempty"`
	AutomaticallyResolved *bool         `json:"automaticallyResolved,omitempty"`
}

type UpdatedEventQuery struct {
	Limit        *int     `json:"limit,omitempty"`
	Offset       *int     `json:"offset,omitempty"`
	Order        *string  `json:"order,omitempty"`
	Ascending    *bool    `json:"ascending,omitempty"`
	Search       *string  `json:"search,omitempty"`
	Active       *bool    `json:"active,omitempty"`
	Closed       *bool    `json:"closed,omitempty"`
	Archived     *bool    `json:"archived,omitempty"`
	Featured     *bool    `json:"featured,omitempty"`
	New          *bool    `json:"new,omitempty"`
	Restricted   *bool    `json:"restricted,omitempty"`
	MinVolume    *float64 `json:"minVolume,omitempty"`
	MaxVolume    *float64 `json:"maxVolume,omitempty"`
	MinLiquidity *float64 `json:"minLiquidity,omitempty"`
	MaxLiquidity *float64 `json:"maxLiquidity,omitempty"`
	Series       *string  `json:"series,omitempty"`
	Tag          *string  `json:"tag,omitempty"`
	StartDate    *string  `json:"startDate,omitempty"`
	EndDate      *string  `json:"endDate,omitempty"`
}

type PaginatedEventQuery struct {
	Limit      *int    `json:"limit,omitempty"`
	Offset     *int    `json:"offset,omitempty"`
	Order      *string `json:"order,omitempty"`
	Ascending  *bool   `json:"ascending,omitempty"`
	Search     *string `json:"search,omitempty"`
	Active     *bool   `json:"active,omitempty"`
	Closed     *bool   `json:"closed,omitempty"`
	Archived   *bool   `json:"archived,omitempty"`
	Featured   *bool   `json:"featured,omitempty"`
	New        *bool   `json:"new,omitempty"`
	Restricted *bool   `json:"restricted,omitempty"`
	Series     *string `json:"series,omitempty"`
	Tag        *string `json:"tag,omitempty"`
	StartDate  *string `json:"startDate,omitempty"`
	EndDate    *string `json:"endDate,omitempty"`
}

type EventByIdQuery struct {
	IncludeChat *bool `json:"include_chat,omitempty"`
}

type Market struct {
	ID               string   `json:"id"`
	Question         string   `json:"question"`
	ConditionID      string   `json:"conditionId"`
	Slug             string   `json:"slug"`
	Liquidity        *string  `json:"liquidity,omitempty"`
	StartDate        *string  `json:"startDate,omitempty"`
	Image            string   `json:"image"`
	Icon             string   `json:"icon"`
	Description      string   `json:"description"`
	Active           bool     `json:"active"`
	Volume           string   `json:"volume"`
	Outcomes         []string `json:"outcomes"`
	OutcomePrices    []string `json:"outcomePrices"`
	Closed           bool     `json:"closed"`
	New              *bool    `json:"new,omitempty"`
	QuestionID       *string  `json:"questionId,omitempty"`
	VolumeNum        float64  `json:"volumeNum"`
	LiquidityNum     *float64 `json:"liquidityNum,omitempty"`
	StartDateIso     *string  `json:"startDateIso,omitempty"`
	HasReviewedDates *bool    `json:"hasReviewedDates,omitempty"`
	ClobTokenIDs     []string `json:"clobTokenIds"`
	EndDate          *string  `json:"endDate,omitempty"`
	LastActiveAt     *string  `json:"lastActiveAt,omitempty"`
}

type UpdatedMarketQuery struct {
	Limit     *int    `json:"limit,omitempty"`
	Offset    *int    `json:"offset,omitempty"`
	Order     *string `json:"order,omitempty"`
	Ascending *bool   `json:"ascending,omitempty"`

	ID              []int    `json:"id,omitempty"`
	Slug            []string `json:"slug,omitempty"`
	ClobTokenIDs    *string  `json:"clob_token_ids,omitempty"`
	ConditionIDs    []string `json:"condition_ids,omitempty"`
	QuestionIDs     []string `json:"question_ids,omitempty"`
	MarketMakerAddr []string `json:"market_maker_address,omitempty"`

	LiquidityNumMin *float64 `json:"liquidity_num_min,omitempty"`
	LiquidityNumMax *float64 `json:"liquidity_num_max,omitempty"`
	VolumeNumMin    *float64 `json:"volume_num_min,omitempty"`
	VolumeNumMax    *float64 `json:"volume_num_max,omitempty"`
	RewardsMinSize  *float64 `json:"rewards_min_size,omitempty"`

	StartDateMin *string `json:"start_date_min,omitempty"`
	StartDateMax *string `json:"start_date_max,omitempty"`
	EndDateMin   *string `json:"end_date_min,omitempty"`
	EndDateMax   *string `json:"end_date_max,omitempty"`

	TagID               *int    `json:"tag_id,omitempty"`
	RelatedTags         *bool   `json:"related_tags,omitempty"`
	IncludeTag          *bool   `json:"include_tag,omitempty"`
	Closed              *bool   `json:"closed,omitempty"`
	CYOM                *bool   `json:"cyom,omitempty"`
	UMAResolutionStatus *string `json:"uma_resolution_status,omitempty"`

	GameID            *string  `json:"game_id,omitempty"`
	SportsMarketTypes []string `json:"sports_market_types,omitempty"`
}

type MarketByIdQuery struct {
	IncludeTag *bool `json:"include_tag,omitempty"`
}

type Series struct {
	ID            string   `json:"id"`
	Ticker        string   `json:"ticker"`
	Slug          string   `json:"slug"`
	Title         string   `json:"title"`
	Subtitle      *string  `json:"subtitle,omitempty"`
	SeriesType    *string  `json:"seriesType,omitempty"`
	Recurrence    *string  `json:"recurrence,omitempty"`
	Image         *string  `json:"image,omitempty"`
	Icon          *string  `json:"icon,omitempty"`
	Active        bool     `json:"active"`
	Closed        bool     `json:"closed"`
	Archived      bool     `json:"archived"`
	Volume        *float64 `json:"volume,omitempty"`
	Liquidity     *float64 `json:"liquidity,omitempty"`
	StartDate     *string  `json:"startDate,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
	Competitive   *float64 `json:"competitive,omitempty"`
	Volume24hr    *float64 `json:"volume24hr,omitempty"`
	PythTokenID   *string  `json:"pythTokenId,omitempty"`
	LastActiveAt  *string  `json:"lastActiveAt,omitempty"`
	SeriesTypeMap *string  `json:"seriesTypeMap,omitempty"`
}

type SeriesQuery struct {
	Limit     *int     `json:"limit,omitempty"`
	Offset    *int     `json:"offset,omitempty"`
	Order     *string  `json:"order,omitempty"`
	Ascending *bool    `json:"ascending,omitempty"`
	Search    *string  `json:"search,omitempty"`
	Active    *bool    `json:"active,omitempty"`
	Closed    *bool    `json:"closed,omitempty"`
	Archived  *bool    `json:"archived,omitempty"`
	MinVolume *float64 `json:"minVolume,omitempty"`
	MaxVolume *float64 `json:"maxVolume,omitempty"`
	StartDate *string  `json:"startDate,omitempty"`
	EndDate   *string  `json:"endDate,omitempty"`
}

type SeriesByIdQuery struct {
	IncludeChat *bool `json:"include_chat,omitempty"`
}

type Comment struct {
	ID               string      `json:"id"`
	Body             string      `json:"body"`
	ParentEntityType string      `json:"parentEntityType"`
	ParentEntityID   int         `json:"parentEntityID"`
	UserAddress      string      `json:"userAddress"`
	CreatedAt        string      `json:"createdAt"`
	Profile          interface{} `json:"profile,omitempty"`
	Reactions        interface{} `json:"reactions,omitempty"`
	ReportCount      int         `json:"reportCount"`
	ReactionCount    int         `json:"reactionCount"`
}

type CommentQuery struct {
	Limit            *int    `json:"limit,omitempty"`
	Offset           *int    `json:"offset,omitempty"`
	Order            *string `json:"order,omitempty"`
	Ascending        *bool   `json:"ascending,omitempty"`
	ParentEntityType *string `json:"parent_entity_type,omitempty"`
	ParentEntityID   *int    `json:"parent_entity_id,omitempty"`
}

type CommentByIdQuery struct {
	Limit     *int    `json:"limit,omitempty"`
	Offset    *int    `json:"offset,omitempty"`
	Order     *string `json:"order,omitempty"`
	Ascending *bool   `json:"ascending,omitempty"`
}

type CommentsByUserQuery struct {
	Limit     *int    `json:"limit,omitempty"`
	Offset    *int    `json:"offset,omitempty"`
	Order     *string `json:"order,omitempty"`
	Ascending *bool   `json:"ascending,omitempty"`
}

type SearchQuery struct {
	Q              *string `json:"q,omitempty"`
	LimitPerType   *int    `json:"limit_per_type,omitempty"`
	EventsStatus   *string `json:"events_status,omitempty"`
	EventsActive   *bool   `json:"events_active,omitempty"`
	EventsClosed   *bool   `json:"events_closed,omitempty"`
	EventsArchived *bool   `json:"events_archived,omitempty"`
	EventsFeatured *bool   `json:"events_featured,omitempty"`
	MarketsActive  *bool   `json:"markets_active,omitempty"`
	MarketsClosed  *bool   `json:"markets_closed,omitempty"`
	TagsCarousel   *bool   `json:"tags_carousel,omitempty"`
	SeriesActive   *bool   `json:"series_active,omitempty"`
	SeriesClosed   *bool   `json:"series_closed,omitempty"`
}

type SearchResponse struct {
	Events     []interface{} `json:"events,omitempty"`
	Tags       []interface{} `json:"tags,omitempty"`
	Profiles   []interface{} `json:"profiles,omitempty"`
	Pagination *Pagination   `json:"pagination,omitempty"`
}

type Pagination struct {
	HasMore bool `json:"hasMore"`
}

type APIResponse struct {
	Data      json.RawMessage `json:"data"`
	Status    int             `json:"status"`
	OK        bool            `json:"ok"`
	ErrorData interface{}     `json:"errorData,omitempty"`
}

type GammaError struct {
	Message   string `json:"message"`
	Code      int    `json:"code"`
	Timestamp string `json:"timestamp"`
	Path      string `json:"path"`
}

type PaginatedEventsResponse struct {
	Data       []Event    `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type IPResponse struct {
	IP       string `json:"ip"`
	Country  string `json:"country,omitempty"`
	Region   string `json:"region,omitempty"`
	City     string `json:"city,omitempty"`
	ISP      string `json:"isp,omitempty"`
	Org      string `json:"org,omitempty"`
	AS       string `json:"as,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

func StringPtr(s string) *string {
	return &s
}

func IntPtr(i int) *int {
	return &i
}

func BoolPtr(b bool) *bool {
	return &b
}
