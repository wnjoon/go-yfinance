package client

import (
	"testing"
)

func TestNewClient(t *testing.T) {
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
	customJA3 := "custom-ja3"
	customUA := "custom-user-agent"

	c, err := New(
		WithTimeout(60),
		WithJA3(customJA3),
		WithUserAgent(customUA),
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
