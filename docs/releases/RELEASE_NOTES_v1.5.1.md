# go-yfinance v1.5.1

**Python yfinance v1.5.1 Parity Release**

This release follows Python yfinance v1.5.1. Python yfinance v1.5.0 was retracted upstream, so go-yfinance targets the corrected v1.5.1 release line.

## New Features

### Valuation Measures Timeseries API

Valuation measures now use Yahoo's fundamentals-timeseries API instead of key-statistics HTML scraping.

```go
t, _ := ticker.New("AAPL")
periods := 5
valuation, err := t.GetValuationMeasures("quarterly", &periods)
```

`Valuation()` and `ValuationMeasures()` remain compatibility entry points and use Python-compatible defaults: quarterly data capped to five periods. Raw numeric cells are available through `FloatValue()`, while `Value()` remains for string compatibility.

### Yahoo Subscription Authentication

Login checks now use Yahoo's subscriptions endpoint instead of scraping the Yahoo Finance home page.

```go
c, _ := client.New()
auth := client.NewAuthManager(c)
ok, err := auth.SetLoginCookiesAndCheck(cookieT, cookieY)
tier, err := auth.SubscriptionTier()
```

`SubscriptionTier()` returns `gold`, `silver`, `bronze`, `premium`, `free`, or an empty string when the session is not logged in.

## Patch Parity

- Financial statements retry fundamentals-timeseries requests in 60-key chunks when the single long request fails or returns an invalid/empty result.
- Screener EPS fields now correctly split `netepsbasic.lasttwelvemonths` and `netepsdiluted.lasttwelvemonths`.
- Info parsing handles empty, missing, and null Yahoo quoteSummary result payloads defensively.
- Configured proxy strings are trimmed and passed through to CycleTLS request options.
- Manual login cookies are merged into the existing cookie jar and invalidate stale crumbs.

## Repair Hardening

- Non-finite adjusted-close values are guarded during history parsing, auto-adjust, dividend repair, zero repair, and unit-correction paths.
- Intraday pre/post dividend repair avoids false positives when a low-price drop recovers by the end of the trading session.
- Unit-switch repair now stops after the first detected switch in a repair pass to avoid overlapping prefix re-correction.

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.5.1
```

## Documentation

- [API Reference](https://wnjoon.github.io/go-yfinance/API/)
- [v1.5.1 Progress](https://wnjoon.github.io/go-yfinance/development/v1.5.1-progress/)
