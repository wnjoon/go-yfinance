# go-yfinance v1.1.0

**Price Repair & International Exchange Support Release**

This release brings the Price Repair system from Python yfinance v1.1.0 and comprehensive international exchange support via MIC code mapping.

## New Features

### Price Repair System

A comprehensive data quality repair system that detects and fixes common Yahoo Finance data errors.

#### Capital Gains Double-Counting Fix
- Fixes ETF/MutualFund Adjusted Close miscalculation
- Detects when capital gains are counted twice in adjustments
- Applies intelligent repair when >66% of events show the issue

```go
bars, _ := t.History(models.HistoryParams{
    Period: "1y",
    Repair: true,
    RepairOptions: &models.RepairOptions{
        FixCapitalGains: true,
    },
})
```

#### Stock Split Repair
- IQR-based outlier detection for unadjusted historical prices
- Detects price changes matching split ratios
- Volume analysis to filter false positives

#### Unit Mixup Repair (100x Errors)
- 2D median filter for random 100x error detection
- Unit switch detection for permanent currency changes
- Currency-specific handling (KWF uses 1000x multiplier)

#### Zero Value Repair
- Interpolation for gaps with valid neighbors
- Forward/backward fill when only one neighbor exists
- Partial zero handling for partially corrupt bars

#### Dividend Repair
- Missing adjustment detection (Adj Close = Close)
- 100x too large/small dividend fixes
- Phantom (duplicate) dividend detection and removal

### MIC Code Mapping

ISO 10383 Market Identifier Code to Yahoo Finance suffix mapping for 60+ global exchanges.

```go
import "github.com/wnjoon/go-yfinance/pkg/utils"

// Format ticker for Yahoo Finance
ticker := utils.FormatYahooTicker("AAPL", "XLON")
// Result: "AAPL.L"

// Parse Yahoo ticker
base, suffix := utils.ParseYahooTicker("7203.T")
// Result: base="7203", suffix="T"

// Lookup functions
suffix := utils.GetYahooSuffix("XTKS")  // "T"
mic := utils.GetMIC("HK")               // "XHKG"
isUS := utils.IsUSExchange("XNYS")      // true
```

#### Supported Regions
- **Americas**: NYSE, NASDAQ, TSX, B3, BMV, and more
- **Europe**: LSE, XETRA, Euronext, SIX, Nasdaq Nordic, and more
- **Asia-Pacific**: TSE, HKEX, SSE, SZSE, SGX, ASX, and more
- **Middle East & Africa**: Tadawul, DFM, TASE, JSE, and more

### Statistics Package

New `pkg/stats` package for data analysis:

- `Percentile`, `IQR`, `Mean`, `Std`, `Median`
- `ZScore`, `RollingMean`, `RollingStd`
- `MedianFilter`, `MedianFilter2D`, `OutlierMask`

## API Changes

### New Types

```go
// pkg/models/history.go
type Bar struct {
    // ... existing fields ...
    CapitalGains float64  // New: Capital gains (ETF/MutualFund)
    Repaired     bool     // New: True if bar was repaired
}

type RepairOptions struct {
    FixUnitMixups   bool
    FixZeroes       bool
    FixSplits       bool
    FixDividends    bool
    FixCapitalGains bool
}
```

### New Packages

| Package | Purpose |
|---------|---------|
| `pkg/repair` | Price data repair operations |
| `pkg/stats` | Statistical functions |
| `pkg/utils` | MIC code mapping utilities |

## Documentation

- [Price Repair Guide](https://wnjoon.github.io/go-yfinance/advanced/price-repair/)
- [MIC Codes Reference](https://wnjoon.github.io/go-yfinance/advanced/mic-codes/)
- [API Reference](https://wnjoon.github.io/go-yfinance/API/)

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.1.0
```

## Compatibility

This release maintains full backward compatibility with v1.0.0 while adding new functionality aligned with Python yfinance v1.1.0.

## Development Notes

Implementation completed in 5 phases:
1. Statistics package and repair foundation
2. Capital Gains and Stock Split repair
3. Unit Mixup and Zero Value repair
4. Dividend repair
5. MIC code mapping

See [v1.1.0 Progress](docs/development/v1.1.0-progress.md) for detailed development history.
