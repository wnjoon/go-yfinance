// Package config provides configuration management for go-yfinance.
//
// # Overview
//
// The config package provides a centralized way to configure go-yfinance behavior.
// It supports both global configuration via [Get] and per-instance configuration
// via [NewDefault].
//
// # Global Configuration
//
//	import "github.com/wnjoon/go-yfinance/pkg/config"
//
//	// Get global config and modify settings
//	config.Get().
//	    SetTimeout(60 * time.Second).
//	    SetProxy("http://proxy:8080").
//	    SetDebug(true)
//
// # Available Settings
//
// HTTP Client:
//   - Timeout: Request timeout duration
//   - UserAgent: Custom User-Agent string
//   - JA3: TLS fingerprint for spoofing
//   - ProxyURL: HTTP/HTTPS proxy URL
//
// Rate Limiting:
//   - MaxRetries: Maximum retry attempts
//   - RetryDelay: Delay between retries
//   - MaxConcurrent: Maximum concurrent requests
//
// Caching:
//   - CacheEnabled: Enable/disable response caching
//   - CacheTTL: Cache time-to-live duration
//
// Debug:
//   - Debug: Enable debug logging
//
// # Thread Safety
//
// All Config methods are safe for concurrent use.
//
// # Reset
//
// Use [Reset] to restore global configuration to defaults:
//
//	config.Reset()
package config
