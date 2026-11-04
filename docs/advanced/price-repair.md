# Price Repair

The `repair` package provides functionality to detect and correct common data quality issues in Yahoo Finance data.

## Overview

Yahoo Finance data sometimes contains errors that can affect analysis:

| Error Type | Description | Example |
|------------|-------------|---------|
| **100x Errors** | Price in wrong currency unit | $150 shown as $15000 (cents) |
| **Stock Split Errors** | Split adjustment not applied | Pre-split prices not adjusted |
| **Unexplained Level Shifts** | Split-like shift with no split | Older prices 15x too high |
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

### How History Runs Repair

`History` follows Python yfinance's `history()` order:

1. The chart response is parsed. With `Repair`, dividend, split and
   capital-gain events are always parsed (repair needs them) and prices Yahoo
   reports as null are NaN, as upstream's DataFrame holds them
2. Without `KeepNA`, bars whose Open, High, Low and Close are all missing are
   dropped
3. The repairs below run on the raw, unadjusted prices
4. Prices still missing are reported as 0 again (bars never contain NaN), and
   the event fields are cleared unless `Actions` is set
5. `AutoAdjust` is applied last, from the repaired Close and Adj Close

Without `Repair`, `History` output is unchanged by these steps.

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
        FixLevelShifts:  true,  // Split-like shifts without a split (1h request)
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

## Dividend Repair

Fixes various dividend-related errors in Yahoo Finance data.

### Error Types

1. **Missing Adjustment**: Adj Close = Close (no dividend applied)
2. **Dividend 100x Too Big**: Currency unit error (e.g., $ shown as cents)
3. **Dividend 100x Too Small**: Currency unit error (e.g., cents shown as $)
4. **Phantom Dividend**: Duplicate dividend within 7-17 days

### Detection Algorithm

For each dividend:
1. Calculate expected price drop based on dividend yield
2. Compare with typical window volatility
3. Check Adj Close ratio before/after dividend
4. Identify anomalies exceeding normal volatility
5. Cross-check "100x too small" verdicts across dividends of similar yield
   (upstream's dividend clusters): a minority verdict is treated as price
   noise and dropped, a clear majority applies to the whole cluster

### Example

```go
// Fix dividend errors
bars, _ := t.History(models.HistoryParams{
    Period:   "1y",
    Interval: "1d",
    Repair:   true,
    RepairOptions: &models.RepairOptions{
        FixDividends: true,
    },
})
```

### Interval Restrictions

Dividend repair is **only applied to 1d intervals**. Weekly and monthly intervals are too volatile for reliable detection. For longer intervals, fetch daily data, repair, then resample.

## Stock Split Repair

Detects when Yahoo fails to apply split adjustments to historical prices.

Split, unit-switch and level-shift repair share one detector, a port of
Python yfinance 1.8.0 `_detect_price_shifts` / `_fix_prices_sudden_change`
that is differentially tested against the upstream code.

### Detection Algorithm

For each split (oldest first), the rows up to five rows after it (one row for
`1wk`/`1mo`/`3mo`) are scanned newest first:

1. Day-to-day changes are computed per price column on dividend-adjusted
   prices, and their median taken (for `1wk`/`1mo`/`3mo`, the average of Open
   and Close only, as High and Low are too volatile). Rows next to a zero Close
   count as no change
2. Normal volatility is estimated from changes inside the interquartile range;
   the split must stand well clear of it
3. Changes beyond `1 + (ratio - 1 + 5 sd%) x 0.6` (or its reciprocal) are
   signals; a drop on abnormally high volume, or a change similar to the local
   volatility, is discarded
4. Signals pair into ranges; ranges more than a year older than the oldest
   split are ignored, and the rest must be confirmed by a matching volume shift
5. Prices, dividends and volume in confirmed ranges are rescaled

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

## Unexplained Level Shift Repair

Sometimes Yahoo applies a split-like adjustment to all daily data before a
date although no split exists near it: older prices are N times too large and
volume N times too small, and N need not be a split ratio (for example SOXS
was 15x too high before 2026-05-26). Yahoo's 1h data is not affected, so it
is used to measure N.

This repair only runs for `Interval: "1d"` and non-FX tickers (no `=` in the
symbol), before dividend repair. Like Python yfinance 1.8.0, it makes **one
extra request** for the last 729 days of 1h data, and only when a candidate
shift is found:

1. Finds day-to-day changes of at least 2x that persist (median Close of the
   five rows after vs. before), away from the ends of the data, more than five
   rows from any split, and within the 1h data window
2. Compares 1d prices with 1h prices aggregated to exchange-timezone days on
   both sides of the shift. A multiple between 0.8 and 1.25 means the move is
   genuine (the 1h data moved too) and nothing changes
3. Requires a clean integer multiple (or reciprocal), within 3%
4. Repairs with the split-repair logic, which also requires volume to confirm
   the shift. Dividends are left unchanged

If the 1h request fails the data is returned unchanged, except for a rate
limit, which `History` returns as an error (check it with
`client.IsRateLimitError`).

```go
bars, _ := t.History(models.HistoryParams{
    Period:   "1y",
    Interval: "1d",
    Repair:   true,
    RepairOptions: &models.RepairOptions{
        FixLevelShifts: true,
    },
})
```

When using the `repair` package directly, the level-shift repair needs an
`Options.FetchHourly` function; without one it is skipped and `Repair` never
makes network requests.

## Unit Mixup Repair (100x Errors)

Yahoo Finance sometimes returns prices in wrong currency units (e.g., dollars vs cents, pounds vs pence).

### Types of Errors

1. **Random 100x Errors**: Sporadic errors scattered throughout data
2. **Unit Switch**: Permanent change in currency unit at some date

### Detection Algorithm

For random errors:
1. Apply 2D median filter (3x3 window) to OHLC prices
2. Calculate ratio of actual to median-filtered prices
3. Round ratio to nearest 20, check if ~100
4. Correct outliers by dividing/multiplying by 100

For unit switch, the shared price-shift detector (see Stock Split Repair) runs
with a 100x change (1000x for KWF):
1. Every range between a pair of 100x signals is corrected, so several
   switches (or a temporary one) are all repaired; prices and dividends are
   rescaled, volume is not
2. A range whose volume moves inversely to the price is a corporate action
   and is left alone
3. If a row's high/low spread exceeds 50x, columns are corrected individually
   and at least two columns must agree; Low/High are then recalculated
4. A suspended stock (no volume on the newest rows) uses the latest active row
   as its baseline
5. A 100x repair is abandoned when a signal falls within 30 days after a split

### Example

```go
// Fix 100x currency errors
bars, _ := t.History(models.HistoryParams{
    Period:   "1y",
    Interval: "1d",
    Repair:   true,
    RepairOptions: &models.RepairOptions{
        FixUnitMixups: true,
    },
})
```

### Currency-Specific Handling

- **USD, GBP, EUR, etc.**: 100x multiplier (dollars/cents)
- **KWF (Kuwaiti Dinar)**: 1000x multiplier (dinar/fils)

## Zero Value Repair

Fixes bars where prices are 0 or NaN but trading likely occurred.

### Detection Criteria

A zero bar is flagged for repair if:
- Volume > 0 (trading occurred)
- Stock split event present
- Dividend payment present
- Continuous price movement around the gap

### Repair Methods

1. **Interpolation**: If both prev and next bars exist, average the boundary prices
2. **Forward Fill**: Use previous bar's close if only prev available
3. **Backward Fill**: Use next bar's open if only next available

### Example

```go
// Fix zero/missing values
bars, _ := t.History(models.HistoryParams{
    Period:   "1y",
    Interval: "1d",
    Repair:   true,
    RepairOptions: &models.RepairOptions{
        FixZeroes: true,
    },
})
```

### Partial Zero Handling

When only some OHLC values are zero:
- Calculate mean of valid prices
- Fill zero values with the mean
- Ensure High >= all prices, Low <= all prices

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

1. **Unexplained level shifts** - Daily data, before dividend repair
2. **Dividend adjustments** - Before price-level errors
3. **100x unit errors** - Price level corrections
4. **Stock split errors** - Split ratio corrections
5. **Zero/missing values** - Gap filling
6. **Capital gains** - Last (needs clean data)

This order ensures each repair has clean data to work with.

## Using the Repair Package Directly

`repair.New(opts).Repair(bars)` expects bars sorted oldest first and follows
upstream's DataFrame conventions:

- A missing Open/High/Low/Close should be `math.NaN()`. Price-shift repair
  (split, unit switch, level shift) sets such rows aside. A `0` price is a real
  zero, as in upstream, and can make unit-switch repair correct columns
  individually
- `AdjClose == 0` (a `Bar` built without Adj Close) means unadjusted: it is
  treated as Adj Close = Close and left at 0. A NaN AdjClose behaves like
  upstream's NaN Adj Close
- `Splits`, `Dividends` and `CapitalGains` must be filled for split, dividend
  and capital-gains repair

## Performance Considerations

- Repair operations add processing overhead
- For large datasets, consider caching repaired data
- Repair is most useful for long-term historical data

## Comparison with Python yfinance

This implementation matches the behavior of Python yfinance's `repair=True` option, ensuring data consistency across languages.

## See Also

- [HistoryParams](../api/models.md#HistoryParams) - History parameters including RepairOptions
- [Statistics Package](../api/stats.md) - Statistical functions for repair algorithms
- [Repair Package](../api/repair.md) - Price repair implementation details
- [MIC Codes](mic-codes.md) - Exchange MIC code mapping
