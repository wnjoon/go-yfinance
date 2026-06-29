package client

import (
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/config"
)

func TestNewClient(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	c, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Note: CycleTLS is lazily initialized on first request
	// Don't call c.Close() here since it would panic on uninitialized client

	if c.timeout != 30 {
		t.Errorf("Default timeout should be 30, got %d", c.timeout)
	}
	if c.ja3 != defaultJA3 {
		t.Error("Default JA3 should be set")
	}
	if c.userAgent == "" {
		t.Error("User-Agent should not be empty")
	}
}

func TestClientOptions(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	customJA3 := "custom-ja3"
	customUA := "custom-user-agent"
	customProxy := "http://proxy.example:8080"

	c, err := New(
		WithTimeout(60),
		WithJA3(customJA3),
		WithUserAgent(customUA),
		WithProxy(customProxy),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if c.timeout != 60 {
		t.Errorf("Timeout should be 60, got %d", c.timeout)
	}
	if c.ja3 != customJA3 {
		t.Errorf("JA3 should be %s, got %s", customJA3, c.ja3)
	}
	if c.userAgent != customUA {
		t.Errorf("User-Agent should be %s, got %s", customUA, c.userAgent)
	}
	if c.proxy != customProxy {
		t.Errorf("Proxy should be %s, got %s", customProxy, c.proxy)
	}
}

func TestNewClientUsesGlobalConfig(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	config.Get().
		SetTimeout(45 * time.Second).
		SetJA3("configured-ja3").
		SetUserAgent("configured-user-agent").
		SetProxy(" http://configured-proxy.example:8080 ")

	c, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if c.timeout != 45 {
		t.Errorf("Timeout should be 45, got %d", c.timeout)
	}
	if c.ja3 != "configured-ja3" {
		t.Errorf("JA3 should come from config, got %s", c.ja3)
	}
	if c.userAgent != "configured-user-agent" {
		t.Errorf("User-Agent should come from config, got %s", c.userAgent)
	}
	if c.proxy != "http://configured-proxy.example:8080" {
		t.Errorf("Proxy should be trimmed from config, got %q", c.proxy)
	}
}

func TestClientCookieMerge(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	c, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	c.SetCookie("A3=crumb-cookie")
	c.SetCookies(map[string]string{
		"T": "cookie-t",
		"Y": "cookie-y",
	})

	got := c.GetCookie()
	want := "A3=crumb-cookie; T=cookie-t; Y=cookie-y"
	if got != want {
		t.Errorf("Expected %q, got %q", want, got)
	}
}

func TestRandomUserAgent(t *testing.T) {
	ua := RandomUserAgent()
	if ua == "" {
		t.Error("RandomUserAgent should not return empty string")
	}

	// Verify it's from our list
	found := false
	for _, agent := range UserAgents {
		if agent == ua {
			found = true
			break
		}
	}
	if !found {
		t.Error("RandomUserAgent should return a value from UserAgents list")
	}
}
