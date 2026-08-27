package client

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
)

type scriptedAuthResponse struct {
	method  string
	rawURL  string
	status  int
	body    string
	headers map[string]string
	cookies map[string]string
	err     error
}

type scriptedAuthClient struct {
	t          *testing.T
	steps      []scriptedAuthResponse
	calls      []string
	cookies    []string
	cookieSets []map[string]string
}

func newScriptedAuthClient(t *testing.T, steps ...scriptedAuthResponse) *scriptedAuthClient {
	t.Helper()
	return &scriptedAuthClient{t: t, steps: steps}
}

func (c *scriptedAuthClient) Get(rawURL string, params url.Values) (*Response, error) {
	c.t.Helper()
	return c.next("GET", rawURL)
}

func (c *scriptedAuthClient) Post(rawURL string, params url.Values, body map[string]string) (*Response, error) {
	c.t.Helper()
	return c.next("POST", rawURL)
}

func (c *scriptedAuthClient) SetCookie(cookie string) {
	c.cookies = append(c.cookies, cookie)
}

func (c *scriptedAuthClient) SetCookies(cookies map[string]string) {
	stored := make(map[string]string, len(cookies))
	for name, value := range cookies {
		stored[name] = value
	}
	c.cookieSets = append(c.cookieSets, stored)
}

func (c *scriptedAuthClient) next(method, rawURL string) (*Response, error) {
	if len(c.steps) == 0 {
		c.t.Fatalf("unexpected %s %s", method, rawURL)
	}
	step := c.steps[0]
	c.steps = c.steps[1:]
	c.calls = append(c.calls, method+" "+rawURL)
	if step.method != method {
		c.t.Fatalf("expected method %s, got %s for %s", step.method, method, rawURL)
	}
	if step.rawURL != rawURL {
		c.t.Fatalf("expected URL %s, got %s", step.rawURL, rawURL)
	}
	if step.err != nil {
		return nil, step.err
	}
	return &Response{
		StatusCode: step.status,
		Body:       step.body,
		Headers:    step.headers,
		Cookies:    step.cookies,
	}, nil
}

func (c *scriptedAuthClient) assertDrained(t *testing.T) {
	t.Helper()
	if len(c.steps) != 0 {
		t.Fatalf("expected all scripted responses to be used, %d left", len(c.steps))
	}
}

func hiddenConsentHTML(csrfToken, sessionID string) string {
	return fmt.Sprintf(
		`<html><form><input type="hidden" name="csrfToken" value="%s"><input type="hidden" name="sessionId" value="%s"></form></html>`,
		csrfToken,
		sessionID,
	)
}

func newScriptedAuthManager(client *scriptedAuthClient, strategy AuthStrategy) *AuthManager {
	return &AuthManager{
		client:   client,
		strategy: strategy,
	}
}

func TestExtractInputValue(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		input    string
		expected string
	}{
		{
			name:     "basic input",
			html:     `<input name="csrfToken" type="hidden" value="abc123"/>`,
			input:    "csrfToken",
			expected: "abc123",
		},
		{
			name:     "value before name",
			html:     `<input type="hidden" value="xyz789" name="sessionId"/>`,
			input:    "sessionId",
			expected: "xyz789",
		},
		{
			name:     "double quotes",
			html:     `<input name="token" value="test-value">`,
			input:    "token",
			expected: "test-value",
		},
		{
			name:     "single quotes",
			html:     `<input name='token' value='test-value'>`,
			input:    "token",
			expected: "test-value",
		},
		{
			name:     "not found",
			html:     `<input name="other" value="value">`,
			input:    "missing",
			expected: "",
		},
		{
			name:     "complex html",
			html:     `<html><body><form><input type="hidden" name="csrfToken" value="complex-token-123"/><input name="sessionId" value="session-456"/></form></body></html>`,
			input:    "csrfToken",
			expected: "complex-token-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractInputValue(tt.html, tt.input)
			if result != tt.expected {
				t.Errorf("extractInputValue(%q, %q) = %q, want %q", tt.html, tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewAuthManager(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	auth := NewAuthManager(client)

	if auth.client != client {
		t.Error("AuthManager should have reference to client")
	}
	if auth.strategy != StrategyBasic {
		t.Error("Default strategy should be StrategyBasic")
	}
	if auth.crumb != "" {
		t.Error("Initial crumb should be empty")
	}
}

func TestAuthManagerBasicSuccessUsesQuery1(t *testing.T) {
	fakeClient := newScriptedAuthClient(t,
		scriptedAuthResponse{
			method:  "GET",
			rawURL:  endpoints.CookieURL,
			status:  404,
			headers: map[string]string{"Set-Cookie": "A3=cookie-secret; Path=/; Secure"},
		},
		scriptedAuthResponse{
			method: "GET",
			rawURL: endpoints.CrumbURL,
			status: 200,
			body:   "basic-crumb\n",
		},
	)
	auth := newScriptedAuthManager(fakeClient, StrategyBasic)

	crumb, err := auth.GetCrumb()
	if err != nil {
		t.Fatalf("GetCrumb returned error: %v", err)
	}
	if crumb != "basic-crumb" {
		t.Fatalf("expected basic crumb, got %q", crumb)
	}
	if endpoints.CrumbURL != "https://query1.finance.yahoo.com/v1/test/getcrumb" {
		t.Fatalf("expected Basic crumb endpoint to use query1, got %s", endpoints.CrumbURL)
	}
	if len(fakeClient.cookies) != 1 || fakeClient.cookies[0] != "A3=cookie-secret" {
		t.Fatalf("expected A3 cookie to be stored, got %v", fakeClient.cookies)
	}
	fakeClient.assertDrained(t)
}

func TestAuthManagerBasicStoresAllStructuredCookies(t *testing.T) {
	fakeClient := newScriptedAuthClient(t,
		scriptedAuthResponse{
			method:  "GET",
			rawURL:  endpoints.CookieURL,
			status:  404,
			cookies: map[string]string{"A1": "first", "A3": "second"},
		},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CrumbURL, status: 200, body: "crumb"},
	)
	auth := newScriptedAuthManager(fakeClient, StrategyBasic)

	if _, err := auth.GetCrumb(); err != nil {
		t.Fatalf("GetCrumb() error: %v", err)
	}
	if len(fakeClient.cookieSets) != 1 {
		t.Fatalf("SetCookies calls = %d, want 1", len(fakeClient.cookieSets))
	}
	if got := fakeClient.cookieSets[0]; got["A1"] != "first" || got["A3"] != "second" {
		t.Fatalf("stored cookies = %v", got)
	}
	if auth.cookie != "A1,A3" {
		t.Fatalf("cookie summary = %q, want %q", auth.cookie, "A1,A3")
	}
}

func TestExtractCookiesCycleTLSHeaderFallback(t *testing.T) {
	fakeClient := newScriptedAuthClient(t)
	auth := newScriptedAuthManager(fakeClient, StrategyBasic)
	auth.extractCookies(&Response{Headers: map[string]string{
		"set-cookie": "A1=first; Expires=Wed, 27 Aug 2026 00:00:00 GMT; Path=/,/A3=second; Secure",
	}})

	want := []string{"A1=first", "A3=second"}
	if len(fakeClient.cookies) != len(want) {
		t.Fatalf("stored cookies = %v, want %v", fakeClient.cookies, want)
	}
	for i := range want {
		if fakeClient.cookies[i] != want[i] {
			t.Fatalf("stored cookies = %v, want %v", fakeClient.cookies, want)
		}
	}
}

func TestAuthManagerBasicFallbackToCSRFSuccess(t *testing.T) {
	sessionID := "session-secret"
	fakeClient := newScriptedAuthClient(t,
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CookieURL, status: 404},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CrumbURL, status: 429, body: "Too Many Requests response-body-secret"},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.ConsentURL, status: 200, body: hiddenConsentHTML("csrf-secret", sessionID)},
		scriptedAuthResponse{method: "POST", rawURL: endpoints.CollectConsentURL + "?sessionId=" + sessionID, status: 200},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CopyConsentURL + "?sessionId=" + sessionID, status: 204},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CrumbCSRFURL, status: 200, body: "csrf-crumb"},
	)
	auth := newScriptedAuthManager(fakeClient, StrategyBasic)

	crumb, err := auth.GetCrumb()
	if err != nil {
		t.Fatalf("GetCrumb returned error: %v", err)
	}
	if crumb != "csrf-crumb" {
		t.Fatalf("expected csrf crumb, got %q", crumb)
	}
	if auth.strategy != StrategyCSRF {
		t.Fatalf("expected strategy to switch to CSRF, got %v", auth.strategy)
	}
	fakeClient.assertDrained(t)
}

func TestAuthManagerCSRFFallbackToBasicSuccess(t *testing.T) {
	fakeClient := newScriptedAuthClient(t,
		scriptedAuthResponse{method: "GET", rawURL: endpoints.ConsentURL, status: 403, body: "response-body-secret"},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CookieURL, status: 404},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CrumbURL, status: 200, body: "basic-crumb"},
	)
	auth := newScriptedAuthManager(fakeClient, StrategyCSRF)

	crumb, err := auth.GetCrumb()
	if err != nil {
		t.Fatalf("GetCrumb returned error: %v", err)
	}
	if crumb != "basic-crumb" {
		t.Fatalf("expected basic crumb, got %q", crumb)
	}
	if auth.strategy != StrategyBasic {
		t.Fatalf("expected strategy to switch to Basic, got %v", auth.strategy)
	}
	fakeClient.assertDrained(t)
}

func TestAuthManagerCombinedFallbackErrorPreservesTypedCauses(t *testing.T) {
	secretValues := []string{
		"response-body-secret",
		"cookie-secret",
		"csrf-secret",
		"session-secret",
		"crumb-secret",
	}
	fakeClient := newScriptedAuthClient(t,
		scriptedAuthResponse{
			method:  "GET",
			rawURL:  endpoints.CookieURL,
			status:  404,
			headers: map[string]string{"Set-Cookie": "A3=cookie-secret; Path=/; Secure"},
		},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CrumbURL, status: 429, body: "Too Many Requests response-body-secret"},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.ConsentURL, status: 403, body: hiddenConsentHTML("csrf-secret", "session-secret")},
	)
	auth := newScriptedAuthManager(fakeClient, StrategyBasic)
	auth.crumb = "crumb-secret"
	auth.expiry = time.Now().Add(-time.Hour)

	crumb, err := auth.GetCrumb()
	if err == nil {
		t.Fatal("expected authentication error")
	}
	if crumb != "" {
		t.Fatalf("expected empty crumb on failure, got %q", crumb)
	}
	if !IsAuthError(err) {
		t.Fatalf("expected IsAuthError to match, got %v", err)
	}
	if !IsRateLimitError(err) {
		t.Fatalf("expected IsRateLimitError to match joined Basic 429, got %v", err)
	}
	errText := err.Error()
	for _, expected := range []string{"basic strategy", "csrf strategy", "HTTP 403"} {
		if !strings.Contains(errText, expected) {
			t.Fatalf("expected error to contain %q, got %q", expected, errText)
		}
	}
	for _, secret := range secretValues {
		if strings.Contains(errText, secret) {
			t.Fatalf("expected error to omit secret %q, got %q", secret, errText)
		}
	}
	fakeClient.assertDrained(t)
}

func TestAuthManagerBasicCrumb429IsRateLimit(t *testing.T) {
	fakeClient := newScriptedAuthClient(t,
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CookieURL, status: 404},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CrumbURL, status: 429, body: "response-body-secret"},
	)
	auth := newScriptedAuthManager(fakeClient, StrategyBasic)

	err := auth.fetchBasic()
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsRateLimitError(err) {
		t.Fatalf("expected rate-limit error, got %v", err)
	}
	if strings.Contains(err.Error(), "response-body-secret") {
		t.Fatalf("expected sanitized error, got %q", err.Error())
	}
	fakeClient.assertDrained(t)
}

func TestAuthManagerBasicRejectsCycleTLSTransportFailures(t *testing.T) {
	tests := []struct {
		name  string
		steps []scriptedAuthResponse
	}{
		{
			name: "cookie status zero",
			steps: []scriptedAuthResponse{
				{method: "GET", rawURL: endpoints.CookieURL, status: 0, body: "-> \ntransport-response-secret"},
			},
		},
		{
			name: "cookie synthetic status",
			steps: []scriptedAuthResponse{
				{method: "GET", rawURL: endpoints.CookieURL, status: 401, body: "Request returned a Syscall Error: transport-response-secret"},
			},
		},
		{
			name: "crumb status zero",
			steps: []scriptedAuthResponse{
				{method: "GET", rawURL: endpoints.CookieURL, status: 404},
				{method: "GET", rawURL: endpoints.CrumbURL, status: 0, body: "-> \ntransport-response-secret"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := newScriptedAuthClient(t, tt.steps...)
			auth := newScriptedAuthManager(fakeClient, StrategyBasic)

			err := auth.fetchBasic()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrNetwork) {
				t.Fatalf("expected network error, got %v", err)
			}
			if strings.Contains(err.Error(), "transport-response-secret") {
				t.Fatalf("expected sanitized error, got %q", err.Error())
			}
			if auth.crumb != "" {
				t.Fatalf("expected no crumb, got %q", auth.crumb)
			}
			fakeClient.assertDrained(t)
		})
	}
}

func TestValidateCrumbResponseRejectsInvalidBodies(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty", body: " \n\t"},
		{name: "html document", body: "<!DOCTYPE html><HTML><body>challenge</body></HTML>"},
		{name: "body fragment", body: "<body>challenge</body>"},
		{name: "head fragment", body: "<head><title>challenge</title></head>"},
		{name: "div fragment", body: `<div id="challenge">blocked</div>`},
		{name: "paragraph fragment", body: "<p>Access denied</p>"},
		{name: "span fragment mixed case", body: "<SpAn>challenge</sPaN>"},
		{name: "meta fragment", body: `<meta http-equiv="refresh" content="0">`},
		{name: "title fragment", body: "<title>Yahoo</title>"},
		{name: "comment fragment", body: "<!-- challenge -->"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCrumbResponse("csrf crumb", &Response{StatusCode: 200, Body: tt.body})
			if !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("expected invalid-response error, got %v", err)
			}
			if !strings.Contains(err.Error(), "csrf crumb") {
				t.Fatalf("expected stage in error, got %q", err.Error())
			}
		})
	}
}

func TestValidateCrumbResponseAllowsTagLikeText(t *testing.T) {
	for _, body := range []string{"crumb<bodyguard>text", "crumb<diversion>text", "<htmlx>not-html</htmlx>"} {
		t.Run(body, func(t *testing.T) {
			err := validateCrumbResponse("basic crumb", &Response{StatusCode: 200, Body: body})
			if err != nil {
				t.Fatalf("expected tag-like text to remain valid, got %v", err)
			}
		})
	}
}

func TestValidateAuthHTTPResponseIncludes404Status(t *testing.T) {
	err := validateAuthHTTPResponse("csrf copy consent", &Response{StatusCode: 404, Body: "response-body-secret"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not-found error, got %v", err)
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatalf("expected HTTP 404, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "response-body-secret") {
		t.Fatalf("expected sanitized error, got %q", err.Error())
	}
}

func TestAuthManagerCSRFConsentStatusCheckedBeforeParsing(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		assert     func(*testing.T, error)
	}{
		{
			name:       "rate limit",
			statusCode: 429,
			body:       "<html>missing inputs response-body-secret</html>",
			assert: func(t *testing.T, err error) {
				t.Helper()
				if !IsRateLimitError(err) {
					t.Fatalf("expected rate-limit error, got %v", err)
				}
			},
		},
		{
			name:       "transport failure",
			statusCode: 0,
			body:       "-> \ntransport-response-secret",
			assert: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, ErrNetwork) {
					t.Fatalf("expected network error, got %v", err)
				}
				if strings.Contains(err.Error(), "missing csrfToken or sessionId") {
					t.Fatalf("expected transport classification before token parsing, got %q", err.Error())
				}
			},
		},
		{
			name:       "forbidden",
			statusCode: 403,
			body:       "<html>missing inputs response-body-secret</html>",
			assert: func(t *testing.T, err error) {
				t.Helper()
				if !IsAuthError(err) {
					t.Fatalf("expected auth error, got %v", err)
				}
				if !strings.Contains(err.Error(), "HTTP 403") {
					t.Fatalf("expected HTTP 403, got %q", err.Error())
				}
				if strings.Contains(err.Error(), "CSRF tokens") {
					t.Fatalf("expected status error before token parsing, got %q", err.Error())
				}
			},
		},
		{
			name:       "successful consent without hidden inputs",
			statusCode: 200,
			body:       "<html>missing inputs response-body-secret</html>",
			assert: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, ErrInvalidResponse) {
					t.Fatalf("expected invalid-response error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := newScriptedAuthClient(t,
				scriptedAuthResponse{method: "GET", rawURL: endpoints.ConsentURL, status: tt.statusCode, body: tt.body},
			)
			auth := newScriptedAuthManager(fakeClient, StrategyCSRF)

			err := auth.fetchCSRF()
			if err == nil {
				t.Fatal("expected error")
			}
			tt.assert(t, err)
			if strings.Contains(err.Error(), "response-body-secret") {
				t.Fatalf("expected sanitized error, got %q", err.Error())
			}
			fakeClient.assertDrained(t)
		})
	}
}

func TestAuthManagerCSRFCollectAndCopyStatusErrorsIncludeStage(t *testing.T) {
	sessionID := "session-secret"
	tests := []struct {
		name          string
		collectStatus int
		copyStatus    int
		wantStage     string
	}{
		{
			name:          "collect consent",
			collectStatus: 500,
			copyStatus:    204,
			wantStage:     "csrf collect consent",
		},
		{
			name:          "copy consent",
			collectStatus: 200,
			copyStatus:    403,
			wantStage:     "csrf copy consent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := []scriptedAuthResponse{
				{method: "GET", rawURL: endpoints.ConsentURL, status: 200, body: hiddenConsentHTML("csrf-secret", sessionID)},
				{method: "POST", rawURL: endpoints.CollectConsentURL + "?sessionId=" + sessionID, status: tt.collectStatus, body: "response-body-secret"},
			}
			if tt.collectStatus < 400 {
				steps = append(steps, scriptedAuthResponse{
					method: "GET",
					rawURL: endpoints.CopyConsentURL + "?sessionId=" + sessionID,
					status: tt.copyStatus,
					body:   "response-body-secret",
				})
			}
			fakeClient := newScriptedAuthClient(t, steps...)
			auth := newScriptedAuthManager(fakeClient, StrategyCSRF)

			err := auth.fetchCSRF()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantStage) {
				t.Fatalf("expected stage %q in error, got %q", tt.wantStage, err.Error())
			}
			if strings.Contains(err.Error(), "response-body-secret") || strings.Contains(err.Error(), sessionID) {
				t.Fatalf("expected sanitized error, got %q", err.Error())
			}
			fakeClient.assertDrained(t)
		})
	}
}

func TestAuthManagerCSRFTransportErrorsAreSanitized(t *testing.T) {
	sessionID := "session-secret"
	csrfToken := "csrf-secret"
	transportSecret := "proxy-user:proxy-password"
	tests := []struct {
		name      string
		failStage string
		steps     []scriptedAuthResponse
	}{
		{
			name:      "collect consent",
			failStage: "csrf collect consent",
			steps: []scriptedAuthResponse{
				{method: "GET", rawURL: endpoints.ConsentURL, status: 200, body: hiddenConsentHTML(csrfToken, sessionID)},
				{method: "POST", rawURL: endpoints.CollectConsentURL + "?sessionId=" + sessionID, err: fmt.Errorf("POST %s via %s", sessionID, transportSecret)},
			},
		},
		{
			name:      "copy consent",
			failStage: "csrf copy consent",
			steps: []scriptedAuthResponse{
				{method: "GET", rawURL: endpoints.ConsentURL, status: 200, body: hiddenConsentHTML(csrfToken, sessionID)},
				{method: "POST", rawURL: endpoints.CollectConsentURL + "?sessionId=" + sessionID, status: 200},
				{method: "GET", rawURL: endpoints.CopyConsentURL + "?sessionId=" + sessionID, err: fmt.Errorf("GET %s via %s", sessionID, transportSecret)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := newScriptedAuthClient(t, tt.steps...)
			auth := newScriptedAuthManager(fakeClient, StrategyCSRF)

			err := auth.fetchCSRF()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrNetwork) {
				t.Fatalf("expected network error, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.failStage) {
				t.Fatalf("expected stage %q, got %q", tt.failStage, err.Error())
			}
			for _, secret := range []string{sessionID, csrfToken, transportSecret} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("expected error to omit secret %q, got %q", secret, err.Error())
				}
			}
			fakeClient.assertDrained(t)
		})
	}
}

func TestAuthManagerCSRFCrumbFailuresAreClassified(t *testing.T) {
	sessionID := "session-secret"
	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{name: "rate limit", status: 429, body: "response-body-secret", want: ErrRateLimit},
		{name: "transport", status: 0, body: "-> \ntransport-response-secret", want: ErrNetwork},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := newScriptedAuthClient(t,
				scriptedAuthResponse{method: "GET", rawURL: endpoints.ConsentURL, status: 200, body: hiddenConsentHTML("csrf-secret", sessionID)},
				scriptedAuthResponse{method: "POST", rawURL: endpoints.CollectConsentURL + "?sessionId=" + sessionID, status: 200},
				scriptedAuthResponse{method: "GET", rawURL: endpoints.CopyConsentURL + "?sessionId=" + sessionID, status: 204},
				scriptedAuthResponse{method: "GET", rawURL: endpoints.CrumbCSRFURL, status: tt.status, body: tt.body},
			)
			auth := newScriptedAuthManager(fakeClient, StrategyCSRF)

			err := auth.fetchCSRF()
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			if strings.Contains(err.Error(), tt.body) {
				t.Fatalf("expected sanitized error, got %q", err.Error())
			}
			if auth.crumb != "" {
				t.Fatalf("expected no crumb, got %q", auth.crumb)
			}
			fakeClient.assertDrained(t)
		})
	}
}

func TestAuthManagerCSRFSuccessUsesQuery2CrumbEndpoint(t *testing.T) {
	sessionID := "session-secret"
	fakeClient := newScriptedAuthClient(t,
		scriptedAuthResponse{method: "GET", rawURL: endpoints.ConsentURL, status: 200, body: hiddenConsentHTML("csrf-secret", sessionID)},
		scriptedAuthResponse{method: "POST", rawURL: endpoints.CollectConsentURL + "?sessionId=" + sessionID, status: 200},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CopyConsentURL + "?sessionId=" + sessionID, status: 204},
		scriptedAuthResponse{method: "GET", rawURL: endpoints.CrumbCSRFURL, status: 200, body: "csrf-crumb\n"},
	)
	auth := newScriptedAuthManager(fakeClient, StrategyCSRF)

	err := auth.fetchCSRF()
	if err != nil {
		t.Fatalf("fetchCSRF returned error: %v", err)
	}
	if auth.crumb != "csrf-crumb" {
		t.Fatalf("expected csrf crumb, got %q", auth.crumb)
	}
	if endpoints.CrumbCSRFURL != "https://query2.finance.yahoo.com/v1/test/getcrumb" {
		t.Fatalf("expected CSRF crumb endpoint to use query2, got %s", endpoints.CrumbCSRFURL)
	}
	fakeClient.assertDrained(t)
}

func TestAuthManagerSwitchStrategy(t *testing.T) {
	client, _ := New()
	auth := NewAuthManager(client)

	if auth.strategy != StrategyBasic {
		t.Error("Initial strategy should be StrategyBasic")
	}

	auth.SwitchStrategy()
	if auth.strategy != StrategyCSRF {
		t.Error("Strategy should be StrategyCSRF after switch")
	}

	auth.SwitchStrategy()
	if auth.strategy != StrategyBasic {
		t.Error("Strategy should be StrategyBasic after second switch")
	}
}

func TestAuthManagerReset(t *testing.T) {
	client, _ := New()
	auth := NewAuthManager(client)

	// Set some values
	auth.crumb = "test-crumb"
	auth.cookie = "test-cookie"
	auth.user = map[string]interface{}{"guid": "abc"}

	auth.Reset()

	if auth.crumb != "" {
		t.Error("Crumb should be empty after reset")
	}
	if auth.cookie != "" {
		t.Error("Cookie should be empty after reset")
	}
	if auth.user != nil {
		t.Error("User should be empty after reset")
	}
}

func TestAuthManagerSetLoginCookies(t *testing.T) {
	client, _ := New()
	auth := NewAuthManager(client)
	auth.crumb = "anonymous-crumb"
	auth.expiry = time.Now().Add(time.Hour)

	auth.SetLoginCookies("cookie-t", "cookie-y")

	cookie := client.GetCookie()
	if cookie != "T=cookie-t; Y=cookie-y" {
		t.Errorf("Expected login cookies, got %q", cookie)
	}
	if auth.crumb != "" {
		t.Error("Expected login cookies to invalidate cached crumb")
	}
	if !auth.expiry.IsZero() {
		t.Error("Expected login cookies to clear crumb expiry")
	}
}

func TestAuthManagerSetLoginCookiesPreservesExistingCookies(t *testing.T) {
	client, _ := New()
	auth := NewAuthManager(client)
	client.SetCookie("A3=crumb-cookie")

	auth.SetLoginCookies("cookie-t", "cookie-y")

	cookie := client.GetCookie()
	expected := "A3=crumb-cookie; T=cookie-t; Y=cookie-y"
	if cookie != expected {
		t.Errorf("Expected merged login cookies, got %q", cookie)
	}
}

func TestAuthManagerCheckLoginSubscriptions(t *testing.T) {
	client, _ := New()
	auth := NewAuthManager(client)

	loggedIn, err := auth.checkLoginWithGetter(func(rawURL string, params url.Values) (*Response, error) {
		if rawURL != "https://query1.finance.yahoo.com/ws/obi-integration/v1/subscriptions" {
			t.Fatalf("Unexpected subscriptions URL %q", rawURL)
		}
		if params != nil {
			t.Fatalf("Expected nil params, got %v", params)
		}
		return &Response{StatusCode: 200, Body: `{"result":{"guid":"abc123","subscriptionView":[]}}`}, nil
	})
	if err != nil {
		t.Fatalf("CheckLogin returned error: %v", err)
	}
	if !loggedIn {
		t.Fatal("Expected subscriptions result with guid to be logged in")
	}
	user := auth.User()
	if user["guid"] != "abc123" {
		t.Errorf("Expected cached guid abc123, got %v", user["guid"])
	}
}

func TestAuthManagerCheckLoginSubscriptionsLoggedOut(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "unauthorized", statusCode: 401, body: `{}`},
		{name: "forbidden", statusCode: 403, body: `{}`},
		{name: "missing guid", statusCode: 200, body: `{"result":{"subscriptionView":[]}}`},
		{name: "missing result", statusCode: 200, body: `{}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := New()
			if err != nil {
				t.Fatalf("New returned error: %v", err)
			}
			auth := NewAuthManager(client)
			auth.user = map[string]interface{}{"guid": "stale"}

			loggedIn, err := auth.checkLoginWithGetter(func(_ string, _ url.Values) (*Response, error) {
				return &Response{StatusCode: tc.statusCode, Body: tc.body}, nil
			})
			if err != nil {
				t.Fatalf("CheckLogin returned error: %v", err)
			}
			if loggedIn {
				t.Fatal("Expected logged out state")
			}
			if user := auth.User(); user != nil {
				t.Fatalf("Expected stale user cache to be cleared, got %v", user)
			}
		})
	}
}

func TestAuthManagerEntitlementTransportFailuresAreNetworkErrors(t *testing.T) {
	transportSecret := "proxy-user:proxy-password"
	tests := []struct {
		name   string
		resp   *Response
		err    error
		secret string
	}{
		{
			name:   "direct transport error",
			err:    fmt.Errorf("GET subscriptions via %s", transportSecret),
			secret: transportSecret,
		},
		{
			name:   "status zero",
			resp:   &Response{StatusCode: 0, Body: "-> \ntransport-response-secret"},
			secret: "transport-response-secret",
		},
		{
			name:   "synthetic unauthorized",
			resp:   &Response{StatusCode: 401, Body: "Request returned a Syscall Error: transport-response-secret"},
			secret: "transport-response-secret",
		},
		{
			name:   "synthetic forbidden",
			resp:   &Response{StatusCode: 403, Body: "Request returned a Syscall Error: transport-response-secret"},
			secret: "transport-response-secret",
		},
		{
			name:   "synthetic timeout",
			resp:   &Response{StatusCode: 408, Body: "Request returned a Syscall Error: transport-response-secret"},
			secret: "transport-response-secret",
		},
		{
			name:   "synthetic DNS failure",
			resp:   &Response{StatusCode: 421, Body: "Request returned a Syscall Error: transport-response-secret"},
			secret: "transport-response-secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := New()
			auth := NewAuthManager(client)
			entitlement, loggedIn, err := auth.fetchEntitlementWithGetter(func(_ string, _ url.Values) (*Response, error) {
				return tt.resp, tt.err
			})

			if !errors.Is(err, ErrNetwork) {
				t.Fatalf("expected network error, got %v", err)
			}
			if loggedIn || entitlement != nil {
				t.Fatalf("expected no entitlement on transport failure, got loggedIn=%v entitlement=%v", loggedIn, entitlement)
			}
			if strings.Contains(err.Error(), tt.secret) {
				t.Fatalf("expected sanitized error, got %q", err.Error())
			}
		})
	}
}

func TestAuthManagerCheckLoginTransportFailurePreservesCachedUser(t *testing.T) {
	client, _ := New()
	auth := NewAuthManager(client)
	auth.user = map[string]interface{}{"guid": "cached-guid"}

	loggedIn, err := auth.checkLoginWithGetter(func(_ string, _ url.Values) (*Response, error) {
		return &Response{StatusCode: 401, Body: "Request returned a Syscall Error: transport-response-secret"}, nil
	})
	if !errors.Is(err, ErrNetwork) {
		t.Fatalf("expected network error, got %v", err)
	}
	if loggedIn {
		t.Fatal("expected unsuccessful login check")
	}
	if user := auth.User(); user["guid"] != "cached-guid" {
		t.Fatalf("expected cached user to remain unchanged, got %v", user)
	}
}

func TestAuthManagerSubscriptionTierTransportFailure(t *testing.T) {
	client, _ := New()
	auth := NewAuthManager(client)

	tier, err := auth.subscriptionTierWithGetter(func(_ string, _ url.Values) (*Response, error) {
		return &Response{StatusCode: 0, Body: "-> \ntransport-response-secret"}, nil
	})
	if !errors.Is(err, ErrNetwork) {
		t.Fatalf("expected network error, got %v", err)
	}
	if tier != "" {
		t.Fatalf("expected empty tier on error, got %q", tier)
	}
}

func TestAuthManagerSubscriptionTier(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "gold",
			body: `{"result":{"guid":"abc123","subscriptionView":[{"action":"ACTIVE","tier":6}]}}`,
			want: "gold",
		},
		{
			name: "silver",
			body: `{"result":{"guid":"abc123","subscriptionView":[{"action":"ACTIVE","tier":5}]}}`,
			want: "silver",
		},
		{
			name: "bronze",
			body: `{"result":{"guid":"abc123","subscriptionView":[{"action":"ACTIVE","tier":3}]}}`,
			want: "bronze",
		},
		{
			name: "premium unknown active tier",
			body: `{"result":{"guid":"abc123","subscriptionView":[{"action":"ACTIVE","tier":4}]}}`,
			want: "premium",
		},
		{
			name: "free with no active subscription",
			body: `{"result":{"guid":"abc123","subscriptionView":[{"action":"EXPIRED","tier":6}]}}`,
			want: "free",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := New()
			auth := NewAuthManager(client)
			got, err := auth.subscriptionTierWithGetter(func(_ string, _ url.Values) (*Response, error) {
				return &Response{StatusCode: 200, Body: tt.body}, nil
			})
			if err != nil {
				t.Fatalf("SubscriptionTier returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Expected tier %q, got %q", tt.want, got)
			}
		})
	}
}

func TestAuthManagerSubscriptionTierLoggedOut(t *testing.T) {
	client, _ := New()
	auth := NewAuthManager(client)

	got, err := auth.subscriptionTierWithGetter(func(_ string, _ url.Values) (*Response, error) {
		return &Response{StatusCode: 401, Body: `{}`}, nil
	})
	if err != nil {
		t.Fatalf("SubscriptionTier returned error: %v", err)
	}
	if got != "" {
		t.Fatalf("Expected empty tier when logged out, got %q", got)
	}
}

func TestParseLoginUser(t *testing.T) {
	html := `<html><script id="nimbus-benji-config">{"i13n":{"user":{"guid":"abc123","login":"user@example.com"}}}</script></html>`

	user, ok, err := parseLoginUser(html)
	if err != nil {
		t.Fatalf("parseLoginUser returned error: %v", err)
	}
	if !ok {
		t.Fatal("Expected login user")
	}
	if user["guid"] != "abc123" {
		t.Errorf("Expected guid abc123, got %v", user["guid"])
	}
}

func TestParseLoginUserMissing(t *testing.T) {
	user, ok, err := parseLoginUser(`<html></html>`)
	if err != nil {
		t.Fatalf("parseLoginUser returned error: %v", err)
	}
	if ok {
		t.Fatal("Expected missing login user")
	}
	if user != nil {
		t.Errorf("Expected nil user, got %v", user)
	}
}
