# go-yfinance v1.6.1

**Go Maintenance Release — Python yfinance v1.6.0 Baseline**

This is a Go-specific patch release built on the Python yfinance v1.6.0
parity baseline. It does not claim Python yfinance v1.6.1 parity. The release
packages authentication hardening from go-yfinance PR #13 and independently
reworks the safety fixes proposed in contributor PR #11 for the current code.

## Changes

### Authentication

- Authentication fallback now preserves labeled Basic and CSRF failures,
  classifies rate limits and invalid responses, sanitizes transport failures,
  and uses the correct query1/query2 crumb endpoint for each strategy.
- Entitlement checks distinguish logged-out responses from network failures
  without clearing cached user state on an inconclusive transport error.
- All cookies parsed by CycleTLS are now preserved across the client response
  boundary. The compatibility header path recognizes CycleTLS's exact `/,/`
  delimiter without splitting commas inside cookie `Expires` attributes.

### Live data safety

- The manual protobuf decoder rejects overflowing varints and validates
  length-delimited fields as `uint64` before converting lengths to `int`,
  preventing malformed WebSocket input from causing slice-bound panics.
- WebSocket shutdown is final and idempotent. Normal `Close` stops listeners
  without reporting an error, cancels reconnect delays, prevents reconnect
  after shutdown, and stops heartbeat work without double-closing channels.

### Cache safety

- Holders and news APIs return independent values instead of mutable cache
  storage. Copies include nested insider date pointers, related-ticker slices,
  thumbnails, and thumbnail resolution slices.

## Deferred

- PR #11's timezone location cache is intentionally deferred. It is an
  unmeasured optimization whose global lifetime, growth bounds, invalid-name
  policy, and test reset behavior need a separate design and benchmark.

## Attribution

The protobuf, WebSocket, cache, cookie, and timezone observations originated
in PR #11 by `shubhbham`. Accepted fixes were reimplemented on the current
`main`, with contributor attribution retained in their commits. The timezone
proposal remains credited but is not part of this release.

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.6.1
```

## Documentation

- [API Reference](https://wnjoon.github.io/go-yfinance/API/)
- [v1.6.1 Progress](https://wnjoon.github.io/go-yfinance/development/v1.6.1-progress/)
