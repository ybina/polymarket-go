package data

type Position struct {
	ProxyWallet        string  `json:"proxyWallet"`
	Asset              string  `json:"asset"`
	ConditionID        string  `json:"conditionId"`
	Size               float64 `json:"size"`
	AvgPrice           float64 `json:"avgPrice"`
	InitialValue       float64 `json:"initialValue"`
	CurrentValue       float64 `json:"currentValue"`
	CashPnl            float64 `json:"cashPnl"`
	PercentPnl         float64 `json:"percentPnl"`
	TotalBought        float64 `json:"totalBought"`
	RealizedPnl        float64 `json:"realizedPnl"`
	PercentRealizedPnl float64 `json:"percentRealizedPnl"`
	CurPrice           float64 `json:"curPrice"`
	Redeemable         bool    `json:"redeemable"`
	Mergeable          bool    `json:"mergeable"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Icon               string  `json:"icon"`
	EventID            string  `json:"eventId"`
	EventSlug          string  `json:"eventSlug"`
	Outcome            string  `json:"outcome"`
	OutcomeIndex       int     `json:"outcomeIndex"`
	OppositeOutcome    string  `json:"oppositeOutcome"`
	OppositeAsset      string  `json:"oppositeAsset"`
	EndDate            *string `json:"endDate,omitempty"`
	NegativeRisk       *bool   `json:"negativeRisk,omitempty"`
}

type ClosedPosition struct {
	ProxyWallet     string  `json:"proxyWallet"`
	Asset           string  `json:"asset"`
	ConditionID     string  `json:"conditionId"`
	Size            float64 `json:"size"`
	AvgPrice        float64 `json:"avgPrice"`
	RealizedPnl     float64 `json:"realizedPnl"`
	ClosedPrice     float64 `json:"closedPrice"`
	ClosedAt        string  `json:"closedAt"`
	Title           string  `json:"title"`
	Slug            string  `json:"slug"`
	Icon            string  `json:"icon"`
	EventID         string  `json:"eventId"`
	EventSlug       string  `json:"eventSlug"`
	Outcome         string  `json:"outcome"`
	OutcomeIndex    int     `json:"outcomeIndex"`
	OppositeOutcome string  `json:"oppositeOutcome"`
	OppositeAsset   string  `json:"oppositeAsset"`
	NegativeRisk    *bool   `json:"negativeRisk,omitempty"`
}

type DataTrade struct {
	ProxyWallet           string   `json:"proxyWallet"`
	Side                  string   `json:"side"`
	ConditionID           string   `json:"conditionId"`
	Outcome               string   `json:"outcome"`
	Market                string   `json:"market"`
	Size                  float64  `json:"size"`
	Price                 float64  `json:"price"`
	Fee                   *float64 `json:"fee,omitempty"`
	Timestamp             int64    `json:"timestamp"`
	TransactionHash       string   `json:"transactionHash"`
	Maker                 string   `json:"maker"`
	Taker                 string   `json:"taker"`
	AssetID               string   `json:"assetId"`
	Title                 string   `json:"title"`
	Slug                  string   `json:"slug"`
	Icon                  string   `json:"icon"`
	EventSlug             string   `json:"eventSlug"`
	OutcomeIndex          int      `json:"outcomeIndex"`
	Name                  string   `json:"name"`
	Pseudonym             string   `json:"pseudonym"`
	Bio                   string   `json:"bio"`
	ProfileImage          string   `json:"profileImage"`
	ProfileImageOptimized string   `json:"profileImageOptimized"`
}

type Activity struct {
	ProxyWallet           string   `json:"proxyWallet"`
	Timestamp             int64    `json:"timestamp"`
	Type                  string   `json:"type"`
	Size                  float64  `json:"size"`
	UsdcSize              float64  `json:"usdcSize"`
	Price                 *float64 `json:"price,omitempty"`
	Fee                   *float64 `json:"fee,omitempty"`
	ConditionID           string   `json:"conditionId"`
	Outcome               string   `json:"outcome"`
	Market                string   `json:"market"`
	TransactionHash       string   `json:"transactionHash"`
	From                  string   `json:"from"`
	To                    string   `json:"to"`
	AssetID               string   `json:"assetId"`
	Value                 *float64 `json:"value,omitempty"`
	Title                 string   `json:"title"`
	Slug                  string   `json:"slug"`
	Icon                  string   `json:"icon"`
	EventSlug             string   `json:"eventSlug"`
	OutcomeIndex          int      `json:"outcomeIndex"`
	Name                  string   `json:"name"`
	Pseudonym             string   `json:"pseudonym"`
	Bio                   string   `json:"bio"`
	ProfileImage          string   `json:"profileImage"`
	ProfileImageOptimized string   `json:"profileImageOptimized"`
}

type Holder struct {
	Wallet  string `json:"wallet"`
	Balance string `json:"balance"`
	Value   string `json:"value"`
}

type MetaHolder struct {
	Token   string   `json:"token"`
	Holders []Holder `json:"holders"`
}

type TotalValue struct {
	User  string  `json:"user"`
	Value float64 `json:"value"`
}

type TotalMarketsTraded struct {
	User   string `json:"user"`
	Traded int    `json:"traded"`
}

type OpenInterest struct {
	Market string  `json:"market"`
	Value  float64 `json:"value"`
}

type LiveVolumeMarket struct {
	Market string  `json:"market"`
	Value  float64 `json:"value"`
}

type LiveVolumeResponse struct {
	Total   int                `json:"total"`
	Markets []LiveVolumeMarket `json:"markets"`
}

type DataHealthResponse struct {
	Data string `json:"data"`
}

type PositionsQuery struct {
	User          *string   `json:"user,omitempty"`
	Market        *[]string `json:"market,omitempty"`
	EventID       *[]string `json:"eventId,omitempty"`
	SizeThreshold *float64  `json:"sizeThreshold,omitempty"`
	Redeemable    *bool     `json:"redeemable,omitempty"`
	Mergeable     *bool     `json:"mergeable,omitempty"`
	Limit         *int      `json:"limit,omitempty"`
	Offset        *int      `json:"offset,omitempty"`
	SortBy        *string   `json:"sortBy,omitempty"`
	SortDirection *string   `json:"sortDirection,omitempty"`
	Title         *string   `json:"title,omitempty"`
}

type ClosedPositionsQuery struct {
	User          *string   `json:"user,omitempty"`
	Market        *[]string `json:"market,omitempty"`
	EventID       *[]string `json:"eventId,omitempty"`
	Title         *string   `json:"title,omitempty"`
	Limit         *int      `json:"limit,omitempty"`
	Offset        *int      `json:"offset,omitempty"`
	SortBy        *string   `json:"sortBy,omitempty"`
	SortDirection *string   `json:"sortDirection,omitempty"`
}

type TradesQuery struct {
	Limit        *int      `json:"limit,omitempty"`
	Offset       *int      `json:"offset,omitempty"`
	TakerOnly    *bool     `json:"takerOnly,omitempty"`
	FilterType   *string   `json:"filterType,omitempty"`
	FilterAmount *float64  `json:"filterAmount,omitempty"`
	Market       *[]string `json:"market,omitempty"`
	EventID      *[]string `json:"eventId,omitempty"`
	User         *string   `json:"user,omitempty"`
	Side         *string   `json:"side,omitempty"`
}

type UserActivityQuery struct {
	User          *string   `json:"user,omitempty"`
	Limit         *int      `json:"limit,omitempty"`
	Offset        *int      `json:"offset,omitempty"`
	Market        *[]string `json:"market,omitempty"`
	EventID       *[]string `json:"eventId,omitempty"`
	Type          *string   `json:"type,omitempty"`
	Start         *string   `json:"start,omitempty"`
	End           *string   `json:"end,omitempty"`
	SortBy        *string   `json:"sortBy,omitempty"`
	SortDirection *string   `json:"sortDirection,omitempty"`
	Side          *string   `json:"side,omitempty"`
}

type TopHoldersQuery struct {
	Limit      *int     `json:"limit,omitempty"`      // 0-500, default 100
	Market     []string `json:"market"`               // Required, comma-separated condition IDs
	MinBalance *int     `json:"minBalance,omitempty"` // 0-999999, default 1
}

type TotalValueQuery struct {
	User   *string   `json:"user,omitempty"`
	Market *[]string `json:"market,omitempty"`
}

type TotalMarketsTradedQuery struct {
	User *string `json:"user,omitempty"`
}

type OpenInterestQuery struct {
	Market []string `json:"market"`
}

type LiveVolumeQuery struct {
	ID int `json:"id"`
}
