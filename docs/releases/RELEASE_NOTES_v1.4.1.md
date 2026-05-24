# go-yfinance v1.4.1

**Go Stability and Quality Patch Release**

This patch release keeps go-yfinance on the Python yfinance v1.4.0 parity line while publishing Go-specific stability fixes and quality cleanup that landed after the v1.4.0 tag.

## Fixes

### Live WebSocket Stability

- Prevented a nil-pointer panic when subscription refreshes run while a reconnect has temporarily cleared the WebSocket connection.
- Serialized WebSocket writes to avoid gorilla/websocket concurrent write panics when heartbeat, reconnect resubscribe, or unsubscribe operations overlap.
- Added focused tests for the nil-connection and concurrent-write cases.

### Go Report Card Cleanup

- Applied `gofmt -s` cleanup across the files flagged by Go Report Card.
- Reduced high cyclomatic complexity warnings in the repair, ticker, and live packages.
- Fixed the remaining screener constant spelling warnings and regenerated API documentation.

## Notes

- The v1.4.0 tag remains unchanged because published Go module tags should not be moved.
- Go Report Card may continue to show v1.4.0 findings until it analyzes this v1.4.1 tag.
- Future releases should run Go Report Card refresh before tagging, in addition to local formatting, test, and lint checks.

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.4.1
```

## Documentation

- [API Reference](https://wnjoon.github.io/go-yfinance/API/)
- [v1.4.0 Progress](https://wnjoon.github.io/go-yfinance/development/v1.4.0-progress/)
