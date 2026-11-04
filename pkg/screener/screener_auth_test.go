package screener

import (
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// statusDoer returns a fixed HTTP status and body for every request.
type statusDoer struct {
	status int
	body   string
	nilRes bool
	calls  int
}

func (d *statusDoer) Get(string, url.Values) (*client.Response, error) {
	d.calls++
	if d.nilRes {
		return nil, nil
	}
	return &client.Response{StatusCode: d.status, Body: d.body}, nil
}

func (d *statusDoer) PostJSON(string, url.Values, []byte) (*client.Response, error) {
	d.calls++
	if d.nilRes {
		return nil, nil
	}
	return &client.Response{StatusCode: d.status, Body: d.body}, nil
}

func TestBaseParamsMatchPython(t *testing.T) {
	want := url.Values{
		"corsDomain": {"finance.yahoo.com"},
		"formatted":  {"false"},
		"lang":       {"en-US"},
		"region":     {"US"},
	}
	if got := baseParams(); !reflect.DeepEqual(got, want) {
		t.Errorf("baseParams() = %v, want %v", got, want)
	}

	// Each call returns a fresh map so request builders can't leak state.
	a := baseParams()
	a.Set("scrIds", "x")
	if baseParams().Has("scrIds") {
		t.Error("baseParams() shares state between calls")
	}
}

func TestScreenerPredefinedURL(t *testing.T) {
	want := "https://query1.finance.yahoo.com/v1/finance/screener/predefined/saved"
	if endpoints.ScreenerPredefinedURL != want {
		t.Errorf("ScreenerPredefinedURL = %s, want %s", endpoints.ScreenerPredefinedURL, want)
	}
}

func TestScreenWithQuerySendsBaseParams(t *testing.T) {
	queries := map[string]models.ScreenerQueryBuilder{}
	eq, err := models.NewEquityQuery("eq", []any{"region", "us"})
	if err != nil {
		t.Fatal(err)
	}
	queries["equity"] = eq
	fq, err := models.NewFundQuery("eq", []any{"exchange", "NAS"})
	if err != nil {
		t.Fatal(err)
	}
	queries["fund"] = fq

	for name, q := range queries {
		t.Run(name, func(t *testing.T) {
			s, f := newFakeScreener()
			if _, err := s.ScreenWithQuery(q, nil); err != nil {
				t.Fatalf("ScreenWithQuery: %v", err)
			}
			req := onlyRequest(t, f)
			if req.method != "POST" || req.url != endpoints.ScreenerURL {
				t.Errorf("request = %s %s", req.method, req.url)
			}
			if !reflect.DeepEqual(req.params, baseParams()) {
				t.Errorf("params = %v, want %v", req.params, baseParams())
			}
		})
	}
}

func TestScreenHTTPErrors(t *testing.T) {
	const (
		notFoundJSON = `{"finance":{"result":null,"error":{"code":"Not Found","description":"No screener records found. Check if scrIds and marketRegion combination are correct"}}}`
		crumbJSON    = `{"finance":{"result":null,"error":{"code":"Unauthorized","description":"Invalid Crumb"}}}`
		htmlPage     = `<!doctype html><html lang="en-us"><head><title>Yahoo</title></head></html>`
	)

	tests := []struct {
		name      string
		status    int
		body      string
		offset    int
		is        func(error) bool
		isName    string
		wantInMsg []string
	}{
		{"unknown predefined screener (404 JSON)", 404, notFoundJSON, 0, client.IsNotFoundError, "IsNotFoundError",
			[]string{"screener API error", "Not Found", "No screener records found"}},
		{"invalid crumb on POST (401 JSON)", 401, crumbJSON, 10, client.IsAuthError, "IsAuthError",
			[]string{"Unauthorized", "Invalid Crumb"}},
		{"rate limited (429 HTML)", 429, htmlPage, 0, client.IsRateLimitError, "IsRateLimitError",
			[]string{"screener request failed"}},
		{"rate limited on POST (429 empty)", 429, "", 10, client.IsRateLimitError, "IsRateLimitError",
			[]string{"screener request failed"}},
		{"forbidden (403 HTML)", 403, htmlPage, 0, client.IsAuthError, "IsAuthError",
			[]string{"screener request failed"}},
		{"server error (500 HTML)", 500, htmlPage, 0, func(err error) bool { return errors.Is(err, client.ErrNetwork) }, "ErrNetwork",
			[]string{"server error: HTTP 500"}},
		{"server error with JSON envelope (503)", 503, `{"finance":{"result":null,"error":{"code":"Service Unavailable","description":"try later"}}}`, 0,
			func(err error) bool { return errors.Is(err, client.ErrNetwork) }, "ErrNetwork",
			[]string{"Service Unavailable", "try later"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &statusDoer{status: tt.status, body: tt.body}
			s := &Screener{http: d}
			result, err := s.Screen(models.ScreenerMostActives, &models.ScreenerParams{Count: 5, Offset: tt.offset})
			if err == nil {
				t.Fatalf("expected error, got result %+v", result)
			}
			if !tt.is(err) {
				t.Errorf("%s(%v) = false", tt.isName, err)
			}
			for _, s := range tt.wantInMsg {
				if !strings.Contains(err.Error(), s) {
					t.Errorf("error %q does not contain %q", err, s)
				}
			}
			if d.calls != 1 {
				t.Errorf("requests = %d, want 1", d.calls)
			}
		})
	}
}

func TestScreenHTTPErrorBodyIsTruncated(t *testing.T) {
	body := strings.Repeat("x", 5000)
	s := &Screener{http: &statusDoer{status: 418, body: body}}
	_, err := s.Screen(models.ScreenerMostActives, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(err.Error()) > 400 {
		t.Errorf("error message not truncated: %d bytes", len(err.Error()))
	}
	if !strings.Contains(err.Error(), "HTTP 418") {
		t.Errorf("error %q lacks status", err)
	}
}

func TestScreenNilResponse(t *testing.T) {
	s := &Screener{http: &statusDoer{nilRes: true}}
	if _, err := s.Screen(models.ScreenerMostActives, nil); !errors.Is(err, client.ErrInvalidResponse) {
		t.Errorf("err = %v, want ErrInvalidResponse", err)
	}
}

func TestScreenSuccessStatusesStillParse(t *testing.T) {
	for _, status := range []int{200, 204, 304} {
		s := &Screener{http: &statusDoer{status: status, body: fakeScreenerResponse}}
		result, err := s.Screen(models.ScreenerMostActives, nil)
		if err != nil {
			t.Fatalf("status %d: %v", status, err)
		}
		if len(result.Quotes) != 1 {
			t.Errorf("status %d: quotes = %d, want 1", status, len(result.Quotes))
		}
	}
}

func TestDefaultTransportUsesCrumbAuth(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.auth == nil {
		t.Fatal("New() did not create an AuthManager")
	}
	d, ok := s.doer().(crumbDoer)
	if !ok {
		t.Fatalf("default doer = %T, want crumbDoer", s.doer())
	}
	if d.auth != s.auth {
		t.Error("crumbDoer does not use the Screener's AuthManager")
	}
}

func TestWithClientGetsAuthManager(t *testing.T) {
	c, err := client.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	s, err := New(WithClient(c))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.client != c || s.auth == nil {
		t.Fatalf("client=%p want %p, auth=%v", s.client, c, s.auth)
	}
	if _, ok := s.doer().(crumbDoer); !ok {
		t.Fatalf("doer = %T, want crumbDoer", s.doer())
	}
}

// TestScreenerIntegration hits Yahoo. Before this test existed the request
// shape tests passed while every live screener call failed (HTML from the old
// /predefined/<name> URL, "Invalid Crumb" from the crumb-less POST).
func TestScreenerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	gainers, err := s.DayGainers(5)
	if err != nil {
		t.Fatalf("DayGainers: %v", err)
	}
	if len(gainers.Quotes) == 0 || len(gainers.Quotes) > 5 {
		t.Errorf("DayGainers(5) returned %d quotes", len(gainers.Quotes))
	}

	page2, err := s.Screen(models.ScreenerMostActives, &models.ScreenerParams{Offset: 5, Count: 7})
	if err != nil {
		t.Fatalf("Screen with offset: %v", err)
	}
	if len(page2.Quotes) != 7 {
		t.Errorf("offset page returned %d quotes, want 7 (size)", len(page2.Quotes))
	}

	q, err := models.NewEquityQuery("eq", []any{"region", "us"})
	if err != nil {
		t.Fatal(err)
	}
	custom, err := s.ScreenWithQuery(q, &models.ScreenerParams{Size: 3, SortField: "ticker"})
	if err != nil {
		t.Fatalf("ScreenWithQuery: %v", err)
	}
	if len(custom.Quotes) != 3 {
		t.Errorf("custom query returned %d quotes, want 3", len(custom.Quotes))
	}

	_, err = s.Screen(models.PredefinedScreener("not_a_real_screener_xyz"), nil)
	if !client.IsNotFoundError(err) {
		t.Errorf("unknown screener err = %v, want not-found", err)
	}
}
