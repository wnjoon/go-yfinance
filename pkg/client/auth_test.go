package client

import (
	"net/url"
	"testing"
	"time"
)

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
	client, _ := New()
	auth := NewAuthManager(client)
	auth.user = map[string]interface{}{"guid": "stale"}

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
