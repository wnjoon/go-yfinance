# go-yfinance v1.2.0

**Screener Python Parity Release**

This release achieves full screener module parity with Python yfinance v1.2.0, including client-side query validation, FundQuery support, IS-IN operator, and all 15 predefined screener queries.

## New Features

### Query Validation (EquityQuery / FundQuery)

Client-side validation matching Python's `EquityQuery` and `FundQuery` classes. Fields and values are validated before sending to Yahoo Finance API.

```go
// Equity query with validation
q1, _ := models.NewEquityQuery("eq", []any{"region", "us"})
q2, _ := models.NewEquityQuery("eq", []any{"sector", "Technology"})
q3, _ := models.NewEquityQuery("gt", []any{"intradayprice", 50})
query, _ := models.NewEquityQuery("and", []any{q1, q2, q3})
result, err := s.ScreenWithQuery(query, nil)
```

```go
// Fund query
q1, _ := models.NewFundQuery("eq", []any{"exchange", "NAS"})
q2, _ := models.NewFundQuery("gt", []any{"performanceratingoverall", 3})
query, _ := models.NewFundQuery("and", []any{q1, q2})
result, err := s.ScreenWithQuery(query, nil)
```

### IS-IN Operator

New operator that auto-expands to OR-of-EQ, matching Python behavior:

```go
// Find stocks on NMS or NYQ exchanges
q, _ := models.NewEquityQuery("is-in", []any{"exchange", "NMS", "NYQ"})
// Expands to: OR(EQ("exchange","NMS"), EQ("exchange","NYQ"))
```

### Fund Screener Support

6 new predefined fund screeners with fund-specific result fields:

```go
result, err := s.Screen(models.ScreenerConservativeForeign, nil)
for _, quote := range result.Quotes {
    fmt.Printf("%s: %s (Rating: %d)\n",
        quote.Symbol, quote.CategoryName, quote.PerformanceRatingOverall)
}
```

### Predefined Screener Queries

All 15 predefined screeners from Python's `PREDEFINED_SCREENER_QUERIES`:

| Type | Screeners |
|------|-----------|
| Equity (9) | `aggressive_small_caps`, `day_gainers`, `day_losers`, `growth_technology_stocks`, `most_actives`, `most_shorted_stocks`, `small_cap_gainers`, `undervalued_growth_stocks`, `undervalued_large_caps` |
| Fund (6) | `conservative_foreign_funds`, `high_yield_bond`, `portfolio_anchors`, `solid_large_growth_funds`, `solid_midcap_growth_funds`, `top_mutual_funds` |

### Validation Maps

Complete screener field and value validation ported from Python `const.py`:

| Map | Size |
|-----|------|
| `EquityScreenerExchangeMap` | 57 regions |
| `FundScreenerExchangeMap` | 56 regions |
| `SectorIndustryMapping` | 11 sectors with industries |
| `EquityScreenerPeerGroups` | 74 peer groups |
| `EquityScreenerFields` | ~91 fields in 12 categories |
| `FundScreenerFields` | 9 fields in 2 categories |

## API Changes

### Breaking Changes

```go
// Before (v1.1.0)
query := models.NewEquityQuery(models.OpAND, []interface{}{...})
result, err := s.ScreenWithQuery(query, nil)  // query *models.ScreenerQuery

// After (v1.2.0)
query, err := models.NewEquityQuery("and", []any{...})  // returns error
result, err := s.ScreenWithQuery(query, nil)  // query models.ScreenerQueryBuilder
```

- `ScreenWithQuery` parameter: `*models.ScreenerQuery` → `models.ScreenerQueryBuilder`
- `NewEquityQuery` return: `*ScreenerQuery` → `(*EquityQuery, error)`
- `QueryOperator` type removed, operators are now plain string constants

### New Types

```go
// pkg/models/screener_query.go
type ScreenerQueryBuilder interface {
    ToDict() map[string]any
    QuoteType() string
}

type EquityQuery struct { /* unexported fields */ }
type FundQuery struct { /* unexported fields */ }

func NewEquityQuery(operator string, operands []any) (*EquityQuery, error)
func NewFundQuery(operator string, operands []any) (*FundQuery, error)
```

```go
// pkg/models/screener.go - New fields in ScreenerQuote
type ScreenerQuote struct {
    // ... existing fields ...
    FundNetAssets                 float64  // Fund total net assets
    CategoryName                 string   // Fund category name
    PerformanceRatingOverall     int      // Overall performance rating (1-5)
    RiskRatingOverall            int      // Overall risk rating (1-5)
    InitialInvestment            float64  // Minimum initial investment
    AnnualReturnNavY1CategoryRank float64 // 1-year annual return category rank
}

// New fields in ScreenerParams
type ScreenerParams struct {
    // ... existing fields ...
    UserID     string  // User identifier (default "")
    UserIDType string  // Type of user ID (default "guid")
}
```

### New Operator

```go
const OpISIN = "is-in"  // Is in (expanded to OR of EQ)
```

### New Predefined Screener Constants

```go
const (
    ScreenerConservativeForeign PredefinedScreener = "conservative_foreign_funds"
    ScreenerHighYieldBond       PredefinedScreener = "high_yield_bond"
    ScreenerPortfolioAnchors    PredefinedScreener = "portfolio_anchors"
    ScreenerSolidLargeGrowth    PredefinedScreener = "solid_large_growth_funds"
    ScreenerSolidMidcapGrowth   PredefinedScreener = "solid_midcap_growth_funds"
    ScreenerTopMutualFunds      PredefinedScreener = "top_mutual_funds"
)
```

### New Files

| File | Purpose |
|------|---------|
| `pkg/models/screener_const.go` | Validation maps from Python `const.py` |
| `pkg/models/screener_query.go` | EquityQuery, FundQuery, validation logic |
| `pkg/screener/predefined.go` | 15 predefined screener queries |

## Documentation

- [API Reference](https://wnjoon.github.io/go-yfinance/API/)
- [v1.2.0 Progress](https://wnjoon.github.io/go-yfinance/development/v1.2.0-progress/)

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.2.0
```

## Development Notes

See [v1.2.0 Progress](https://wnjoon.github.io/go-yfinance/development/v1.2.0-progress/) for detailed development history.
