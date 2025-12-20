package models

// SearchResult represents the complete search response from Yahoo Finance.
//
// This includes quotes (stock matches), news articles, lists, and research reports.
//
// Example:
//
//	result, err := search.Search("AAPL")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, quote := range result.Quotes {
//	    fmt.Printf("%s: %s\n", quote.Symbol, quote.ShortName)
//	}
type SearchResult struct {
	// Quotes contains matching stock/asset quotes.
	Quotes []SearchQuote `json:"quotes"`

	// News contains related news articles.
	News []SearchNews `json:"news,omitempty"`

	// Lists contains Yahoo Finance lists.
	Lists []SearchList `json:"lists,omitempty"`

	// Research contains research reports (if requested).
	Research []SearchResearch `json:"research,omitempty"`

	// Nav contains navigation links (if requested).
	Nav []SearchNav `json:"nav,omitempty"`

	// TotalCount is the total number of matching results.
	TotalCount int `json:"totalCount,omitempty"`
}

// SearchQuote represents a single quote result from search.
type SearchQuote struct {
	// Symbol is the ticker symbol.
	Symbol string `json:"symbol"`

	// ShortName is the short company name.
	ShortName string `json:"shortname"`

	// LongName is the full company name.
	LongName string `json:"longname,omitempty"`

	// Exchange is the exchange where the asset is traded.
	Exchange string `json:"exchange"`

	// ExchangeDisp is the display name of the exchange.
	ExchangeDisp string `json:"exchDisp,omitempty"`

	// QuoteType is the type of asset (EQUITY, ETF, MUTUALFUND, etc.).
	QuoteType string `json:"quoteType"`

	// TypeDisp is the display name of the asset type.
	TypeDisp string `json:"typeDisp,omitempty"`

	// Score is the relevance score of this result.
	Score float64 `json:"score,omitempty"`

	// IsYahooFinance indicates if this is a Yahoo Finance asset.
	IsYahooFinance bool `json:"isYahooFinance,omitempty"`

	// Industry is the industry of the company.
	Industry string `json:"industry,omitempty"`

	// Sector is the sector of the company.
	Sector string `json:"sector,omitempty"`
}

// SearchNews represents a news article from search results.
type SearchNews struct {
	// UUID is the unique identifier for the news article.
	UUID string `json:"uuid"`

	// Title is the headline of the article.
	Title string `json:"title"`

	// Publisher is the source of the article.
	Publisher string `json:"publisher"`

	// Link is the URL to the article.
	Link string `json:"link"`

	// PublishTime is the Unix timestamp when the article was published.
	PublishTime int64 `json:"providerPublishTime"`

	// Type is the news type (e.g., "STORY").
	Type string `json:"type,omitempty"`

	// Thumbnail contains thumbnail image information.
	Thumbnail *SearchThumbnail `json:"thumbnail,omitempty"`

	// RelatedTickers contains symbols related to this news.
	RelatedTickers []string `json:"relatedTickers,omitempty"`
}

// SearchThumbnail represents thumbnail image information.
type SearchThumbnail struct {
	Resolutions []ThumbnailResolution `json:"resolutions,omitempty"`
}

// ThumbnailResolution represents a single thumbnail resolution.
type ThumbnailResolution struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Tag    string `json:"tag,omitempty"`
}

// SearchList represents a Yahoo Finance list from search results.
type SearchList struct {
	// ID is the list identifier.
	ID string `json:"id"`

	// Name is the list name.
	Name string `json:"name"`

	// Description is the list description.
	Description string `json:"description,omitempty"`

	// SymbolCount is the number of symbols in the list.
	SymbolCount int `json:"symbolCount,omitempty"`

	// URL is the link to the list.
	URL string `json:"url,omitempty"`
}

// SearchResearch represents a research report from search results.
type SearchResearch struct {
	// ReportID is the unique identifier.
	ReportID string `json:"reportId"`

	// Title is the report title.
	Title string `json:"title"`

	// Provider is the research provider.
	Provider string `json:"provider"`

	// Ticker is the related ticker symbol.
	Ticker string `json:"ticker,omitempty"`

	// PublishDate is the publication date.
	PublishDate string `json:"publishDate,omitempty"`
}

// SearchNav represents a navigation link from search results.
type SearchNav struct {
	// Name is the navigation item name.
	Name string `json:"name"`

	// URL is the navigation URL.
	URL string `json:"url"`
}

// SearchParams represents parameters for the Search function.
type SearchParams struct {
	// Query is the search query (required).
	Query string

	// MaxResults is the maximum number of stock quotes to return (default 8).
	MaxResults int

	// NewsCount is the number of news articles to return (default 8).
	NewsCount int

	// ListsCount is the number of lists to return (default 8).
	ListsCount int

	// IncludeResearch includes research reports in results.
	IncludeResearch bool

	// IncludeNav includes navigation links in results.
	IncludeNav bool

	// EnableFuzzyQuery enables fuzzy matching for typos.
	EnableFuzzyQuery bool
}

// DefaultSearchParams returns default search parameters.
func DefaultSearchParams() SearchParams {
	return SearchParams{
		MaxResults: 8,
		NewsCount:  8,
		ListsCount: 8,
	}
}

// SearchResponse represents the raw API response from Yahoo Finance search.
type SearchResponse struct {
	Quotes   []map[string]interface{} `json:"quotes"`
	News     []map[string]interface{} `json:"news"`
	Lists    []map[string]interface{} `json:"lists,omitempty"`
	Research []map[string]interface{} `json:"researchReports,omitempty"`
	Nav      []map[string]interface{} `json:"nav,omitempty"`
	Count    int                      `json:"count,omitempty"`
}
