// Package client provides HTTP client functionality for accessing Yahoo Finance API.
//
// # Overview
//
// The client package handles all HTTP communication with Yahoo Finance servers,
// including TLS fingerprint spoofing using CycleTLS to avoid detection and blocking.
//
// # Authentication
//
// Yahoo Finance API requires cookie/crumb authentication. This is handled automatically
// by the [AuthManager] which supports two strategies:
//
//   - Basic: Uses fc.yahoo.com for cookie acquisition (default)
//   - CSRF: Uses guce.yahoo.com consent flow (for EU users)
//
// The AuthManager automatically falls back to the alternate strategy if one fails.
//
// # Usage
//
//	c, err := client.New(
//	    client.WithTimeout(30),
//	    client.WithUserAgent("Custom UA"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer c.Close()
//
//	resp, err := c.Get("https://query2.finance.yahoo.com/v8/finance/chart/AAPL", nil)
//
// # Error Handling
//
// The package provides typed errors via [YFError] for easy error handling:
//
//	if client.IsRateLimitError(err) {
//	    // Handle rate limiting
//	}
//
// See [ErrorCode] for all available error types.
package client
