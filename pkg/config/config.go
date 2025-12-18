// Package config provides configuration management for go-yfinance.
package config

import (
	"sync"
	"time"
)

// Config holds the global configuration for go-yfinance.
type Config struct {
	mu sync.RWMutex

	// HTTP Client settings
	Timeout   time.Duration
	UserAgent string
	JA3       string

	// Proxy settings
	ProxyURL string

	// Rate limiting
	MaxRetries    int
	RetryDelay    time.Duration
	MaxConcurrent int

	// Cache settings
	CacheEnabled bool
	CacheTTL     time.Duration

	// Debug settings
	Debug bool
}

// Default configuration values
const (
	DefaultTimeout       = 30 * time.Second
	DefaultMaxRetries    = 3
	DefaultRetryDelay    = 1 * time.Second
	DefaultMaxConcurrent = 10
	DefaultCacheTTL      = 5 * time.Minute
)

// Default JA3 fingerprint (Chrome)
const DefaultJA3 = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0"

var (
	globalConfig *Config
	once         sync.Once
)

// Get returns the global configuration instance.
func Get() *Config {
	once.Do(func() {
		globalConfig = NewDefault()
	})
	return globalConfig
}

// NewDefault creates a new Config with default values.
func NewDefault() *Config {
	return &Config{
		Timeout:       DefaultTimeout,
		UserAgent:     "", // Will use random User-Agent if empty
		JA3:           DefaultJA3,
		ProxyURL:      "",
		MaxRetries:    DefaultMaxRetries,
		RetryDelay:    DefaultRetryDelay,
		MaxConcurrent: DefaultMaxConcurrent,
		CacheEnabled:  false,
		CacheTTL:      DefaultCacheTTL,
		Debug:         false,
	}
}

// SetTimeout sets the HTTP request timeout.
func (c *Config) SetTimeout(d time.Duration) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Timeout = d
	return c
}

// SetUserAgent sets the User-Agent string.
func (c *Config) SetUserAgent(ua string) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.UserAgent = ua
	return c
}

// SetJA3 sets the JA3 TLS fingerprint.
func (c *Config) SetJA3(ja3 string) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.JA3 = ja3
	return c
}

// SetProxy sets the proxy URL.
func (c *Config) SetProxy(proxyURL string) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ProxyURL = proxyURL
	return c
}

// SetMaxRetries sets the maximum number of retries.
func (c *Config) SetMaxRetries(n int) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.MaxRetries = n
	return c
}

// SetRetryDelay sets the delay between retries.
func (c *Config) SetRetryDelay(d time.Duration) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RetryDelay = d
	return c
}

// SetMaxConcurrent sets the maximum concurrent requests.
func (c *Config) SetMaxConcurrent(n int) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.MaxConcurrent = n
	return c
}

// EnableCache enables response caching.
func (c *Config) EnableCache(ttl time.Duration) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CacheEnabled = true
	c.CacheTTL = ttl
	return c
}

// DisableCache disables response caching.
func (c *Config) DisableCache() *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CacheEnabled = false
	return c
}

// SetDebug enables or disables debug mode.
func (c *Config) SetDebug(debug bool) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Debug = debug
	return c
}

// GetTimeout returns the timeout value.
func (c *Config) GetTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Timeout
}

// GetUserAgent returns the User-Agent string.
func (c *Config) GetUserAgent() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.UserAgent
}

// GetJA3 returns the JA3 fingerprint.
func (c *Config) GetJA3() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.JA3
}

// GetProxyURL returns the proxy URL.
func (c *Config) GetProxyURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ProxyURL
}

// IsDebug returns whether debug mode is enabled.
func (c *Config) IsDebug() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Debug
}

// IsCacheEnabled returns whether caching is enabled.
func (c *Config) IsCacheEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CacheEnabled
}

// Clone creates a copy of the configuration.
func (c *Config) Clone() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &Config{
		Timeout:       c.Timeout,
		UserAgent:     c.UserAgent,
		JA3:           c.JA3,
		ProxyURL:      c.ProxyURL,
		MaxRetries:    c.MaxRetries,
		RetryDelay:    c.RetryDelay,
		MaxConcurrent: c.MaxConcurrent,
		CacheEnabled:  c.CacheEnabled,
		CacheTTL:      c.CacheTTL,
		Debug:         c.Debug,
	}
}

// Reset resets the global configuration to defaults.
func Reset() {
	globalConfig = NewDefault()
}
