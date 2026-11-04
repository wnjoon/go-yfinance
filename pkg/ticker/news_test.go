package ticker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/config"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// --- fake news GraphQL server -------------------------------------------

// newsRequest is one request received by the fake news server.
type newsRequest struct {
	Method  string
	Path    string
	Query   string
	Header  http.Header
	RawBody string
	Body    struct {
		OperationName string                 `json:"operationName"`
		Query         string                 `json:"query"`
		Variables     map[string]interface{} `json:"variables"`
	}
}

func (r newsRequest) variable(path ...string) interface{} {
	var current interface{} = r.Body.Variables
	for _, key := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[key]
	}
	return current
}

func (r newsRequest) mlRecsFirst() int {
	v, _ := r.variable("mlRecsFirst").(float64)
	return int(v)
}

func (r newsRequest) slug() string {
	v, _ := r.variable("listInput", "slug").(string)
	return v
}

// newsReply is what the fake server answers for one request.
type newsReply struct {
	status int
	body   string
}

type fakeNewsServer struct {
	t       *testing.T
	srv     *httptest.Server
	mu      sync.Mutex
	reqs    []newsRequest
	handler func(req newsRequest, n int) newsReply
}

func newFakeNewsServer(t *testing.T, handler func(req newsRequest, n int) newsReply) *fakeNewsServer {
	t.Helper()
	f := &fakeNewsServer{t: t, handler: handler}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		req := newsRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Query:   r.URL.RawQuery,
			Header:  r.Header.Clone(),
			RawBody: string(raw),
		}
		if err := json.Unmarshal(raw, &req.Body); err != nil {
			t.Errorf("request body is not JSON: %v (%s)", err, raw)
		}
		f.mu.Lock()
		f.reqs = append(f.reqs, req)
		n := len(f.reqs)
		f.mu.Unlock()

		reply := f.handler(req, n)
		if reply.status == 0 {
			reply.status = http.StatusOK
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(reply.status)
		_, _ = w.Write([]byte(reply.body))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeNewsServer) requests() []newsRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]newsRequest(nil), f.reqs...)
}

func (f *fakeNewsServer) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.reqs)
}

// newNewsTicker returns a ticker whose news requests go to the fake server.
func newNewsTicker(t *testing.T, f *fakeNewsServer, symbol string) *Ticker {
	t.Helper()
	tkr, err := New(symbol)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	tkr.newsURL = f.srv.URL + "/"
	t.Cleanup(tkr.Close)
	return tkr
}

// setNewsLocale sets the global locale for the duration of the test.
func setNewsLocale(t *testing.T, lang, region string) {
	t.Helper()
	oldLang, oldRegion := config.Get().GetLocale()
	config.Get().SetLocale(lang, region)
	t.Cleanup(func() { config.Get().SetLocale(oldLang, oldRegion) })
}

// --- payload builders ----------------------------------------------------

func storyAsset(id string) map[string]interface{} {
	return typedAsset(id, "Story")
}

func typedAsset(id, typename string) map[string]interface{} {
	return map[string]interface{}{
		"__typename":    typename,
		"id":            id,
		"title":         "title " + id,
		"previewUrl":    nil,
		"isPremiumNews": false,
		"contentAttributes": map[string]interface{}{
			"description":     nil,
			"summary":         "summary " + id,
			"publishTime":     "2026-10-25T20:05:00Z",
			"displayTime":     "2026-10-25T20:06:00Z",
			"isHosted":        true,
			"canonicalUrl":    "https://example.com/" + id,
			"clickthroughUrl": "https://finance.yahoo.com/news/" + id + ".html",
			"provider":        map[string]interface{}{"displayName": "Provider", "url": "https://example.com/"},
			"thumbnail": map[string]interface{}{
				"url": "https://media.example.com/" + id + ".jpg", "width": 1400, "height": 933,
				"resized": []interface{}{map[string]interface{}{
					"url": "https://s.yimg.com/" + id + ".jpg", "width": 170, "height": 128,
					"transformLabel": "170x128",
				}},
			},
		},
		"finance": map[string]interface{}{"stockTickers": []interface{}{map[string]interface{}{"symbol": "MSFT"}}},
	}
}

func outlinkAsset(id string) map[string]interface{} {
	return map[string]interface{}{
		"__typename":  "Outlink",
		"id":          id,
		"title":       "headline " + id,
		"description": "outlink description",
		"displayTime": "2026-10-26T08:00:00.000Z",
		"url":         "https://outlink.example.com/" + id,
		"provider":    map[string]interface{}{"displayName": "Outlink Provider", "url": "https://outlink.example.com/"},
		"contentAttributes": map[string]interface{}{
			"thumbnail": map[string]interface{}{
				"url": "https://outlink.example.com/" + id + ".png", "width": 640, "height": 360,
				"resized": []interface{}{},
			},
		},
	}
}

// newsPage builds a GraphQL response; cursor nil encodes endCursor: null.
func newsPage(assets []interface{}, cursor *string, hasNext bool) string {
	edges := make([]interface{}, 0, len(assets))
	for _, asset := range assets {
		edges = append(edges, map[string]interface{}{"node": map[string]interface{}{"asset": asset}})
	}
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"lightyearList": map[string]interface{}{
				"main": map[string]interface{}{
					"edges":    edges,
					"pageInfo": map[string]interface{}{"hasNextPage": hasNext, "endCursor": cursor},
				},
			},
		},
	}
	raw, _ := json.Marshal(payload)
	return string(raw)
}

func strPtr(s string) *string { return &s }

// storiesFor returns n stories with ids "<prefix>-0".."<prefix>-(n-1)".
func storiesFor(prefix string, n int) []interface{} {
	assets := make([]interface{}, n)
	for i := range assets {
		assets[i] = storyAsset(fmt.Sprintf("%s-%d", prefix, i))
	}
	return assets
}

// echoHandler answers each request with mlRecsFirst stories named after the
// list name and no further pages, like the upstream Python test fake.
func echoHandler(req newsRequest, _ int) newsReply {
	slug := req.slug()
	listName := slug[strings.LastIndex(slug, "-")+1:]
	return newsReply{body: newsPage(storiesFor(listName, req.mlRecsFirst()), nil, false)}
}

func articleIDs(articles []models.NewsArticle) []string {
	ids := make([]string, len(articles))
	for i, article := range articles {
		ids[i] = article.UUID
	}
	return ids
}

// --- NewsTab -------------------------------------------------------------

func TestNewsTabListName(t *testing.T) {
	tests := []struct {
		tab  models.NewsTab
		want string
	}{
		{models.NewsTabNews, "news"},
		{models.NewsTabAll, "all"},
		{models.NewsTabPressReleases, "pressrelease"},
		{models.NewsTab(""), "news"},
		{models.NewsTab("NEWS"), "news"},
		{models.NewsTab("Press Releases"), "pressrelease"},
		{models.NewsTab("All"), "all"},
	}
	for _, tt := range tests {
		t.Run(string(tt.tab), func(t *testing.T) {
			got, err := tt.tab.ListName()
			if err != nil || got != tt.want {
				t.Fatalf("ListName() = %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestNewsTabListNameInvalid(t *testing.T) {
	for _, tab := range []models.NewsTab{"bogus", "press", "pressrelease", "latestNews", " news"} {
		_, err := tab.ListName()
		if !errors.Is(err, models.ErrInvalidNewsTab) {
			t.Errorf("ListName(%q) error = %v, want ErrInvalidNewsTab", tab, err)
		}
		if _, err := tab.Slug("en-US", "US"); !errors.Is(err, models.ErrInvalidNewsTab) {
			t.Errorf("Slug(%q) error = %v, want ErrInvalidNewsTab", tab, err)
		}
	}
}

func TestNewsTabSlug(t *testing.T) {
	tests := []struct {
		tab          models.NewsTab
		lang, region string
		want         string
	}{
		{models.NewsTabNews, "en-US", "US", "list=finance-US-en-US-ticker-news"},
		{models.NewsTabAll, "en-US", "US", "list=finance-US-en-US-ticker-all"},
		{models.NewsTabPressReleases, "en-US", "US", "list=finance-US-en-US-ticker-pressrelease"},
		{models.NewsTabNews, "zh-Hant-HK", "HK", "list=finance-HK-zh-Hant-HK-ticker-news"},
		{models.NewsTabAll, "en-GB", "GB", "list=finance-GB-en-GB-ticker-all"},
	}
	for _, tt := range tests {
		got, err := tt.tab.Slug(tt.lang, tt.region)
		if err != nil || got != tt.want {
			t.Errorf("Slug(%q, %q, %q) = %q, %v; want %q", tt.tab, tt.lang, tt.region, got, err, tt.want)
		}
	}
}

func TestNewsTabQueryRefDeprecatedStillWorks(t *testing.T) {
	tests := []struct {
		tab      models.NewsTab
		expected string
	}{
		{models.NewsTabAll, "newsAll"},
		{models.NewsTabNews, "latestNews"},
		{models.NewsTabPressReleases, "pressRelease"},
		{models.NewsTab(""), "latestNews"}, // Default
	}

	for _, tt := range tests {
		t.Run(string(tt.tab), func(t *testing.T) {
			got := tt.tab.QueryRef() //nolint:staticcheck // keeps the deprecated API covered
			if got != tt.expected {
				t.Errorf("NewsTab(%q).QueryRef() = %s, want %s", tt.tab, got, tt.expected)
			}
		})
	}
}

func TestNewsArticlePublishedAt(t *testing.T) {
	timestamp := int64(1705329000) // 2024-01-15 14:30:00 UTC
	article := &models.NewsArticle{
		PublishTime: timestamp,
	}

	publishedAt := article.PublishedAt()
	expected := time.Unix(timestamp, 0)

	if !publishedAt.Equal(expected) {
		t.Errorf("PublishedAt() = %v, want %v", publishedAt, expected)
	}
}

func TestNewsTabString(t *testing.T) {
	tests := []struct {
		tab      models.NewsTab
		expected string
	}{
		{models.NewsTabAll, "all"},
		{models.NewsTabNews, "news"},
		{models.NewsTabPressReleases, "press releases"},
	}

	for _, tt := range tests {
		t.Run(string(tt.tab), func(t *testing.T) {
			got := tt.tab.String()
			if got != tt.expected {
				t.Errorf("NewsTab.String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

// --- request shape -------------------------------------------------------

func TestNewsRequestShape(t *testing.T) {
	setNewsLocale(t, "en-US", "US")
	f := newFakeNewsServer(t, echoHandler)
	tkr := newNewsTicker(t, f, "msft")

	if _, err := tkr.News(3, models.NewsTabAll); err != nil {
		t.Fatalf("News() error: %v", err)
	}
	reqs := f.requests()
	if len(reqs) != 1 {
		t.Fatalf("requests = %d, want 1", len(reqs))
	}
	req := reqs[0]

	if req.Method != http.MethodPost || req.Path != "/" {
		t.Fatalf("request = %s %s", req.Method, req.Path)
	}
	if req.Query != "" {
		t.Fatalf("news request must not carry query params (no crumb), got %q", req.Query)
	}
	if got := req.Header.Get("X-Yahoo-Cg-Client-Name"); got != "finance" {
		t.Fatalf("x-yahoo-cg-client-name = %q", got)
	}
	if got := req.Header.Values("Content-Type"); len(got) != 1 || got[0] != "application/json" {
		t.Fatalf("Content-Type = %v", got)
	}
	if req.Body.OperationName != "FinanceTickerListNews" {
		t.Fatalf("operationName = %q", req.Body.OperationName)
	}
	if req.Body.Query != newsGraphQLQuery {
		t.Fatal("query must be the upstream GraphQL query verbatim")
	}

	slug := "list=finance-US-en-US-ticker-all"
	wantVariables := map[string]interface{}{
		"clientContext": map[string]interface{}{"device": "desktop", "lang": "en-US", "region": "US", "site": "finance"},
		"gqlContext":    []interface{}{map[string]interface{}{"listAlias": slug}},
		"imageResize": []interface{}{map[string]interface{}{
			"operations":     []interface{}{"resizefill_w170_h128"},
			"transformLabel": "170x128",
		}},
		"listInput": map[string]interface{}{
			"disableDedupe":               false,
			"enableBlockedContent":        false,
			"enableQueryTimeLicenseCheck": true,
			"filterClientContext":         false,
			"getFullList":                 true,
			"queryVariables":              map[string]interface{}{"tickerSymbol": []interface{}{"MSFT"}},
			"slug":                        slug,
		},
		"mlRecsInput": map[string]interface{}{
			"count":    float64(200),
			"instance": "FINANCE",
			"rankInput": map[string]interface{}{"financeQuoteNewsV2": map[string]interface{}{
				"aboutnessWeight":    float64(0),
				"boostDurationHours": float64(16),
				"halfLifeHours":      float64(24),
				"recencyWeight":      float64(1),
				"tickerSymbol":       []interface{}{"MSFT"},
			}},
		},
		"mlRecsFirst": float64(3),
		"after":       nil,
	}
	if !reflect.DeepEqual(req.Body.Variables, wantVariables) {
		got, _ := json.MarshalIndent(req.Body.Variables, "", "  ")
		t.Fatalf("variables mismatch:\n%s", got)
	}
	// "after" must be sent as an explicit null on the first page.
	if !strings.Contains(req.RawBody, `"after":null`) {
		t.Fatalf("first page must send after:null, body=%s", req.RawBody)
	}
}

func TestNewsSlugPerTab(t *testing.T) {
	setNewsLocale(t, "en-US", "US")
	tests := []struct {
		tab  models.NewsTab
		want string
	}{
		{models.NewsTabNews, "list=finance-US-en-US-ticker-news"},
		{models.NewsTabAll, "list=finance-US-en-US-ticker-all"},
		{models.NewsTabPressReleases, "list=finance-US-en-US-ticker-pressrelease"},
		{"", "list=finance-US-en-US-ticker-news"},
		{"Press Releases", "list=finance-US-en-US-ticker-pressrelease"},
	}
	for _, tt := range tests {
		t.Run(string(tt.tab), func(t *testing.T) {
			f := newFakeNewsServer(t, echoHandler)
			tkr := newNewsTicker(t, f, "AAPL")
			if _, err := tkr.News(1, tt.tab); err != nil {
				t.Fatalf("News() error: %v", err)
			}
			req := f.requests()[0]
			if got := req.slug(); got != tt.want {
				t.Fatalf("slug = %q, want %q", got, tt.want)
			}
			if got := req.variable("gqlContext").([]interface{})[0].(map[string]interface{})["listAlias"]; got != tt.want {
				t.Fatalf("listAlias = %v, want %q", got, tt.want)
			}
		})
	}
}

func TestNewsFollowsConfigLocale(t *testing.T) {
	setNewsLocale(t, "zh-Hant-HK", "HK")
	f := newFakeNewsServer(t, echoHandler)
	tkr := newNewsTicker(t, f, "MSFT")

	if _, err := tkr.News(1, models.NewsTabNews); err != nil {
		t.Fatalf("News() error: %v", err)
	}
	req := f.requests()[0]
	if got := req.slug(); got != "list=finance-HK-zh-Hant-HK-ticker-news" {
		t.Fatalf("slug = %q", got)
	}
	if got := req.variable("clientContext", "lang"); got != "zh-Hant-HK" {
		t.Fatalf("clientContext.lang = %v", got)
	}
	if got := req.variable("clientContext", "region"); got != "HK" {
		t.Fatalf("clientContext.region = %v", got)
	}
}

func TestNewsDefaultCountIsTen(t *testing.T) {
	f := newFakeNewsServer(t, echoHandler)
	tkr := newNewsTicker(t, f, "AAPL")

	for _, count := range []int{0, -5} {
		news, err := tkr.News(count, "")
		if err != nil {
			t.Fatalf("News(%d) error: %v", count, err)
		}
		if len(news) != 10 {
			t.Fatalf("News(%d) returned %d articles, want 10", count, len(news))
		}
	}
	if got := f.requests()[0].mlRecsFirst(); got != 10 {
		t.Fatalf("mlRecsFirst = %d, want 10", got)
	}
	// count<=0 and count=10 share one cache entry.
	if _, err := tkr.GetNews(); err != nil {
		t.Fatalf("GetNews() error: %v", err)
	}
	if f.count() != 1 {
		t.Fatalf("requests = %d, want 1 (default count must share the cache)", f.count())
	}
}

// --- pagination ----------------------------------------------------------

func TestNewsFollowsPagination(t *testing.T) {
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		switch n {
		case 1:
			return newsReply{body: newsPage([]interface{}{storyAsset("a"), storyAsset("b")}, strPtr("c1"), true)}
		default:
			return newsReply{body: newsPage([]interface{}{storyAsset("c")}, strPtr("c2"), false)}
		}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(5, models.NewsTabNews)
	if err != nil {
		t.Fatalf("News() error: %v", err)
	}
	if got := articleIDs(news); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("ids = %v", got)
	}
	reqs := f.requests()
	if len(reqs) != 2 {
		t.Fatalf("requests = %d, want 2", len(reqs))
	}
	if reqs[0].variable("after") != nil || reqs[0].mlRecsFirst() != 5 {
		t.Fatalf("first page after=%v first=%d", reqs[0].variable("after"), reqs[0].mlRecsFirst())
	}
	if reqs[1].variable("after") != "c1" || reqs[1].mlRecsFirst() != 3 {
		t.Fatalf("second page after=%v first=%d", reqs[1].variable("after"), reqs[1].mlRecsFirst())
	}
}

func TestNewsPaginationMLRecsFirstDecreases(t *testing.T) {
	// Each page returns two items and a fresh cursor.
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		assets := []interface{}{storyAsset(fmt.Sprintf("p%d-0", n)), storyAsset(fmt.Sprintf("p%d-1", n))}
		return newsReply{body: newsPage(assets, strPtr(fmt.Sprintf("c%d", n)), true)}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(7, models.NewsTabNews)
	if err != nil {
		t.Fatalf("News() error: %v", err)
	}
	if len(news) != 7 {
		t.Fatalf("len = %d, want 7", len(news))
	}
	reqs := f.requests()
	var firsts []int
	var afters []interface{}
	for _, req := range reqs {
		firsts = append(firsts, req.mlRecsFirst())
		afters = append(afters, req.variable("after"))
	}
	if !reflect.DeepEqual(firsts, []int{7, 5, 3, 1}) {
		t.Fatalf("mlRecsFirst sequence = %v", firsts)
	}
	if !reflect.DeepEqual(afters, []interface{}{nil, "c1", "c2", "c3"}) {
		t.Fatalf("after sequence = %v", afters)
	}
	if news[6].UUID != "p4-0" {
		t.Fatalf("last article = %q, want p4-0 (truncated to count)", news[6].UUID)
	}
}

func TestNewsPaginationStopsWhenNoNextPage(t *testing.T) {
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		return newsReply{body: newsPage([]interface{}{storyAsset("a")}, strPtr("c1"), false)}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(10, models.NewsTabNews)
	if err != nil || len(news) != 1 {
		t.Fatalf("News() = %d articles, %v", len(news), err)
	}
	if f.count() != 1 {
		t.Fatalf("requests = %d, want 1", f.count())
	}
}

func TestNewsPaginationStopsOnStalledCursor(t *testing.T) {
	// The second page adds nothing and repeats the cursor while claiming more pages.
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		switch n {
		case 1:
			return newsReply{body: newsPage([]interface{}{storyAsset("a")}, strPtr("c1"), true)}
		case 2:
			return newsReply{body: newsPage([]interface{}{map[string]interface{}{"__typename": "AssetError"}}, strPtr("c1"), true)}
		default:
			return newsReply{body: newsPage([]interface{}{storyAsset("b")}, strPtr("c2"), false)}
		}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(5, models.NewsTabNews)
	if err != nil {
		t.Fatalf("News() error: %v", err)
	}
	if got := articleIDs(news); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("ids = %v", got)
	}
	if f.count() != 2 {
		t.Fatalf("requests = %d, want 2 (must stop when endCursor does not advance)", f.count())
	}
}

func TestNewsPaginationStopsOnEmptyEdges(t *testing.T) {
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		if n == 1 {
			return newsReply{body: newsPage([]interface{}{storyAsset("a")}, strPtr("c1"), true)}
		}
		return newsReply{body: newsPage(nil, strPtr(fmt.Sprintf("c%d", n)), true)}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(5, models.NewsTabNews)
	if err != nil || len(news) != 1 {
		t.Fatalf("News() = %d articles, %v", len(news), err)
	}
	if f.count() != 2 {
		t.Fatalf("requests = %d, want 2 (must stop on an empty page)", f.count())
	}
}

func TestNewsPaginationStopsWithoutCursor(t *testing.T) {
	for name, cursor := range map[string]*string{"null": nil, "empty": strPtr("")} {
		t.Run(name, func(t *testing.T) {
			f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
				return newsReply{body: newsPage([]interface{}{storyAsset(fmt.Sprintf("a%d", n))}, cursor, true)}
			})
			tkr := newNewsTicker(t, f, "MSFT")

			news, err := tkr.News(5, models.NewsTabNews)
			if err != nil || len(news) != 1 {
				t.Fatalf("News() = %d articles, %v", len(news), err)
			}
			if f.count() != 1 {
				t.Fatalf("requests = %d, want 1", f.count())
			}
		})
	}
}

func TestNewsPaginationStopsWithoutPageInfo(t *testing.T) {
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		return newsReply{body: `{"data":{"lightyearList":{"main":{"edges":[{"node":{"asset":` +
			mustJSON(storyAsset("a")) + `}}]}}}}`}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(5, models.NewsTabNews)
	if err != nil || len(news) != 1 {
		t.Fatalf("News() = %d articles, %v", len(news), err)
	}
	if f.count() != 1 {
		t.Fatalf("requests = %d, want 1", f.count())
	}
}

func TestNewsTruncatesToCount(t *testing.T) {
	// Yahoo may return more items than requested; never exceed count.
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		return newsReply{body: newsPage(storiesFor("x", 8), strPtr("c1"), true)}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(3, models.NewsTabNews)
	if err != nil {
		t.Fatalf("News() error: %v", err)
	}
	if got := articleIDs(news); !reflect.DeepEqual(got, []string{"x-0", "x-1", "x-2"}) {
		t.Fatalf("ids = %v", got)
	}
	if f.count() != 1 {
		t.Fatalf("requests = %d, want 1", f.count())
	}
}

// --- asset parsing -------------------------------------------------------

func TestNewsSkipsAssetErrorsAndNulls(t *testing.T) {
	assets := []interface{}{
		storyAsset("a"),
		map[string]interface{}{"__typename": "AssetError"},
		nil,
		typedAsset("b", "Video"),
		map[string]interface{}{"__typename": "Advertisement", "id": "ad"},
		map[string]interface{}{"id": "no-typename"},
		map[string]interface{}{"__typename": "Story", "id": 12345}, // undecodable: id is a number
		outlinkAsset("c"),
	}
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		return newsReply{body: newsPage(assets, nil, false)}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(10, models.NewsTabNews)
	if err != nil {
		t.Fatalf("News() error: %v", err)
	}
	if got := articleIDs(news); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Fatalf("ids = %v", got)
	}
	if news[1].Type != "VIDEO" || news[2].Type != "OUTLINK" {
		t.Fatalf("types = %q, %q", news[1].Type, news[2].Type)
	}
}

func TestNewsSkipsNullNodes(t *testing.T) {
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		return newsReply{body: `{"data":{"lightyearList":{"main":{"edges":[{"node":null},{},{"node":{"asset":` +
			mustJSON(storyAsset("a")) + `}}],"pageInfo":{"hasNextPage":false,"endCursor":null}}}}}`}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(10, models.NewsTabNews)
	if err != nil {
		t.Fatalf("News() error: %v", err)
	}
	if got := articleIDs(news); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("ids = %v", got)
	}
}

func TestParseNewsAssetStory(t *testing.T) {
	asset := storyAsset("abc")
	asset["previewUrl"] = "https://finance.yahoo.com/preview/abc"
	asset["isPremiumNews"] = true
	asset["finance"] = map[string]interface{}{"stockTickers": []interface{}{
		map[string]interface{}{"symbol": "MSFT"},
		map[string]interface{}{"symbol": ""},
		map[string]interface{}{"symbol": "AAPL"},
	}}

	article, ok := parseNewsAsset(json.RawMessage(mustJSON(asset)))
	if !ok {
		t.Fatal("story asset was skipped")
	}
	want := models.NewsArticle{
		UUID:           "abc",
		Title:          "title abc",
		Publisher:      "Provider",
		Link:           "https://finance.yahoo.com/news/abc.html",
		PublishTime:    mustUnix(t, "2026-10-25T20:05:00Z"),
		Type:           "STORY",
		RelatedTickers: []string{"MSFT", "AAPL"},
		Summary:        "summary abc",
		CanonicalURL:   "https://example.com/abc",
		PreviewURL:     "https://finance.yahoo.com/preview/abc",
		PublisherURL:   "https://example.com/",
		DisplayTime:    mustUnix(t, "2026-10-25T20:06:00Z"),
		IsHosted:       true,
		IsPremium:      true,
		Thumbnail: &models.NewsThumbnail{
			OriginalURL:    "https://media.example.com/abc.jpg",
			OriginalWidth:  1400,
			OriginalHeight: 933,
			Resolutions: []models.ThumbnailResolution{
				{URL: "https://media.example.com/abc.jpg", Width: 1400, Height: 933, Tag: "original"},
				{URL: "https://s.yimg.com/abc.jpg", Width: 170, Height: 128, Tag: "170x128"},
			},
		},
	}
	if !reflect.DeepEqual(article, want) {
		t.Fatalf("article mismatch:\n got %+v\nwant %+v", article, want)
	}
}

func TestParseNewsAssetVideo(t *testing.T) {
	article, ok := parseNewsAsset(json.RawMessage(mustJSON(typedAsset("vid", "Video"))))
	if !ok || article.Type != "VIDEO" || article.UUID != "vid" || article.Title != "title vid" {
		t.Fatalf("video = %+v, ok=%v", article, ok)
	}
	if article.Link != "https://finance.yahoo.com/news/vid.html" || article.Publisher != "Provider" {
		t.Fatalf("video link/publisher = %q / %q", article.Link, article.Publisher)
	}
}

func TestParseNewsAssetOutlink(t *testing.T) {
	asset := outlinkAsset("out")
	// Outlink must use its own url/provider, never contentAttributes'.
	asset["contentAttributes"].(map[string]interface{})["provider"] = map[string]interface{}{"displayName": "Wrong"}
	asset["contentAttributes"].(map[string]interface{})["canonicalUrl"] = "https://wrong.example.com/"

	article, ok := parseNewsAsset(json.RawMessage(mustJSON(asset)))
	if !ok {
		t.Fatal("outlink asset was skipped")
	}
	if article.Type != "OUTLINK" || article.UUID != "out" || article.Title != "headline out" {
		t.Fatalf("outlink identity = %+v", article)
	}
	if article.Link != "https://outlink.example.com/out" || article.CanonicalURL != "https://outlink.example.com/out" {
		t.Fatalf("outlink urls = %q / %q", article.Link, article.CanonicalURL)
	}
	if article.Publisher != "Outlink Provider" || article.PublisherURL != "https://outlink.example.com/" {
		t.Fatalf("outlink provider = %q / %q", article.Publisher, article.PublisherURL)
	}
	if article.Description != "outlink description" {
		t.Fatalf("outlink description = %q", article.Description)
	}
	wantTime := mustUnix(t, "2026-10-26T08:00:00Z")
	if article.PublishTime != wantTime || article.DisplayTime != wantTime {
		t.Fatalf("outlink times = %d / %d, want %d", article.PublishTime, article.DisplayTime, wantTime)
	}
	if article.Thumbnail == nil || len(article.Thumbnail.Resolutions) != 1 ||
		article.Thumbnail.Resolutions[0].Tag != "original" || article.Thumbnail.OriginalWidth != 640 {
		t.Fatalf("outlink thumbnail = %+v", article.Thumbnail)
	}
	if article.RelatedTickers != nil || article.IsPremium || article.IsHosted {
		t.Fatalf("outlink unexpected fields = %+v", article)
	}
}

func TestParseNewsAssetOutlinkWithoutProvider(t *testing.T) {
	asset := outlinkAsset("out")
	delete(asset, "provider")
	delete(asset, "url")
	article, ok := parseNewsAsset(json.RawMessage(mustJSON(asset)))
	if !ok || article.Publisher != "" || article.Link != "" || article.CanonicalURL != "" {
		t.Fatalf("outlink = %+v, ok=%v", article, ok)
	}
}

func TestParseNewsAssetLinkFallsBackToCanonical(t *testing.T) {
	asset := storyAsset("abc")
	asset["contentAttributes"].(map[string]interface{})["clickthroughUrl"] = nil
	article, ok := parseNewsAsset(json.RawMessage(mustJSON(asset)))
	if !ok || article.Link != "https://example.com/abc" {
		t.Fatalf("Link = %q, ok=%v", article.Link, ok)
	}
}

func TestParseNewsAssetDescriptionPrecedence(t *testing.T) {
	asset := storyAsset("abc")
	asset["contentAttributes"].(map[string]interface{})["description"] = "attrs description"
	article, _ := parseNewsAsset(json.RawMessage(mustJSON(asset)))
	if article.Description != "attrs description" {
		t.Fatalf("Description = %q", article.Description)
	}
	asset["description"] = "asset description"
	article, _ = parseNewsAsset(json.RawMessage(mustJSON(asset)))
	if article.Description != "asset description" {
		t.Fatalf("asset description must win, got %q", article.Description)
	}
}

func TestParseNewsAssetSparse(t *testing.T) {
	article, ok := parseNewsAsset(json.RawMessage(`{"__typename":"Story","id":"bare","contentAttributes":null,"finance":null}`))
	if !ok {
		t.Fatal("sparse story was skipped")
	}
	want := models.NewsArticle{UUID: "bare", Type: "STORY"}
	if !reflect.DeepEqual(article, want) {
		t.Fatalf("sparse article = %+v", article)
	}
}

func TestParseNewsAssetRejects(t *testing.T) {
	for _, raw := range []string{``, `null`, `{}`, `{"__typename":"AssetError"}`, `"string"`, `[1,2]`, `{"__typename":"Story","title":42}`} {
		if article, ok := parseNewsAsset(json.RawMessage(raw)); ok {
			t.Errorf("parseNewsAsset(%s) = %+v, want skipped", raw, article)
		}
	}
}

func TestNewsThumbnailMapping(t *testing.T) {
	if got := newsThumbnail(nil); got != nil {
		t.Fatalf("nil image -> %+v", got)
	}
	if got := newsThumbnail(&newsImage{Width: 10, Height: 10}); got != nil {
		t.Fatalf("image without URL -> %+v", got)
	}

	var image newsImage
	raw := `{"url":"https://o.example/x.jpg","width":1400.0,"height":933,"resized":[
		{"url":"https://r.example/a.jpg","width":170,"height":128,"transformLabel":"170x128"},
		{"url":"","width":1,"height":1,"transformLabel":"broken"},
		{"url":"https://r.example/b.jpg","width":340,"height":256,"transformLabel":null}
	]}`
	if err := json.Unmarshal([]byte(raw), &image); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := newsThumbnail(&image)
	want := &models.NewsThumbnail{
		OriginalURL:    "https://o.example/x.jpg",
		OriginalWidth:  1400,
		OriginalHeight: 933,
		Resolutions: []models.ThumbnailResolution{
			{URL: "https://o.example/x.jpg", Width: 1400, Height: 933, Tag: "original"},
			{URL: "https://r.example/a.jpg", Width: 170, Height: 128, Tag: "170x128"},
			{URL: "https://r.example/b.jpg", Width: 340, Height: 256},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("thumbnail = %+v", got)
	}
}

func TestNewsThumbnailMissingInArticle(t *testing.T) {
	asset := storyAsset("abc")
	asset["contentAttributes"].(map[string]interface{})["thumbnail"] = nil
	article, ok := parseNewsAsset(json.RawMessage(mustJSON(asset)))
	if !ok || article.Thumbnail != nil {
		t.Fatalf("thumbnail = %+v, ok=%v", article.Thumbnail, ok)
	}
}

func TestParseNewsTime(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"2026-10-25T20:05:00Z", 1792958700},
		{"2026-10-25T20:05:00.123Z", 1792958700},
		{"2026-10-25T22:05:00+02:00", 1792958700},
		{"", 0},
		{"not a date", 0},
		{"2026-10-25", 0},
		{"1793045100", 0},
	}
	for _, tt := range tests {
		if got := parseNewsTime(tt.in); got != tt.want {
			t.Errorf("parseNewsTime(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestParseNewsAssetPublishTimeFallbacks(t *testing.T) {
	asset := storyAsset("abc")
	attrs := asset["contentAttributes"].(map[string]interface{})
	attrs["publishTime"] = nil
	attrs["displayTime"] = "garbage"
	article, ok := parseNewsAsset(json.RawMessage(mustJSON(asset)))
	if !ok {
		t.Fatal("story skipped")
	}
	if article.PublishTime != 0 || article.DisplayTime != 0 {
		t.Fatalf("missing/invalid times = %d / %d, want 0", article.PublishTime, article.DisplayTime)
	}
	if !article.PublishedAt().Equal(time.Unix(0, 0)) {
		t.Fatalf("PublishedAt() = %v", article.PublishedAt())
	}
}

// --- empty results and errors --------------------------------------------

func TestNewsGraphQLErrorsReturnEmpty(t *testing.T) {
	bodies := map[string]string{
		"errors-null-list": `{"data":{"lightyearList":null},"errors":[{"message":"Error","extensions":{"code":"NOT_FOUND"}}]}`,
		"errors-no-data":   `{"errors":[{"message":"Error"}]}`,
		"data-null":        `{"data":null}`,
		"empty-object":     `{}`,
		"main-null":        `{"data":{"lightyearList":{"main":null}}}`,
		"edges-null":       `{"data":{"lightyearList":{"main":{"edges":null,"pageInfo":null}}}}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply { return newsReply{body: body} })
			tkr := newNewsTicker(t, f, "MSFT")

			news, err := tkr.News(10, models.NewsTabNews)
			if err != nil {
				t.Fatalf("News() error: %v", err)
			}
			if news == nil || len(news) != 0 {
				t.Fatalf("News() = %#v, want empty non-nil slice", news)
			}
		})
	}
}

func TestNewsEmptyResultIsCached(t *testing.T) {
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		return newsReply{body: `{"data":{"lightyearList":null},"errors":[{"message":"Error"}]}`}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	for i := 0; i < 3; i++ {
		news, err := tkr.News(10, models.NewsTabNews)
		if err != nil || news == nil || len(news) != 0 {
			t.Fatalf("call %d: News() = %#v, %v", i, news, err)
		}
	}
	if f.count() != 1 {
		t.Fatalf("requests = %d, want 1 (empty result not cached)", f.count())
	}
}

func TestNewsMalformedJSON(t *testing.T) {
	for name, body := range map[string]string{
		"html":          "<html>Will be right back</html>",
		"truncated":     `{"data":{"lightyearList":`,
		"wrong-type":    `{"data":{"lightyearList":"oops"}}`,
		"edges-wrong":   `{"data":{"lightyearList":{"main":{"edges":{}}}}}`,
		"empty-body":    ``,
		"json-null-str": `"null"`,
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply { return newsReply{body: body} })
			tkr := newNewsTicker(t, f, "MSFT")

			news, err := tkr.News(10, models.NewsTabNews)
			if err == nil {
				t.Fatalf("News() = %+v, want error", news)
			}
			if !errors.Is(err, client.ErrInvalidResponse) {
				t.Fatalf("error = %v, want ErrInvalidResponse", err)
			}
			// Errors are not cached.
			_, _ = tkr.News(10, models.NewsTabNews)
			if f.count() != 2 {
				t.Fatalf("requests = %d, want 2 (errors must not be cached)", f.count())
			}
		})
	}
}

func TestNewsHTTPErrors(t *testing.T) {
	tests := []struct {
		status int
		check  func(error) bool
		name   string
	}{
		{http.StatusNotFound, client.IsNotFoundError, "not found"},
		{http.StatusTooManyRequests, client.IsRateLimitError, "rate limit"},
		{http.StatusUnauthorized, client.IsAuthError, "auth"},
		{http.StatusForbidden, client.IsAuthError, "auth"},
		{http.StatusInternalServerError, func(err error) bool { return errors.Is(err, client.ErrNetwork) }, "network"},
		{http.StatusTeapot, func(err error) bool {
			var yfErr *client.YFError
			return errors.As(err, &yfErr) && yfErr.Code == client.ErrCodeUnknown
		}, "unknown"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
				return newsReply{status: tt.status, body: `{"message":"Not Found"}`}
			})
			tkr := newNewsTicker(t, f, "MSFT")

			news, err := tkr.News(5, models.NewsTabNews)
			if err == nil || news != nil {
				t.Fatalf("News() = %+v, %v; want error", news, err)
			}
			if !tt.check(err) {
				t.Fatalf("error %v is not a %s error", err, tt.name)
			}
			_, _ = tkr.News(5, models.NewsTabNews)
			if f.count() != 2 {
				t.Fatalf("requests = %d, want 2 (errors must not be cached)", f.count())
			}
		})
	}
}

func TestNewsErrorOnLaterPageReturnsError(t *testing.T) {
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		if n == 1 {
			return newsReply{body: newsPage([]interface{}{storyAsset("a")}, strPtr("c1"), true)}
		}
		return newsReply{status: http.StatusTooManyRequests}
	})
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(5, models.NewsTabNews)
	if !client.IsRateLimitError(err) || news != nil {
		t.Fatalf("News() = %+v, %v; want rate-limit error and no partial result", news, err)
	}
	tkr.mu.RLock()
	cached := len(tkr.newsCache)
	tkr.mu.RUnlock()
	if cached != 0 {
		t.Fatalf("partial result was cached")
	}
}

func TestNewsNetworkError(t *testing.T) {
	f := newFakeNewsServer(t, echoHandler)
	tkr := newNewsTicker(t, f, "MSFT")
	f.srv.Close()

	if _, err := tkr.News(5, models.NewsTabNews); err == nil {
		t.Fatal("News() against a closed server should fail")
	}
}

func TestNewsInvalidTab(t *testing.T) {
	f := newFakeNewsServer(t, echoHandler)
	tkr := newNewsTicker(t, f, "MSFT")

	if _, err := tkr.News(5, models.NewsTabNews); err != nil {
		t.Fatalf("News() error: %v", err)
	}
	// Invalid tab must fail even after a cached call, without a request.
	for _, tab := range []models.NewsTab{"bogus", "latestNews", "pressrelease"} {
		news, err := tkr.News(5, tab)
		if !errors.Is(err, models.ErrInvalidNewsTab) || news != nil {
			t.Fatalf("News(%q) = %+v, %v; want ErrInvalidNewsTab", tab, news, err)
		}
		if !strings.Contains(err.Error(), string(tab)) {
			t.Fatalf("error %q should name the tab", err)
		}
	}
	if f.count() != 1 {
		t.Fatalf("requests = %d, want 1", f.count())
	}
}

// --- cache ---------------------------------------------------------------

func TestNewsCachedPerCountAndTab(t *testing.T) {
	setNewsLocale(t, "en-US", "US")
	f := newFakeNewsServer(t, echoHandler)
	tkr := newNewsTicker(t, f, "MSFT")

	news, err := tkr.News(2, models.NewsTabNews)
	if err != nil {
		t.Fatal(err)
	}
	press, err := tkr.News(5, models.NewsTabPressReleases)
	if err != nil {
		t.Fatal(err)
	}
	newsAgain, err := tkr.News(2, models.NewsTabNews)
	if err != nil {
		t.Fatal(err)
	}

	if got := articleIDs(news); !reflect.DeepEqual(got, []string{"news-0", "news-1"}) {
		t.Fatalf("news ids = %v", got)
	}
	if got := articleIDs(press); !reflect.DeepEqual(got, []string{"pressrelease-0", "pressrelease-1", "pressrelease-2", "pressrelease-3", "pressrelease-4"}) {
		t.Fatalf("press ids = %v", got)
	}
	if !reflect.DeepEqual(newsAgain, news) {
		t.Fatalf("cached news differs: %v vs %v", articleIDs(newsAgain), articleIDs(news))
	}
	if f.count() != 2 {
		t.Fatalf("requests = %d, want 2 (third call must hit the cache)", f.count())
	}

	// A different count is a different cache entry.
	if _, err := tkr.News(3, models.NewsTabNews); err != nil {
		t.Fatal(err)
	}
	// Case variants of a tab share one entry.
	if _, err := tkr.News(5, "Press Releases"); err != nil {
		t.Fatal(err)
	}
	if f.count() != 3 {
		t.Fatalf("requests = %d, want 3", f.count())
	}
}

func TestNewsCachedPerLocale(t *testing.T) {
	setNewsLocale(t, "en-US", "US")
	f := newFakeNewsServer(t, echoHandler)
	tkr := newNewsTicker(t, f, "MSFT")

	if _, err := tkr.News(2, models.NewsTabNews); err != nil {
		t.Fatal(err)
	}
	config.Get().SetLocale("en-GB", "GB")
	if _, err := tkr.News(2, models.NewsTabNews); err != nil {
		t.Fatal(err)
	}
	config.Get().SetLocale("en-US", "US")
	if _, err := tkr.News(2, models.NewsTabNews); err != nil {
		t.Fatal(err)
	}

	reqs := f.requests()
	if len(reqs) != 2 {
		t.Fatalf("requests = %d, want 2 (locale switch-back must hit the cache)", len(reqs))
	}
	if reqs[1].slug() != "list=finance-GB-en-GB-ticker-news" {
		t.Fatalf("second slug = %q", reqs[1].slug())
	}
}

func TestNewsClearCache(t *testing.T) {
	f := newFakeNewsServer(t, echoHandler)
	tkr := newNewsTicker(t, f, "MSFT")

	if _, err := tkr.News(2, models.NewsTabNews); err != nil {
		t.Fatal(err)
	}
	if _, err := tkr.News(2, models.NewsTabAll); err != nil {
		t.Fatal(err)
	}
	tkr.ClearCache()
	tkr.mu.RLock()
	cleared := tkr.newsCache == nil
	tkr.mu.RUnlock()
	if !cleared {
		t.Fatal("ClearCache did not reset the news cache")
	}
	if _, err := tkr.News(2, models.NewsTabNews); err != nil {
		t.Fatal(err)
	}
	if f.count() != 3 {
		t.Fatalf("requests = %d, want 3", f.count())
	}
}

func TestNewsClearCacheDuringFetchDoesNotRepopulate(t *testing.T) {
	arrived := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		if n == 1 {
			once.Do(func() { close(arrived) })
			<-release
		}
		return echoHandler(req, n)
	})
	tkr := newNewsTicker(t, f, "MSFT")

	done := make(chan error, 1)
	go func() {
		_, err := tkr.News(2, models.NewsTabNews)
		done <- err
	}()
	<-arrived
	tkr.ClearCache()
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("News() error: %v", err)
	}

	if _, err := tkr.News(2, models.NewsTabNews); err != nil {
		t.Fatal(err)
	}
	if f.count() != 2 {
		t.Fatalf("requests = %d, want 2 (stale fetch must not repopulate a cleared cache)", f.count())
	}
}

func TestNewsCacheResultsAreDeepCopies(t *testing.T) {
	f := newFakeNewsServer(t, func(req newsRequest, n int) newsReply {
		return newsReply{body: newsPage([]interface{}{storyAsset("orig")}, nil, false)}
	})
	tkr := newNewsTicker(t, f, "AAPL")

	first, err := tkr.News(10, models.NewsTabNews)
	if err != nil {
		t.Fatalf("News() error: %v", err)
	}
	first[0].Title = "mutated"
	first[0].RelatedTickers[0] = "XXXX"
	first[0].Thumbnail.Resolutions[0].URL = "https://example.test/mutated.jpg"
	first[0].Thumbnail.OriginalURL = "https://example.test/mutated.jpg"

	second, err := tkr.News(10, models.NewsTabNews)
	if err != nil {
		t.Fatalf("second News() error: %v", err)
	}
	if second[0].Title != "title orig" || second[0].RelatedTickers[0] != "MSFT" {
		t.Fatalf("news cache mutated: %+v", second[0])
	}
	if got := second[0].Thumbnail.Resolutions[0].URL; got != "https://media.example.com/orig.jpg" {
		t.Fatalf("nested resolution cache mutated: %q", got)
	}
	if got := second[0].Thumbnail.OriginalURL; got != "https://media.example.com/orig.jpg" {
		t.Fatalf("thumbnail cache mutated: %q", got)
	}
	if f.count() != 1 {
		t.Fatalf("requests = %d, want 1", f.count())
	}
}

func TestNewsConcurrentAccess(t *testing.T) {
	f := newFakeNewsServer(t, echoHandler)
	tkr := newNewsTicker(t, f, "MSFT")

	tabs := []models.NewsTab{models.NewsTabNews, models.NewsTabAll, models.NewsTabPressReleases}
	var wg sync.WaitGroup
	errs := make(chan error, 64)
	for i := 0; i < 48; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			count := 1 + i%4
			tab := tabs[i%len(tabs)]
			news, err := tkr.News(count, tab)
			if err != nil {
				errs <- err
				return
			}
			if len(news) != count {
				errs <- fmt.Errorf("News(%d, %q) returned %d articles", count, tab, len(news))
				return
			}
			news[0].Title = "mutated concurrently"
			if i%16 == 0 {
				tkr.ClearCache()
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	news, err := tkr.News(1, models.NewsTabNews)
	if err != nil || news[0].Title != "title news-0" {
		t.Fatalf("after concurrency: %+v, %v", news, err)
	}
}

// --- helpers -------------------------------------------------------------

func mustJSON(v interface{}) string {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func mustUnix(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse %q: %v", value, err)
	}
	return parsed.Unix()
}

// Integration test - requires network access
// func TestNewsIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	ticker, err := New("AAPL")
// 	if err != nil {
// 		t.Fatalf("Failed to create ticker: %v", err)
// 	}
// 	defer ticker.Close()
//
// 	news, err := ticker.News(5, models.NewsTabNews)
// 	if err != nil {
// 		t.Fatalf("Failed to get news: %v", err)
// 	}
//
// 	t.Logf("Got %d news articles", len(news))
// 	for _, article := range news {
// 		t.Logf("  - %s: %s", article.Publisher, article.Title)
// 	}
// }
