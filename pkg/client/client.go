// Package client provides HTTP client functionality for Yahoo Finance API.
package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/Danny-Dasilva/CycleTLS/cycletls"
)

// Client is the HTTP client for Yahoo Finance API with TLS fingerprint spoofing.
type Client struct {
	cycleTLS    cycletls.CycleTLS
	initOnce    sync.Once
	mu          sync.RWMutex
	closed      bool
	initialized bool

	// Configuration
	timeout   int
	ja3       string
	userAgent string

	// Cookie storage for authentication
	cookie string
}

// Chrome JA3 fingerprint for TLS spoofing
const defaultJA3 = "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0"

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithTimeout sets the request timeout in seconds.
func WithTimeout(timeout int) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithJA3 sets a custom JA3 fingerprint.
func WithJA3(ja3 string) ClientOption {
	return func(c *Client) {
		c.ja3 = ja3
	}
}

// WithUserAgent sets a custom User-Agent.
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// New creates a new Client with optional configuration.
// The underlying CycleTLS client is lazily initialized on first request.
func New(opts ...ClientOption) (*Client, error) {
	c := &Client{
		timeout:   30,
		ja3:       defaultJA3,
		userAgent: RandomUserAgent(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// init initializes the CycleTLS client lazily.
func (c *Client) init() {
	c.initOnce.Do(func() {
		c.cycleTLS = cycletls.Init()
		c.initialized = true
	})
}

// Response represents an HTTP response.
type Response struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// Get performs an HTTP GET request.
func (c *Client) Get(rawURL string, params url.Values) (*Response, error) {
	c.init()

	c.mu.RLock()
	defer c.mu.RUnlock()

	if params != nil && len(params) > 0 {
		rawURL = fmt.Sprintf("%s?%s", rawURL, params.Encode())
	}

	headers := map[string]string{
		"Accept":          "application/json,text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.5",
		"Connection":      "keep-alive",
	}

	// Add cookie if available
	if c.cookie != "" {
		headers["Cookie"] = c.cookie
	}

	resp, err := c.cycleTLS.Do(rawURL, cycletls.Options{
		Timeout:   c.timeout,
		Ja3:       c.ja3,
		UserAgent: c.userAgent,
		Headers:   headers,
	}, "GET")
	if err != nil {
		return nil, fmt.Errorf("GET request failed: %w", err)
	}

	return &Response{
		StatusCode: resp.Status,
		Body:       resp.Body,
		Headers:    resp.Headers,
	}, nil
}

// SetCookie sets the cookie for subsequent requests.
func (c *Client) SetCookie(cookie string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cookie = cookie
}

// GetCookie returns the current cookie.
func (c *Client) GetCookie() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cookie
}

// GetJSON performs an HTTP GET request and unmarshals the JSON response.
func (c *Client) GetJSON(rawURL string, params url.Values, v interface{}) error {
	resp, err := c.Get(rawURL, params)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Body)
	}

	if err := json.Unmarshal([]byte(resp.Body), v); err != nil {
		return fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	return nil
}

// Post performs an HTTP POST request with form data.
func (c *Client) Post(rawURL string, params url.Values, body map[string]string) (*Response, error) {
	c.init()

	c.mu.RLock()
	defer c.mu.RUnlock()

	if params != nil && len(params) > 0 {
		rawURL = fmt.Sprintf("%s?%s", rawURL, params.Encode())
	}

	resp, err := c.cycleTLS.Do(rawURL, cycletls.Options{
		Timeout:   c.timeout,
		Ja3:       c.ja3,
		UserAgent: c.userAgent,
		Body:      mapToFormData(body),
		Headers: map[string]string{
			"Accept":          "application/json,text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.5",
			"Content-Type":    "application/x-www-form-urlencoded",
			"Connection":      "keep-alive",
		},
	}, "POST")
	if err != nil {
		return nil, fmt.Errorf("POST request failed: %w", err)
	}

	return &Response{
		StatusCode: resp.Status,
		Body:       resp.Body,
		Headers:    resp.Headers,
	}, nil
}

// PostJSON performs an HTTP POST request with JSON body.
func (c *Client) PostJSON(rawURL string, params url.Values, body []byte) (*Response, error) {
	c.init()

	c.mu.RLock()
	defer c.mu.RUnlock()

	if params != nil && len(params) > 0 {
		rawURL = fmt.Sprintf("%s?%s", rawURL, params.Encode())
	}

	headers := map[string]string{
		"Accept":          "application/json",
		"Accept-Language": "en-US,en;q=0.5",
		"Content-Type":    "application/json",
		"Connection":      "keep-alive",
	}

	// Add cookie if available
	if c.cookie != "" {
		headers["Cookie"] = c.cookie
	}

	resp, err := c.cycleTLS.Do(rawURL, cycletls.Options{
		Timeout:   c.timeout,
		Ja3:       c.ja3,
		UserAgent: c.userAgent,
		Body:      string(body),
		Headers:   headers,
	}, "POST")
	if err != nil {
		return nil, fmt.Errorf("POST request failed: %w", err)
	}

	return &Response{
		StatusCode: resp.Status,
		Body:       resp.Body,
		Headers:    resp.Headers,
	}, nil
}

// Close closes the CycleTLS client.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Only close if initialized and not already closed
	if c.initialized && !c.closed {
		// Recover from panic in case CycleTLS has internal nil channel issue
		defer func() {
			if r := recover(); r != nil {
				// Silently ignore panic from CycleTLS close
			}
		}()
		c.cycleTLS.Close()
		c.closed = true
	}
}

// mapToFormData converts a map to URL-encoded form data.
func mapToFormData(data map[string]string) string {
	values := url.Values{}
	for k, v := range data {
		values.Set(k, v)
	}
	return values.Encode()
}
