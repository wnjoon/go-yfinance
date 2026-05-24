# go-yfinance v1.3.0

**Python yfinance v1.3.0 Parity Release**

This release catches go-yfinance up with Python yfinance changes after v1.2.0, including the v1.2.1 and v1.2.2 patch releases.

## New Features

### ETFQuery Support

Build validated ETF screener queries and run the new ETF predefined screeners:

```go
q1, _ := models.NewETFQuery("gt", []any{"intradayprice", 10})
q2, _ := models.NewETFQuery("eq", []any{"region", "us"})
query, _ := models.NewETFQuery("and", []any{q1, q2})
result, err := s.ScreenWithQuery(query, nil)
```

New predefined ETF screeners:

| Screener | Sort Field |
|----------|------------|
| `top_etfs_us` | `percentchange` |
| `top_performing_etfs` | `annualreportnetexpenseratio` |
| `technology_etfs` | `annualreportnetexpenseratio` |
| `bond_etfs` | `annualreportnetexpenseratio` |

### Valuation Measures

Fetch the key-statistics valuation table:

```go
t, _ := ticker.New("AAPL")
valuation, err := t.Valuation()
if err != nil {
    log.Fatal(err)
}

marketCap, ok := valuation.Value("Market Cap", "Current")
```

## Patch Parity

- Preserves dividend currency separately from numeric dividend amounts.
- Parses dividend, split, and capital gain action events directly from Yahoo chart payloads.
- Adds `Ticker.CapitalGains()` and includes capital gains in `Ticker.Actions()`.
- Adds currency fields to analysis estimates and EPS trend/revision models.
- Confirms Go multi-ticker download does not use Python's shared global DataFrame state.

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.3.0
```

## Documentation

- [API Reference](https://wnjoon.github.io/go-yfinance/API/)
- [v1.3.0 Progress](https://wnjoon.github.io/go-yfinance/development/v1.3.0-progress/)
