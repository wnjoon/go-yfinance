package client

import (
	"testing"
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

	auth.SetLoginCookies("cookie-t", "cookie-y")

	cookie := client.GetCookie()
	if cookie != "T=cookie-t; Y=cookie-y" {
		t.Errorf("Expected login cookies, got %q", cookie)
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
