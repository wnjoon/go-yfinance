package repair

import (
	"math"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/stats"
)

// repairStockSplits fixes bad stock split adjustments. It ports upstream
// _fix_bad_stock_splits (Python yfinance 1.8.0).
//
// Yahoo Finance sometimes fails to apply a stock split adjustment to older
// prices, or applies it twice, to all or only some rows. For each split,
// oldest first, the rows up to a short look-ahead past the split go through
// fixPricesSuddenChange with change = the split ratio, which looks for
// day-to-day price changes that match the ratio, keeps the ranges that
// volume confirms and rescales prices, dividends and volume.
func (r *Repairer) repairStockSplits(bars []models.Bar) []models.Bar {
	if len(bars) < 2 || !isInterday(r.opts.Interval) {
		return bars
	}

	// Upstream visits every row with a nonzero split (NaN included); a 1:1
	// split is then ignored as too close to 1.
	var splitIndices []int
	for i, bar := range bars {
		if bar.Splits != 0 {
			splitIndices = append(splitIndices, i)
		}
	}

	result := bars
	for _, idx := range splitIndices {
		if idx == 0 {
			continue
		}
		result = r.repairSplitAtIndex(result, idx, result[idx].Splits)
	}
	return result
}

// findSplitIndices returns indices of bars with stock splits.
func findSplitIndices(bars []models.Bar) []int {
	var indices []int
	for i, bar := range bars {
		if bar.Splits != 0 && bar.Splits != 1 {
			indices = append(indices, i)
		}
	}
	return indices
}

// repairSplitAtIndex repairs the rows up to a split plus a look-ahead of 5
// rows (1 row for weekly and monthly intervals), which is essential when
// corruption starts or stops near the split. Later rows are untouched.
func (r *Repairer) repairSplitAtIndex(bars []models.Bar, splitIdx int, splitRatio float64) []models.Bar {
	lookAhead := 5
	if r.opts.Interval != "1d" {
		lookAhead = 1
	}
	cutoff := minInt(len(bars), splitIdx+lookAhead+1)
	head := r.fixPricesSuddenChange(bars[:cutoff], splitRatio, priceShiftFix{correctVolume: true, correctDividend: true})
	if cutoff == len(bars) {
		return head
	}
	result := make([]models.Bar, 0, len(bars))
	result = append(result, head...)
	return append(result, bars[cutoff:]...)
}

// ohlcComplete reports whether a bar has no missing (NaN) OHLC value.
func ohlcComplete(bar models.Bar) bool {
	return !math.IsNaN(bar.Open) && !math.IsNaN(bar.High) && !math.IsNaN(bar.Low) && !math.IsNaN(bar.Close)
}

// ohlcMedian calculates the median of OHLC prices.
// This provides a robust estimate of the typical price.
func ohlcMedian(bar models.Bar) float64 {
	return stats.OHLCMedian(bar.Open, bar.High, bar.Low, bar.Close)
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SplitRepairStats contains statistics about split repair.
type SplitRepairStats struct {
	TotalSplits    int         // Number of split events found
	SplitsRepaired int         // Number of splits that were repaired
	BarsRepaired   int         // Total bars modified
	Splits         []SplitInfo // Details of each split
}

// SplitInfo contains information about a single split.
type SplitInfo struct {
	Date        time.Time
	Ratio       float64
	WasRepaired bool
}

// AnalyzeSplits analyzes bars for split issues without modifying.
func (r *Repairer) AnalyzeSplits(bars []models.Bar) SplitRepairStats {
	stats := SplitRepairStats{}

	splitIndices := findSplitIndices(bars)
	stats.TotalSplits = len(splitIndices)

	for _, idx := range splitIndices {
		info := SplitInfo{
			Date:  bars[idx].Date,
			Ratio: bars[idx].Splits,
		}
		stats.Splits = append(stats.Splits, info)
	}

	return stats
}

// DetectBadSplits checks if there are unadjusted splits in the data.
func DetectBadSplits(bars []models.Bar) []int {
	var badSplitIndices []int

	splitIndices := findSplitIndices(bars)
	for _, idx := range splitIndices {
		if idx == 0 {
			continue
		}

		splitRatio := bars[idx].Splits
		if splitRatio == 0 || splitRatio == 1 {
			continue
		}

		// Check if price change matches unadjusted split
		prevPrice := ohlcMedian(bars[idx-1])
		currPrice := ohlcMedian(bars[idx])
		if prevPrice == 0 {
			continue
		}

		pctChange := (currPrice - prevPrice) / prevPrice

		// For unadjusted n:1 split, price should drop by (1 - 1/n)
		// e.g., 2:1 split -> price drops by 50% -> pctChange = -0.5
		var expectedChange float64
		if splitRatio > 1 {
			// Normal split: price drops
			expectedChange = -(1.0 - 1.0/splitRatio) // e.g., 2:1 -> -0.5
		} else {
			// Reverse split: price increases
			expectedChange = (1.0/splitRatio - 1.0) // e.g., 1:4 (0.25) -> 3.0
		}

		// If price change is close to expected split change, it's likely unadjusted
		if math.Abs(pctChange-expectedChange) < 0.15 {
			badSplitIndices = append(badSplitIndices, idx)
		}
	}

	return badSplitIndices
}
