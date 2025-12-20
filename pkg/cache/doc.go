// Package cache provides an in-memory caching layer for go-yfinance.
//
// # Overview
//
// The cache package provides a thread-safe, TTL-based in-memory cache
// for storing frequently accessed data like timezone mappings and
// ticker information.
//
// # Basic Usage
//
//	c := cache.New(cache.WithTTL(5 * time.Minute))
//	c.Set("AAPL:tz", "America/New_York")
//	if tz, ok := c.Get("AAPL:tz"); ok {
//	    fmt.Println(tz)
//	}
//
// # Global Cache
//
// A global cache instance is available for convenience:
//
//	cache.SetGlobal("key", "value")
//	value, ok := cache.GetGlobal("key")
//
// # Configuration Options
//
//   - [WithTTL]: Set custom TTL for cache entries (default: 5 minutes)
//
// # Automatic Cleanup
//
// The cache automatically removes expired entries every 10 minutes
// to prevent memory leaks.
//
// # Thread Safety
//
// All cache operations are thread-safe and can be used from multiple goroutines.
package cache
