package config

import (
	"testing"
	"time"
)

func TestNewDefault(t *testing.T) {
	cfg := NewDefault()

	if cfg.Timeout != DefaultTimeout {
		t.Errorf("Timeout should be %v, got %v", DefaultTimeout, cfg.Timeout)
	}
	if cfg.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries should be %d, got %d", DefaultMaxRetries, cfg.MaxRetries)
	}
	if cfg.JA3 != DefaultJA3 {
		t.Errorf("JA3 should be default")
	}
	if cfg.CacheEnabled {
		t.Error("Cache should be disabled by default")
	}
	if cfg.Debug {
		t.Error("Debug should be disabled by default")
	}
}

func TestConfigSetters(t *testing.T) {
	cfg := NewDefault()

	cfg.SetTimeout(60 * time.Second)
	if cfg.GetTimeout() != 60*time.Second {
		t.Errorf("Timeout should be 60s")
	}

	cfg.SetUserAgent("test-agent")
	if cfg.GetUserAgent() != "test-agent" {
		t.Errorf("UserAgent should be 'test-agent'")
	}

	cfg.SetJA3("custom-ja3")
	if cfg.GetJA3() != "custom-ja3" {
		t.Errorf("JA3 should be 'custom-ja3'")
	}

	cfg.SetProxy("http://proxy:8080")
	if cfg.GetProxyURL() != "http://proxy:8080" {
		t.Errorf("ProxyURL should be set")
	}

	cfg.SetMaxRetries(5)
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries should be 5")
	}

	cfg.SetDebug(true)
	if !cfg.IsDebug() {
		t.Errorf("Debug should be true")
	}
}

func TestConfigCache(t *testing.T) {
	cfg := NewDefault()

	if cfg.IsCacheEnabled() {
		t.Error("Cache should be disabled by default")
	}

	cfg.EnableCache(10 * time.Minute)
	if !cfg.IsCacheEnabled() {
		t.Error("Cache should be enabled")
	}
	if cfg.CacheTTL != 10*time.Minute {
		t.Errorf("CacheTTL should be 10 minutes")
	}

	cfg.DisableCache()
	if cfg.IsCacheEnabled() {
		t.Error("Cache should be disabled")
	}
}

func TestConfigChaining(t *testing.T) {
	cfg := NewDefault().
		SetTimeout(60 * time.Second).
		SetUserAgent("chained-agent").
		SetMaxRetries(5).
		SetDebug(true)

	if cfg.GetTimeout() != 60*time.Second {
		t.Error("Chained timeout should work")
	}
	if cfg.GetUserAgent() != "chained-agent" {
		t.Error("Chained user agent should work")
	}
	if cfg.MaxRetries != 5 {
		t.Error("Chained max retries should work")
	}
	if !cfg.IsDebug() {
		t.Error("Chained debug should work")
	}
}

func TestConfigClone(t *testing.T) {
	cfg := NewDefault()
	cfg.SetTimeout(45 * time.Second)
	cfg.SetDebug(true)

	cloned := cfg.Clone()

	if cloned.GetTimeout() != 45*time.Second {
		t.Error("Cloned config should have same timeout")
	}
	if !cloned.IsDebug() {
		t.Error("Cloned config should have same debug setting")
	}

	// Modify original, cloned should not change
	cfg.SetTimeout(90 * time.Second)
	if cloned.GetTimeout() != 45*time.Second {
		t.Error("Cloned config should be independent")
	}
}

func TestGlobalConfig(t *testing.T) {
	Reset() // Reset before test

	cfg := Get()
	if cfg == nil {
		t.Fatal("Global config should not be nil")
	}

	// Should return same instance
	cfg2 := Get()
	if cfg != cfg2 {
		t.Error("Get() should return same instance")
	}

	// Modify global config
	cfg.SetDebug(true)
	if !Get().IsDebug() {
		t.Error("Global config modification should persist")
	}

	Reset() // Clean up
}
