# Price Repair

The `repair` package provides functionality to detect and correct common data quality issues in Yahoo Finance data.

## Overview

Yahoo Finance data sometimes contains errors that can affect analysis:

| Error Type | Description | Example |
|------------|-------------|---------|
| **100x Errors** | Price in wrong currency unit | $150 shown as $15000 (cents) |
| **Stock Split Errors** | Split adjustment not applied | Pre-split prices not adjusted |
| **Dividend Errors** | Dividend adjustment incorrect | Adj Close miscalculated |
| **Capital Gains Double-Count** | CG counted twice in Adj Close | ETF/MutualFund specific |
| **Missing/Zero Values** | Price gaps or zeros | Trading halts, data errors |

## Basic Usage

Enable repair when fetching history:

```go
import (
    "github.com/wnjoon/go-yfinance/pkg/ticker"
    "github.com/wnjoon/go-yfinance/pkg/models"
)

t, err := ticker.New("AAPL")
if err != nil {
    log.Fatal(err)
}
defer t.Close()

// Enable all repairs
bars, err := t.History(models.HistoryParams{
    Period:   "1y",
    Interval: "1d",
    Repair:   true,
})
```

## Fine-Grained Control

Use `RepairOptions` for specific repairs:

```go
bars, err := t.History(models.HistoryParams{
    Period:   "1y",
    Interval: "1d",
    Repair:   true,
    RepairOptions: &models.RepairOptions{
        FixCapitalGains: true,  // ETF/MutualFund
        FixSplits:       true,  // Stock splits
        FixDividends:    false, // Disable dividend repair
        FixUnitMixups:   true,  // 100x errors
        FixZeroes:       true,  // Missing values
    },
})
```

## Capital Gains Repair

For ETFs and Mutual Funds, Yahoo Finance sometimes double-counts capital gains in the Adjusted Close calculation.

### The Problem

When an ETF distributes capital gains:
1. Yahoo adds Capital Gains to the Dividends column
2. Yahoo also adjusts Adj Close for Capital Gains separately
3. Result: Capital Gains counted twice, historical prices appear too low

### Detection

The repair algorithm:
1. Calculates normal price volatility (excluding distribution days)
2. Compares price drop vs dividend expectation vs (dividend + capital gains)
3. If >66% of events show double-counting, repairs all

### Example

```go
// For ETF with capital gains
t, _ := ticker.New("VWILX")

bars, _ := t.History(models.HistoryParams{
    Period:   "1y",
    Interval: "1d",
    Repair:   true,
    RepairOptions: &models.RepairOptions{
        FixCapitalGains: true,
    },
})

// Check if any bars were repaired
for _, bar := range bars {
    if bar.Repaired {
        fmt.Printf("%s was repaired\n", bar.Date)
    }
}
```

## Stock Split Repair

Detects when Yahoo fails to apply split adjustments to historical prices.

### Detection Algorithm

1. Identifies split events from data
2. Uses IQR-based outlier detection for normal volatility
3. Detects price changes matching split ratio
4. Filters false positives using volume analysis

### Example

```go
// Check for split repair
bars, _ := t.History(models.HistoryParams{
    Period:   "max",
    Interval: "1wk",
    Repair:   true,
    RepairOptions: &models.RepairOptions{
        FixSplits: true,
    },
})
```

## Data Fields

The `Bar` struct includes repair-related fields:

```go
type Bar struct {
    Date         time.Time
    Open         float64
    High         float64
    Low          float64
    Close        float64
    AdjClose     float64
    Volume       int64
    Dividends    float64   // Dividend amount
    Splits       float64   // Split ratio
    CapitalGains float64   // Capital gains (ETF/MutualFund)
    Repaired     bool      // True if this bar was repaired
}
```

## Repair Order

The repair system applies fixes in a specific order:

1. **Dividend adjustments** - Must come first
2. **100x unit errors** - Price level corrections
3. **Stock split errors** - Split ratio corrections
4. **Zero/missing values** - Gap filling
5. **Capital gains** - Last (needs clean data)

This order ensures each repair has clean data to work with.

## Performance Considerations

- Repair operations add processing overhead
- For large datasets, consider caching repaired data
- Repair is most useful for long-term historical data

## Comparison with Python yfinance

This implementation matches the behavior of Python yfinance's `repair=True` option, ensuring data consistency across languages.

## See Also

- [History API](../API.md#history)
- [Statistics Package](../API.md#stats-package)
