// Package repair provides price data repair functionality for financial time series.
//
// This package detects and corrects common data quality issues in Yahoo Finance data,
// including 100x currency errors, bad stock split adjustments, dividend double-counting,
// capital gains double-counting, and missing/zero values.
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
// match the split ratio, then applies corrections.
//
// This package is designed to match the behavior of Python yfinance's
// price repair functionality.
package repair
