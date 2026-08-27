# go-yfinance v1.7.0

**Python yfinance v1.7.0 Parity Release**

This release ports the complete Python yfinance `1.6.0...1.7.0` behavior
range onto the Go `v1.6.1` maintenance baseline. It preserves the Go-specific
cookie, protobuf, WebSocket, and cache-safety fixes released in `v1.6.1`.

## Changes

### Lazy trading-period metadata

- `Ticker.GetTradingPeriods()` exposes Yahoo regular, pre-market, and
  post-market schedules through typed `models.TradingPeriod` values.
- Trading periods are loaded only when explicitly requested. Existing
  `GetHistoryMetadata()` behavior remains cache-only and ordinary daily
  history and FastInfo calls do not gain a hidden intraday request.
- Successful enrichment is coalesced and cached for concurrent callers.
  Failures preserve base metadata and can be retried; `ClearCache()` also
  invalidates an in-flight enrichment.
- Metadata and schedule results are deep copies, including nested timestamp
  pointers and valid-range slices.

### Cookie, crumb, and proxy resilience

- Transient cookie/crumb rate-limit, network, and timeout failures now degrade
  to a crumb-less target request instead of aborting endpoints that can work
  without a crumb.
- Target HTTP failures receive at most one retry with the alternate auth
  strategy. A retry never inserts an empty crumb, while non-transient auth and
  parsing failures still propagate.
- Crumb-endpoint rate limits remain distinct from target-endpoint rate limits,
  and caller query parameters are never mutated.
- All eight authenticated request paths share the same behavior, including the
  calendars JSON POST path. SOCKS5/SOCKS5h proxy configuration remains an
  immutable request snapshot.

### Range-based stock-split repair

- Split repair now detects and selectively corrects alternating missing or
  double-adjusted ranges instead of rescaling every pre-split row.
- Detection uses adjusted OHLC signals, upstream local-volatility and
  exceptional-volume false-positive suppression, and the finalized `0.2`
  volume-threshold coefficient.
- Price, dividend, repaired-marker, and integer-volume semantics follow the
  final Python v1.7.0 implementation. Go uses half-even rounding to match
  Pandas/NumPy before storing public `int64` volume values.
- The complete upstream NRDY fixture and its last-27-row subset are included as
  golden regressions, alongside forward/reverse, intraday, already-correct,
  large-dividend, and exceptional-volume cases.

## Adapted or not applicable

- Python's dict-like lazy metadata wrapper is adapted to the explicit Go
  `GetTradingPeriods() ([]models.TradingPeriod, error)` API so network access
  and failure remain visible.
- Python packaging/test cleanup (`nospam`, Python 2 cruft) and user-injected
  session ownership are not applicable to Go.
- Python merge and release-aggregation commits have no standalone Go runtime
  change; their contained behavior is covered above.
- The minor upstream zero-repair test correction is already represented by a
  deterministic Go test that selects an explicitly positive-volume row.

## Preserved from Go v1.6.1

- Complete CycleTLS cookie preservation and sanitized auth errors.
- Overflow-safe protobuf decoding and final, race-safe WebSocket shutdown.
- Immutable holders/news cache results and request-keyed news caching.

The previously deferred timezone location cache remains outside this parity
release; it is not part of Python yfinance v1.7.0.

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.7.0
```

## Documentation

- [API Reference](https://wnjoon.github.io/go-yfinance/API/)
- [v1.7.0 Progress](https://wnjoon.github.io/go-yfinance/development/v1.7.0-progress/)
