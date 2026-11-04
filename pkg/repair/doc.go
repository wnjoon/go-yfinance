// Package repair provides price data repair functionality for financial time series.
//
// This package detects and corrects common data quality issues in Yahoo Finance data,
// including 100x currency errors, bad stock split adjustments, split-like level
// shifts with no matching split, dividend double-counting, capital gains
// double-counting, and missing/zero values.
//
// # Overview
//
// Yahoo Finance data sometimes contains errors that need to be repaired:
//   - 100x errors: Price appears in cents instead of dollars (or vice versa)
//   - Bad stock splits: Split adjustments not applied or applied incorrectly
//   - Bad dividends: Dividend adjustments not applied correctly
//   - Capital gains double-counting: For ETFs/MutualFunds, capital gains counted twice
//   - Zero/missing values: Prices showing as 0 or NaN
//
// # Usage
//
// Create a Repairer and call Repair on your bar data:
//
//	opts := repair.DefaultOptions()
//	opts.Interval = "1d"
//	opts.QuoteType = "ETF"
//
//	repairer := repair.New(opts)
//	repairedBars, err := repairer.Repair(bars)
//
// # Repair Options
//
// Individual repair functions can be enabled/disabled:
//
//	opts := repair.Options{
//	    FixUnitMixups:   true,   // Fix 100x errors
//	    FixZeroes:       true,   // Fix zero/missing values
//	    FixSplits:       true,   // Fix stock split errors
//	    FixDividends:    true,   // Fix dividend adjustment errors
//	    FixCapitalGains: true,   // Fix capital gains double-counting
//	    FixLevelShifts:  true,   // Fix split-like shifts with no split (needs FetchHourly)
//	}
//
// # Capital Gains Repair (v1.1.0)
//
// For ETFs and Mutual Funds, Yahoo Finance sometimes double-counts capital gains
// in the Adjusted Close calculation. This repair detects and corrects this issue:
//
//	// Only applies to ETF and MUTUALFUND quote types
//	opts.QuoteType = "ETF"
//	opts.FixCapitalGains = true
//
// The algorithm compares price drops on distribution days against expected drops
// based on dividend vs dividend+capital_gains to detect double-counting.
//
// # Stock Split Repair
//
// Detects when Yahoo fails to apply stock split adjustments to historical data:
//
//	opts.FixSplits = true
//
// Uses IQR-based outlier detection to identify suspicious price changes that
// match the split ratio, then applies corrections. Each candidate range must
// be confirmed by trading volume (volume moving inversely to price, or a
// shift of more than 15 standard deviations); unconfirmed ranges are left
// unchanged.
//
// # Unexplained Level Shift Repair
//
// Yahoo sometimes applies a split-like adjustment to daily data before a date
// with no split nearby: older prices are N times too large and volume N times
// too small, where N need not be a split ratio. Yahoo's 1h data is not
// affected, so N is measured from it:
//
//	opts.FixLevelShifts = true
//	opts.FetchHourly = func(start, end time.Time) ([]models.Bar, error) {
//	    // return raw 1h bars in [start, end), without running repair
//	}
//
// It runs first (before dividend repair), only for interval "1d" and
// tickers without '='. Candidate 2x+ day-to-day changes must persist, lie
// away from the data ends and more than five rows from any split, and fall
// within the last 729 days. FetchHourly is then called once; the 1d/1h price
// ratios on each side of the shift give the multiple, which must be outside
// (0.8, 1.25) (otherwise the move is genuine) and within 3% of an integer or
// integer reciprocal. The repair reuses split repair, including its volume
// confirmation, and leaves dividends unchanged. With a nil FetchHourly the
// step is skipped, so Repair never makes network requests by itself. A
// rate-limit error from FetchHourly is returned by Repair; other errors skip
// the step. [github.com/wnjoon/go-yfinance/pkg/ticker.Ticker.History] wires
// FetchHourly when repair is requested.
//
// This package is designed to match the behavior of Python yfinance's
// price repair functionality.
package repair
