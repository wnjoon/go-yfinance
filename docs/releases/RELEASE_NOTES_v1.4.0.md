# go-yfinance v1.4.0

**Python yfinance v1.4.0 Parity Release**

This release catches go-yfinance up with the main Python yfinance v1.4.0 changes around authentication, locale/region scoping, market behavior, and history repair activation.

## New Features

### Locale Configuration

Configure Yahoo locale parameters for endpoints that support localized fields:

```go
cfg := config.Get()
cfg.SetLocale("ja-JP", "JP")
```

Ticker quote/summary, lookup, and calendars requests now forward the configured `lang` and `region`.

### Market Regions

Use Python-compatible market regions:

```go
m, _ := market.NewWithRegion(models.MarketRegionEurope)
summary, _ := m.Summary()
```

Legacy values such as `us_market`, `gb_market`, and `jp_market` are normalized to compatible Yahoo market regions. Non-U.S. market status avoids returning misleading U.S. `markettime` data.

### Sector and Industry Region Scoping

Scope regional sector and industry lists:

```go
s, _ := sector.New("technology", sector.WithRegion("GB"))
i, _ := industry.New("software-infrastructure", industry.WithRegion("JP"))
```

### Manual Yahoo Login Cookies

Set manually retrieved Yahoo Finance login cookies:

```go
c, _ := client.New()
auth := client.NewAuthManager(c)
auth.SetLoginCookies(cookieT, cookieY)
ok, err := auth.CheckLogin()
```

Automated username/password login is intentionally not supported because Yahoo requires browser-based verification.

## Patch Parity

- `HistoryParams.Repair` now runs the repair pipeline when enabled.
- Client cookie handling now merges named cookies instead of overwriting the entire cookie header.
- Market summary/status requests use Yahoo's `market` parameter instead of the previous regional approximation.

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.4.0
```

## Documentation

- [API Reference](https://wnjoon.github.io/go-yfinance/API/)
- [v1.4.0 Progress](https://wnjoon.github.io/go-yfinance/development/v1.4.0-progress/)
