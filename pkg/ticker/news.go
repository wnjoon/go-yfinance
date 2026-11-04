package ticker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/config"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// newsGraphQLOperation is the GraphQL operation name sent with every news request.
const newsGraphQLOperation = "FinanceTickerListNews"

// newsGraphQLQuery is the GraphQL query behind the news list on
// finance.yahoo.com/quote/<symbol>/news/, copied verbatim from Python yfinance
// 1.8.0. Assets only resolve when a field takes $clientContext (previewUrl,
// clickthroughUrl); otherwise Yahoo returns an "AssetError" for every item.
const newsGraphQLQuery = `
query FinanceTickerListNews($listInput: LightyearListInput!, $clientContext: ClientContext!, $mlRecsInput: MLRecsInput!, $gqlContext: [GqlContext] = [], $imageResize: [ImageResizeInput!]! = [], $mlRecsFirst: Int, $after: String) {
  lightyearList: financeTickerList(list_input: $listInput, cc: $clientContext) {
    main: mlRecsStream(mlRecsInput: $mlRecsInput, first: $mlRecsFirst, after: $after) {
      edges { node { asset(gqlContext: $gqlContext) {
        __typename
        ... on Story { id: uuid title previewUrl(cc: $clientContext) isPremiumNews contentAttributes { ...Attrs } finance { ...Tickers } }
        ... on Video { id: uuid title previewUrl(cc: $clientContext) contentAttributes { ...Attrs } finance { ...Tickers } }
        ... on Outlink { id: uuid title: headline description displayTime url provider { displayName url } contentAttributes { thumbnail { ...Image } } }
      } } }
      pageInfo { hasNextPage endCursor }
    }
  }
}
fragment Image on Image { url height width resized(resizeInput: $imageResize) { url height width transformLabel } }
fragment Attrs on ContentAttributes {
  description summary publishTime displayTime isHosted canonicalUrl clickthroughUrl(cc: $clientContext)
  provider { displayName url } thumbnail { ...Image }
}
fragment Tickers on Finance { stockTickers { symbol } }
`

// newsGraphQLClientName is the value of the x-yahoo-cg-client-name header the
// GraphQL gateway requires.
const newsGraphQLClientName = "finance"

// newsMLRecsCount is the size of the candidate pool Yahoo ranks per request.
const newsMLRecsCount = 200

// newsContentTypes maps the GraphQL asset typename to NewsArticle.Type.
// Any other typename (e.g. "AssetError") is skipped.
var newsContentTypes = map[string]string{
	"Story":   "STORY",
	"Video":   "VIDEO",
	"Outlink": "OUTLINK",
}

// newsCacheKey identifies one cached News result. A different count, tab or
// locale is a different result.
type newsCacheKey struct {
	count  int
	list   string
	lang   string
	region string
}

// newsGraphQLRequest is the JSON body of a news GraphQL request.
type newsGraphQLRequest struct {
	OperationName string               `json:"operationName"`
	Query         string               `json:"query"`
	Variables     newsGraphQLVariables `json:"variables"`
}

// newsGraphQLVariables holds the GraphQL variables of a news request.
type newsGraphQLVariables struct {
	ClientContext newsClientContext `json:"clientContext"`
	GQLContext    []newsGQLContext  `json:"gqlContext"`
	ImageResize   []newsImageResize `json:"imageResize"`
	ListInput     newsListInput     `json:"listInput"`
	MLRecsInput   newsMLRecsInput   `json:"mlRecsInput"`
	MLRecsFirst   int               `json:"mlRecsFirst"`
	After         *string           `json:"after"`
}

type newsClientContext struct {
	Device string `json:"device"`
	Lang   string `json:"lang"`
	Region string `json:"region"`
	Site   string `json:"site"`
}

type newsGQLContext struct {
	ListAlias string `json:"listAlias"`
}

type newsImageResize struct {
	Operations     []string `json:"operations"`
	TransformLabel string   `json:"transformLabel"`
}

type newsListInput struct {
	DisableDedupe               bool               `json:"disableDedupe"`
	EnableBlockedContent        bool               `json:"enableBlockedContent"`
	EnableQueryTimeLicenseCheck bool               `json:"enableQueryTimeLicenseCheck"`
	FilterClientContext         bool               `json:"filterClientContext"`
	GetFullList                 bool               `json:"getFullList"`
	QueryVariables              newsQueryVariables `json:"queryVariables"`
	Slug                        string             `json:"slug"`
}

type newsQueryVariables struct {
	TickerSymbol []string `json:"tickerSymbol"`
}

type newsMLRecsInput struct {
	Count     int               `json:"count"`
	Instance  string            `json:"instance"`
	RankInput newsRankInputWrap `json:"rankInput"`
}

type newsRankInputWrap struct {
	FinanceQuoteNewsV2 newsRankInput `json:"financeQuoteNewsV2"`
}

type newsRankInput struct {
	AboutnessWeight    float64  `json:"aboutnessWeight"`
	BoostDurationHours int      `json:"boostDurationHours"`
	HalfLifeHours      int      `json:"halfLifeHours"`
	RecencyWeight      float64  `json:"recencyWeight"`
	TickerSymbol       []string `json:"tickerSymbol"`
}

// newsGraphQLResponse is the subset of the GraphQL response News reads.
// GraphQL "errors" are intentionally ignored: missing data yields no articles.
type newsGraphQLResponse struct {
	Data *struct {
		LightyearList *struct {
			Main *newsStream `json:"main"`
		} `json:"lightyearList"`
	} `json:"data"`
}

type newsStream struct {
	Edges []struct {
		Node *struct {
			Asset json.RawMessage `json:"asset"`
		} `json:"node"`
	} `json:"edges"`
	PageInfo *struct {
		HasNextPage bool    `json:"hasNextPage"`
		EndCursor   *string `json:"endCursor"`
	} `json:"pageInfo"`
}

type newsAsset struct {
	Typename          string                 `json:"__typename"`
	ID                string                 `json:"id"`
	Title             string                 `json:"title"`
	Description       string                 `json:"description"`
	DisplayTime       string                 `json:"displayTime"`
	URL               string                 `json:"url"`
	Provider          *newsProvider          `json:"provider"`
	PreviewURL        string                 `json:"previewUrl"`
	IsPremiumNews     bool                   `json:"isPremiumNews"`
	ContentAttributes *newsContentAttributes `json:"contentAttributes"`
	Finance           *struct {
		StockTickers []struct {
			Symbol string `json:"symbol"`
		} `json:"stockTickers"`
	} `json:"finance"`
}

type newsContentAttributes struct {
	Description     string        `json:"description"`
	Summary         string        `json:"summary"`
	PublishTime     string        `json:"publishTime"`
	DisplayTime     string        `json:"displayTime"`
	IsHosted        bool          `json:"isHosted"`
	CanonicalURL    string        `json:"canonicalUrl"`
	ClickthroughURL string        `json:"clickthroughUrl"`
	Provider        *newsProvider `json:"provider"`
	Thumbnail       *newsImage    `json:"thumbnail"`
}

type newsProvider struct {
	DisplayName string `json:"displayName"`
	URL         string `json:"url"`
}

type newsImage struct {
	URL     string  `json:"url"`
	Width   float64 `json:"width"`
	Height  float64 `json:"height"`
	Resized []struct {
		URL            string  `json:"url"`
		Width          float64 `json:"width"`
		Height         float64 `json:"height"`
		TransformLabel string  `json:"transformLabel"`
	} `json:"resized"`
}

// News fetches the latest news articles for the ticker, newest first.
//
// News is loaded from Yahoo's news GraphQL API (the same request the
// finance.yahoo.com quote news page makes), following the result cursor until
// count articles are collected or Yahoo has no more. The request follows the
// locale set with config.SetLocale; Yahoo only has news for some locales and
// returns an empty slice for the others.
//
// Parameters:
//   - count: Maximum number of articles to return (default: 10 when <= 0)
//   - tab: Type of news to fetch (default: NewsTabNews when empty); matched
//     case-insensitively
//
// An unknown tab returns an error wrapping models.ErrInvalidNewsTab. HTTP
// failures return the pkg/client error types (for example a rate-limit error
// for HTTP 429). GraphQL errors or a response without data return an empty,
// non-nil slice. Results, including empty ones, are cached per count, tab and
// locale until ClearCache is called.
//
// Example:
//
//	news, err := ticker.News(10, models.NewsTabNews)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, article := range news {
//	    fmt.Printf("%s: %s\n", article.Publisher, article.Title)
//	}
func (t *Ticker) News(count int, tab models.NewsTab) ([]models.NewsArticle, error) {
	if count <= 0 {
		count = 10
	}
	listName, err := tab.ListName()
	if err != nil {
		return nil, err
	}

	lang, region := config.Get().GetLocale()
	key := newsCacheKey{count: count, list: listName, lang: lang, region: region}

	t.mu.RLock()
	if cached, ok := t.newsCache[key]; ok {
		result := cloneNewsArticles(cached)
		t.mu.RUnlock()
		return result, nil
	}
	generation := t.cacheGeneration
	t.mu.RUnlock()

	slug, err := tab.Slug(lang, region)
	if err != nil {
		return nil, err
	}
	articles, err := t.fetchNews(count, slug, lang, region)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	// Do not repopulate a cache that ClearCache reset while we were fetching.
	if t.cacheGeneration == generation {
		if t.newsCache == nil {
			t.newsCache = make(map[newsCacheKey][]models.NewsArticle)
		}
		t.newsCache[key] = articles
	}
	t.mu.Unlock()

	return cloneNewsArticles(articles), nil
}

// fetchNews pages through the news GraphQL API and returns at most count
// articles. The returned slice is never nil.
func (t *Ticker) fetchNews(count int, slug, lang, region string) ([]models.NewsArticle, error) {
	variables := newsGraphQLVariables{
		ClientContext: newsClientContext{Device: "desktop", Lang: lang, Region: region, Site: "finance"},
		GQLContext:    []newsGQLContext{{ListAlias: slug}},
		ImageResize: []newsImageResize{{
			Operations:     []string{"resizefill_w170_h128"},
			TransformLabel: "170x128",
		}},
		ListInput: newsListInput{
			EnableQueryTimeLicenseCheck: true,
			GetFullList:                 true,
			QueryVariables:              newsQueryVariables{TickerSymbol: []string{t.symbol}},
			Slug:                        slug,
		},
		MLRecsInput: newsMLRecsInput{
			Count:    newsMLRecsCount,
			Instance: "FINANCE",
			// aboutnessWeight 0 ranks purely by recency, so items come back
			// newest first (the news page uses 0.05, which mixes in relevance).
			RankInput: newsRankInputWrap{FinanceQuoteNewsV2: newsRankInput{
				AboutnessWeight:    0,
				BoostDurationHours: 16,
				HalfLifeHours:      24,
				RecencyWeight:      1,
				TickerSymbol:       []string{t.symbol},
			}},
		},
	}

	articles := make([]models.NewsArticle, 0, count)
	var after *string
	for len(articles) < count {
		variables.MLRecsFirst = count - len(articles)
		variables.After = after

		stream, err := t.fetchNewsPage(variables)
		if err != nil {
			return nil, err
		}
		if stream == nil {
			break
		}

		for _, edge := range stream.Edges {
			if edge.Node == nil {
				continue
			}
			if article, ok := parseNewsAsset(edge.Node.Asset); ok {
				articles = append(articles, article)
			}
		}

		var next string
		hasNext := false
		if stream.PageInfo != nil {
			hasNext = stream.PageInfo.HasNextPage
			if stream.PageInfo.EndCursor != nil {
				next = *stream.PageInfo.EndCursor
			}
		}
		// A repeated cursor would request the same page forever.
		if len(stream.Edges) == 0 || !hasNext || next == "" || (after != nil && *after == next) {
			break
		}
		after = &next
	}

	if len(articles) > count {
		articles = articles[:count]
	}
	return articles, nil
}

// fetchNewsPage posts one GraphQL page request. It returns a nil stream when
// the response carries no news data (for example on GraphQL errors).
func (t *Ticker) fetchNewsPage(variables newsGraphQLVariables) (*newsStream, error) {
	body, err := json.Marshal(newsGraphQLRequest{
		OperationName: newsGraphQLOperation,
		Query:         newsGraphQLQuery,
		Variables:     variables,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal news request: %w", err)
	}

	newsURL := t.newsURL
	if newsURL == "" {
		newsURL = endpoints.NewsGraphQLURL
	}

	// The GraphQL gateway authenticates with the client cookie (when present)
	// and the x-yahoo-cg-client-name header; it does not use a crumb.
	resp, err := t.client.PostJSONWithHeaders(newsURL, nil, body, map[string]string{
		"x-yahoo-cg-client-name": newsGraphQLClientName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch news: %w", err)
	}

	if resp.StatusCode == 429 {
		return nil, client.WrapRateLimitError()
	}
	if resp.StatusCode >= 400 {
		return nil, client.HTTPStatusToError(resp.StatusCode, resp.Body)
	}

	var apiResp newsGraphQLResponse
	if err := json.Unmarshal([]byte(resp.Body), &apiResp); err != nil {
		return nil, client.WrapInvalidResponseError(fmt.Errorf("failed to parse news response: %w", err))
	}
	if apiResp.Data == nil || apiResp.Data.LightyearList == nil {
		return nil, nil
	}
	return apiResp.Data.LightyearList.Main, nil
}

// parseNewsAsset converts one GraphQL news asset into a NewsArticle. It
// reports false for null assets, undecodable assets and typenames other than
// Story, Video and Outlink (for example "AssetError").
func parseNewsAsset(raw json.RawMessage) (models.NewsArticle, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return models.NewsArticle{}, false
	}
	var asset newsAsset
	if err := json.Unmarshal(raw, &asset); err != nil {
		return models.NewsArticle{}, false
	}
	contentType, ok := newsContentTypes[asset.Typename]
	if !ok {
		return models.NewsArticle{}, false
	}

	attrs := asset.ContentAttributes
	if attrs == nil {
		attrs = &newsContentAttributes{}
	}

	var canonicalURL, clickThroughURL string
	var provider *newsProvider
	if asset.Typename == "Outlink" {
		canonicalURL, clickThroughURL = asset.URL, asset.URL
		provider = asset.Provider
	} else {
		canonicalURL, clickThroughURL = attrs.CanonicalURL, attrs.ClickthroughURL
		provider = attrs.Provider
	}

	article := models.NewsArticle{
		UUID:         asset.ID,
		Title:        asset.Title,
		Type:         contentType,
		Link:         firstNonEmpty(clickThroughURL, canonicalURL),
		CanonicalURL: canonicalURL,
		PreviewURL:   asset.PreviewURL,
		Summary:      attrs.Summary,
		Description:  firstNonEmpty(asset.Description, attrs.Description),
		PublishTime:  parseNewsTime(firstNonEmpty(attrs.PublishTime, asset.DisplayTime)),
		DisplayTime:  parseNewsTime(firstNonEmpty(attrs.DisplayTime, asset.DisplayTime)),
		IsHosted:     attrs.IsHosted,
		IsPremium:    asset.IsPremiumNews,
		Thumbnail:    newsThumbnail(attrs.Thumbnail),
	}
	if provider != nil {
		article.Publisher = provider.DisplayName
		article.PublisherURL = provider.URL
	}
	if asset.Finance != nil {
		for _, stock := range asset.Finance.StockTickers {
			if stock.Symbol != "" {
				article.RelatedTickers = append(article.RelatedTickers, stock.Symbol)
			}
		}
	}
	return article, true
}

// newsThumbnail converts a GraphQL image into a NewsThumbnail whose first
// resolution is the original image (tag "original") followed by the resized
// versions tagged with their transform label. It returns nil without an image URL.
func newsThumbnail(image *newsImage) *models.NewsThumbnail {
	if image == nil || image.URL == "" {
		return nil
	}
	thumbnail := &models.NewsThumbnail{
		OriginalURL:    image.URL,
		OriginalWidth:  int(image.Width),
		OriginalHeight: int(image.Height),
		Resolutions: []models.ThumbnailResolution{{
			URL:    image.URL,
			Width:  int(image.Width),
			Height: int(image.Height),
			Tag:    "original",
		}},
	}
	for _, resized := range image.Resized {
		if resized.URL == "" {
			continue
		}
		thumbnail.Resolutions = append(thumbnail.Resolutions, models.ThumbnailResolution{
			URL:    resized.URL,
			Width:  int(resized.Width),
			Height: int(resized.Height),
			Tag:    resized.TransformLabel,
		})
	}
	return thumbnail
}

// parseNewsTime parses an ISO 8601 / RFC 3339 timestamp into Unix seconds.
// Empty or unparseable values return 0.
func parseNewsTime(value string) int64 {
	if value == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0
	}
	return parsed.Unix()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func cloneNewsArticles(articles []models.NewsArticle) []models.NewsArticle {
	if articles == nil {
		return nil
	}
	result := make([]models.NewsArticle, len(articles))
	for i, article := range articles {
		result[i] = article
		result[i].RelatedTickers = append([]string(nil), article.RelatedTickers...)
		if article.Thumbnail != nil {
			thumbnail := *article.Thumbnail
			thumbnail.Resolutions = append(
				[]models.ThumbnailResolution(nil),
				article.Thumbnail.Resolutions...,
			)
			result[i].Thumbnail = &thumbnail
		}
	}
	return result
}

// GetNews is an alias for News with default parameters.
// It fetches 10 news articles of type "news".
//
// Example:
//
//	news, err := ticker.GetNews()
func (t *Ticker) GetNews() ([]models.NewsArticle, error) {
	return t.News(10, models.NewsTabNews)
}
