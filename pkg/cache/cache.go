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
// # Thread Safety
//
// All cache operations are thread-safe and can be used from multiple goroutines.
package cache

import (
	"sync"
	"time"
)

// Default settings
const (
	// DefaultTTL is the default time-to-live for cache entries.
	DefaultTTL = 5 * time.Minute

	// DefaultCleanupInterval is the default interval for cleaning expired entries.
	DefaultCleanupInterval = 10 * time.Minute
)

// entry represents a single cache entry with expiration time.
type entry struct {
	value      interface{}
	expiration time.Time
}

// isExpired returns true if the entry has expired.
func (e *entry) isExpired() bool {
	return time.Now().After(e.expiration)
}

// Cache is a thread-safe, TTL-based in-memory cache.
type Cache struct {
	mu       sync.RWMutex
	items    map[string]*entry
	ttl      time.Duration
	stopChan chan struct{}
}

// Option is a function that configures a Cache.
type Option func(*Cache)

// WithTTL sets the default TTL for cache entries.
func WithTTL(ttl time.Duration) Option {
	return func(c *Cache) {
		c.ttl = ttl
	}
}

// New creates a new Cache with the given options.
func New(opts ...Option) *Cache {
	c := &Cache{
		items:    make(map[string]*entry),
		ttl:      DefaultTTL,
		stopChan: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Start cleanup goroutine
	go c.cleanupLoop()

	return c
}

// Set stores a value in the cache with the default TTL.
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in the cache with a custom TTL.
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &entry{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

// Get retrieves a value from the cache.
// Returns the value and true if found and not expired, otherwise nil and false.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	if item.isExpired() {
		return nil, false
	}

	return item.value, true
}

// GetString retrieves a string value from the cache.
// Returns the value and true if found, not expired, and is a string.
func (c *Cache) GetString(key string) (string, bool) {
	value, ok := c.Get(key)
	if !ok {
		return "", false
	}

	s, ok := value.(string)
	return s, ok
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all entries from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*entry)
}

// Len returns the number of items in the cache (including expired ones).
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Close stops the cleanup goroutine and releases resources.
func (c *Cache) Close() {
	close(c.stopChan)
}

// cleanupLoop periodically removes expired entries.
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(DefaultCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// cleanup removes all expired entries.
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if item.isExpired() {
			delete(c.items, key)
		}
	}
}

// Global cache instance
var (
	globalCache     *Cache
	globalCacheOnce sync.Once
)

// getGlobalCache returns the global cache instance, creating it if necessary.
func getGlobalCache() *Cache {
	globalCacheOnce.Do(func() {
		globalCache = New()
	})
	return globalCache
}

// SetGlobal stores a value in the global cache.
func SetGlobal(key string, value interface{}) {
	getGlobalCache().Set(key, value)
}

// SetGlobalWithTTL stores a value in the global cache with a custom TTL.
func SetGlobalWithTTL(key string, value interface{}, ttl time.Duration) {
	getGlobalCache().SetWithTTL(key, value, ttl)
}

// GetGlobal retrieves a value from the global cache.
func GetGlobal(key string) (interface{}, bool) {
	return getGlobalCache().Get(key)
}

// GetGlobalString retrieves a string value from the global cache.
func GetGlobalString(key string) (string, bool) {
	return getGlobalCache().GetString(key)
}

// DeleteGlobal removes a key from the global cache.
func DeleteGlobal(key string) {
	getGlobalCache().Delete(key)
}

// ClearGlobal removes all entries from the global cache.
func ClearGlobal() {
	getGlobalCache().Clear()
}
