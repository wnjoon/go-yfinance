package screener

import (
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

const fakeScreenerResponse = `{"finance":{"result":[{"total":2,"count":1,"quotes":[{"symbol":"AAPL","regularMarketPrice":190.5}]}],"error":null}}`

// recordedRequest captures one outgoing screener request.
type recordedRequest struct {
	method string
	url    string
	params url.Values
	body   map[string]any
}

// fakeDoer records requests and returns a canned response.
type fakeDoer struct {
	requests []recordedRequest
	respBody string
	err      error
}

func (f *fakeDoer) Get(rawURL string, params url.Values) (*client.Response, error) {
	f.requests = append(f.requests, recordedRequest{method: "GET", url: rawURL, params: params})
	if f.err != nil {
		return nil, f.err
	}
	return &client.Response{StatusCode: 200, Body: f.response()}, nil
}

func (f *fakeDoer) PostJSON(rawURL string, params url.Values, body []byte) (*client.Response, error) {
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	f.requests = append(f.requests, recordedRequest{method: "POST", url: rawURL, params: params, body: decoded})
	if f.err != nil {
		return nil, f.err
	}
	return &client.Response{StatusCode: 200, Body: f.response()}, nil
}

func (f *fakeDoer) response() string {
	if f.respBody != "" {
		return f.respBody
	}
	return fakeScreenerResponse
}

func newFakeScreener() (*Screener, *fakeDoer) {
	f := &fakeDoer{}
	return &Screener{http: f}, f
}

// onlyRequest returns the single recorded request or fails the test.
func onlyRequest(t *testing.T, f *fakeDoer) recordedRequest {
	t.Helper()
	if len(f.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(f.requests))
	}
	return f.requests[0]
}

// jsonRoundTrip normalizes a value to its decoded-JSON form for comparison.
func jsonRoundTrip(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func TestScreenPredefinedOffsetSendsSizeNotCount(t *testing.T) {
	s, f := newFakeScreener()

	params := &models.ScreenerParams{Offset: 50, Count: 100, UserIDType: "guid"}
	result, err := s.Screen(models.ScreenerDayGainers, params)
	if err != nil {
		t.Fatalf("Screen: %v", err)
	}

	req := onlyRequest(t, f)
	if req.method != "POST" {
		t.Fatalf("method = %s, want POST", req.method)
	}
	if req.url != endpoints.ScreenerURL {
		t.Errorf("url = %s, want %s", req.url, endpoints.ScreenerURL)
	}
	if len(req.params) != 0 {
		t.Errorf("unexpected query params: %v", req.params)
	}

	if _, ok := req.body["count"]; ok {
		t.Errorf("body must not contain count: %v", req.body)
	}
	want := map[string]any{
		"offset":     float64(50),
		"size":       float64(100),
		"sortField":  "percentchange",
		"sortType":   "DESC",
		"userId":     "",
		"userIdType": "guid",
		"quoteType":  "EQUITY",
		"query":      jsonRoundTrip(t, PredefinedScreenerQueries["day_gainers"].Query.ToDict()),
	}
	if !reflect.DeepEqual(req.body, want) {
		t.Errorf("body mismatch\n got: %v\nwant: %v", req.body, want)
	}

	if result.Offset != 50 || result.Total != 2 || len(result.Quotes) != 1 || result.Quotes[0].Symbol != "AAPL" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestScreenPredefinedOffsetAllScreeners(t *testing.T) {
	for _, name := range models.AllPredefinedScreeners() {
		t.Run(string(name), func(t *testing.T) {
			predefined, ok := PredefinedScreenerQueries[string(name)]
			if !ok {
				t.Fatalf("no POST query mapping for %s", name)
			}

			s, f := newFakeScreener()
			params := models.DefaultScreenerParams()
			params.Offset = 25
			if _, err := s.Screen(name, &params); err != nil {
				t.Fatalf("Screen: %v", err)
			}

			req := onlyRequest(t, f)
			if req.method != "POST" {
				t.Fatalf("method = %s, want POST", req.method)
			}
			if _, ok := req.body["count"]; ok {
				t.Errorf("body must not contain count")
			}
			if got := req.body["size"]; got != float64(25) {
				t.Errorf("size = %v, want 25", got)
			}
			if got := req.body["offset"]; got != float64(25) {
				t.Errorf("offset = %v, want 25", got)
			}
			if got := req.body["sortField"]; got != predefined.SortField {
				t.Errorf("sortField = %v, want %s", got, predefined.SortField)
			}
			if got, want := req.body["sortType"], sortType(predefined.SortAsc); got != want {
				t.Errorf("sortType = %v, want %s", got, want)
			}
			if got, want := req.body["quoteType"], predefined.Query.QuoteType(); got != want {
				t.Errorf("quoteType = %v, want %s", got, want)
			}
			if got, want := req.body["query"], jsonRoundTrip(t, predefined.Query.ToDict()); !reflect.DeepEqual(got, want) {
				t.Errorf("query mismatch\n got: %v\nwant: %v", got, want)
			}
		})
	}
}

func TestScreenPredefinedOffsetBodyVariants(t *testing.T) {
	tests := []struct {
		name          string
		screener      models.PredefinedScreener
		params        models.ScreenerParams
		wantSize      any // nil means absent
		wantCount     any // nil means absent
		wantSortField string
		wantSortType  string
	}{
		{
			name:          "count becomes size",
			screener:      models.ScreenerMostActives,
			params:        models.ScreenerParams{Offset: 10, Count: 40},
			wantSize:      float64(40),
			wantSortField: "dayvolume",
			wantSortType:  "DESC",
		},
		{
			name:          "max count 250 becomes size",
			screener:      models.ScreenerMostActives,
			params:        models.ScreenerParams{Offset: 10, Count: 250},
			wantSize:      float64(250),
			wantSortField: "dayvolume",
			wantSortType:  "DESC",
		},
		{
			name:          "zero count omits size and count",
			screener:      models.ScreenerMostActives,
			params:        models.ScreenerParams{Offset: 10},
			wantSortField: "dayvolume",
			wantSortType:  "DESC",
		},
		{
			name:          "explicit size wins and count passes through",
			screener:      models.ScreenerMostActives,
			params:        models.ScreenerParams{Offset: 10, Count: 30, Size: 60},
			wantSize:      float64(60),
			wantCount:     float64(30),
			wantSortField: "dayvolume",
			wantSortType:  "DESC",
		},
		{
			name:          "explicit size without count",
			screener:      models.ScreenerMostActives,
			params:        models.ScreenerParams{Offset: 10, Size: 60},
			wantSize:      float64(60),
			wantSortField: "dayvolume",
			wantSortType:  "DESC",
		},
		{
			name:          "custom sort field overrides predefined",
			screener:      models.ScreenerDayGainers,
			params:        models.ScreenerParams{Offset: 5, Count: 10, SortField: "dayvolume"},
			wantSize:      float64(10),
			wantSortField: "dayvolume",
			wantSortType:  "DESC",
		},
		{
			name:          "ticker sort field falls back to predefined",
			screener:      models.ScreenerDayGainers,
			params:        models.ScreenerParams{Offset: 5, Count: 10, SortField: "ticker"},
			wantSize:      float64(10),
			wantSortField: "percentchange",
			wantSortType:  "DESC",
		},
		{
			name:          "sortAsc true forces ASC",
			screener:      models.ScreenerDayGainers,
			params:        models.ScreenerParams{Offset: 5, Count: 10, SortAsc: true},
			wantSize:      float64(10),
			wantSortField: "percentchange",
			wantSortType:  "ASC",
		},
		{
			name:          "ascending predefined keeps ASC",
			screener:      models.ScreenerDayLosers,
			params:        models.ScreenerParams{Offset: 5, Count: 10},
			wantSize:      float64(10),
			wantSortField: "percentchange",
			wantSortType:  "ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, f := newFakeScreener()
			params := tt.params
			if _, err := s.Screen(tt.screener, &params); err != nil {
				t.Fatalf("Screen: %v", err)
			}
			req := onlyRequest(t, f)
			if req.method != "POST" {
				t.Fatalf("method = %s, want POST", req.method)
			}
			assertBodyField(t, req.body, "size", tt.wantSize)
			assertBodyField(t, req.body, "count", tt.wantCount)
			assertBodyField(t, req.body, "sortField", tt.wantSortField)
			assertBodyField(t, req.body, "sortType", tt.wantSortType)
			assertBodyField(t, req.body, "offset", float64(tt.params.Offset))

			if params != tt.params {
				t.Errorf("Screen mutated caller params: got %+v, want %+v", params, tt.params)
			}
		})
	}
}

func assertBodyField(t *testing.T, body map[string]any, key string, want any) {
	t.Helper()
	got, ok := body[key]
	if want == nil {
		if ok {
			t.Errorf("body[%q] = %v, want absent", key, got)
		}
		return
	}
	if !ok {
		t.Errorf("body[%q] missing, want %v", key, want)
		return
	}
	if got != want {
		t.Errorf("body[%q] = %v, want %v", key, got, want)
	}
}

func TestScreenPredefinedWithoutOffsetUsesGet(t *testing.T) {
	tests := []struct {
		name       string
		screener   models.PredefinedScreener
		params     *models.ScreenerParams
		wantParams url.Values
	}{
		{
			name:     "nil params use defaults",
			screener: models.ScreenerDayGainers,
			params:   nil,
			wantParams: url.Values{
				"count": {"25"}, "offset": {"0"}, "sortField": {"ticker"},
			},
		},
		{
			name:     "count and sortAsc",
			screener: models.ScreenerMostActives,
			params:   &models.ScreenerParams{Count: 10, SortField: "dayvolume", SortAsc: true},
			wantParams: url.Values{
				"count": {"10"}, "offset": {"0"}, "sortField": {"dayvolume"}, "sortAsc": {"true"},
			},
		},
		{
			name:     "empty sort field omitted",
			screener: models.ScreenerMostActives,
			params:   &models.ScreenerParams{Count: 5},
			wantParams: url.Values{
				"count": {"5"}, "offset": {"0"},
			},
		},
		{
			name:     "size overrides count on GET",
			screener: models.ScreenerMostActives,
			params:   &models.ScreenerParams{Count: 5, Size: 70},
			wantParams: url.Values{
				"count": {"70"}, "offset": {"0"},
			},
		},
		{
			name:     "negative offset stays on GET",
			screener: models.ScreenerMostActives,
			params:   &models.ScreenerParams{Count: 5, Offset: -1},
			wantParams: url.Values{
				"count": {"5"}, "offset": {"-1"},
			},
		},
		{
			name:     "unknown screener with offset falls back to GET",
			screener: models.PredefinedScreener("not_a_screener"),
			params:   &models.ScreenerParams{Count: 5, Offset: 20},
			wantParams: url.Values{
				"count": {"5"}, "offset": {"20"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, f := newFakeScreener()
			result, err := s.Screen(tt.screener, tt.params)
			if err != nil {
				t.Fatalf("Screen: %v", err)
			}
			req := onlyRequest(t, f)
			if req.method != "GET" {
				t.Fatalf("method = %s, want GET", req.method)
			}
			wantURL := endpoints.ScreenerURL + "/predefined/" + string(tt.screener)
			if req.url != wantURL {
				t.Errorf("url = %s, want %s", req.url, wantURL)
			}
			if !reflect.DeepEqual(req.params, tt.wantParams) {
				t.Errorf("params = %v, want %v", req.params, tt.wantParams)
			}
			if len(result.Quotes) != 1 {
				t.Errorf("expected 1 quote, got %d", len(result.Quotes))
			}
		})
	}
}

func TestScreenWithQueryBody(t *testing.T) {
	eq, _ := models.NewEquityQuery("eq", []any{"region", "us"})
	fq, _ := models.NewFundQuery("eq", []any{"exchange", "NAS"})
	etf, _ := models.NewETFQuery("eq", []any{"region", "us"})

	tests := []struct {
		name   string
		query  models.ScreenerQueryBuilder
		params *models.ScreenerParams
		want   map[string]any
	}{
		{
			name:   "defaults without size",
			query:  eq,
			params: nil,
			want: map[string]any{
				"offset": float64(0), "count": float64(25), "sortField": "ticker", "sortType": "DESC",
				"userId": "", "userIdType": "guid", "quoteType": "EQUITY",
			},
		},
		{
			name:  "size sent alongside count",
			query: eq,
			params: &models.ScreenerParams{
				Offset: 100, Count: 25, Size: 200, SortField: "percentchange", SortAsc: true, UserIDType: "guid",
			},
			want: map[string]any{
				"offset": float64(100), "count": float64(25), "size": float64(200), "sortField": "percentchange",
				"sortType": "ASC", "userId": "", "userIdType": "guid", "quoteType": "EQUITY",
			},
		},
		{
			name:   "fund query with offset keeps count",
			query:  fq,
			params: &models.ScreenerParams{Offset: 10, Count: 50, SortField: "fundnetassets"},
			want: map[string]any{
				"offset": float64(10), "count": float64(50), "sortField": "fundnetassets", "sortType": "DESC",
				"userId": "", "userIdType": "", "quoteType": "MUTUALFUND",
			},
		},
		{
			name:   "etf query with max size",
			query:  etf,
			params: &models.ScreenerParams{Count: 250, Size: 250},
			want: map[string]any{
				"offset": float64(0), "count": float64(250), "size": float64(250), "sortField": "", "sortType": "DESC",
				"userId": "", "userIdType": "", "quoteType": "ETF",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, f := newFakeScreener()
			if _, err := s.ScreenWithQuery(tt.query, tt.params); err != nil {
				t.Fatalf("ScreenWithQuery: %v", err)
			}
			req := onlyRequest(t, f)
			if req.method != "POST" || req.url != endpoints.ScreenerURL {
				t.Fatalf("request = %s %s, want POST %s", req.method, req.url, endpoints.ScreenerURL)
			}
			want := make(map[string]any, len(tt.want)+1)
			for k, v := range tt.want {
				want[k] = v
			}
			want["query"] = jsonRoundTrip(t, tt.query.ToDict())
			if !reflect.DeepEqual(req.body, want) {
				t.Errorf("body mismatch\n got: %v\nwant: %v", req.body, want)
			}
		})
	}
}

func TestScreenerLimitValidation(t *testing.T) {
	eq, _ := models.NewEquityQuery("eq", []any{"region", "us"})

	tests := []struct {
		name    string
		params  models.ScreenerParams
		wantErr string
	}{
		{"count over limit", models.ScreenerParams{Count: 251}, "count to 250"},
		{"size over limit", models.ScreenerParams{Count: 25, Size: 251}, "size to 250"},
		{"count over limit with offset", models.ScreenerParams{Count: 300, Offset: 10}, "count to 250"},
		{"size over limit with offset", models.ScreenerParams{Size: 300, Offset: 10}, "size to 250"},
		{"at limit ok", models.ScreenerParams{Count: 250, Size: 250}, ""},
		{"at limit with offset ok", models.ScreenerParams{Count: 250, Offset: 10}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callers := map[string]func(*Screener, *models.ScreenerParams) error{
				"Screen": func(s *Screener, p *models.ScreenerParams) error {
					_, err := s.Screen(models.ScreenerDayGainers, p)
					return err
				},
				"ScreenWithQuery": func(s *Screener, p *models.ScreenerParams) error {
					_, err := s.ScreenWithQuery(eq, p)
					return err
				},
			}
			for name, call := range callers {
				s, f := newFakeScreener()
				params := tt.params
				err := call(s, &params)
				if tt.wantErr == "" {
					if err != nil {
						t.Errorf("%s: unexpected error %v", name, err)
					}
					continue
				}
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("%s: error = %v, want containing %q", name, err, tt.wantErr)
				}
				if len(f.requests) != 0 {
					t.Errorf("%s: request sent despite validation error", name)
				}
			}
		})
	}
}

func TestScreenConvenienceMethodsUseGet(t *testing.T) {
	tests := []struct {
		name     string
		call     func(*Screener) (*models.ScreenerResult, error)
		screener models.PredefinedScreener
	}{
		{"DayGainers", func(s *Screener) (*models.ScreenerResult, error) { return s.DayGainers(7) }, models.ScreenerDayGainers},
		{"DayLosers", func(s *Screener) (*models.ScreenerResult, error) { return s.DayLosers(7) }, models.ScreenerDayLosers},
		{"MostActives", func(s *Screener) (*models.ScreenerResult, error) { return s.MostActives(7) }, models.ScreenerMostActives},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, f := newFakeScreener()
			if _, err := tt.call(s); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			req := onlyRequest(t, f)
			if req.method != "GET" || req.url != endpoints.ScreenerURL+"/predefined/"+string(tt.screener) {
				t.Errorf("request = %s %s", req.method, req.url)
			}
			if got := req.params.Get("count"); got != "7" {
				t.Errorf("count = %s, want 7", got)
			}
		})
	}
}

func TestScreenRequestErrors(t *testing.T) {
	eq, _ := models.NewEquityQuery("eq", []any{"region", "us"})
	transportErr := errors.New("boom")

	t.Run("transport error on GET", func(t *testing.T) {
		s, f := newFakeScreener()
		f.err = transportErr
		_, err := s.Screen(models.ScreenerDayGainers, nil)
		if !errors.Is(err, transportErr) {
			t.Errorf("error = %v, want wrapping %v", err, transportErr)
		}
	})

	t.Run("transport error on offset POST", func(t *testing.T) {
		s, f := newFakeScreener()
		f.err = transportErr
		_, err := s.Screen(models.ScreenerDayGainers, &models.ScreenerParams{Offset: 1, Count: 1})
		if !errors.Is(err, transportErr) {
			t.Errorf("error = %v, want wrapping %v", err, transportErr)
		}
	})

	t.Run("API error on POST", func(t *testing.T) {
		s, f := newFakeScreener()
		f.respBody = `{"finance":{"result":null,"error":{"code":"Bad Request","description":"invalid size"}}}`
		_, err := s.ScreenWithQuery(eq, nil)
		if err == nil || !strings.Contains(err.Error(), "invalid size") {
			t.Errorf("error = %v, want API error", err)
		}
	})

	t.Run("empty result keeps offset", func(t *testing.T) {
		s, f := newFakeScreener()
		f.respBody = `{"finance":{"result":[],"error":null}}`
		result, err := s.Screen(models.ScreenerDayGainers, &models.ScreenerParams{Offset: 75, Count: 25})
		if err != nil {
			t.Fatalf("Screen: %v", err)
		}
		if result.Offset != 75 || len(result.Quotes) != 0 {
			t.Errorf("unexpected result: %+v", result)
		}
	})
}
