package repair

import (
	"math"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/stats"
)

// repairStockSplits fixes bad stock split adjustments.
//
// Yahoo Finance sometimes fails to apply stock split adjustments to historical
// prices, or applies them incorrectly. This function detects and corrects these issues.
//
// Algorithm:
// 1. Find all split events in the data
// 2. For each split, analyze price changes around the split date
// 3. Use IQR-based outlier detection to identify normal volatility
// 4. Detect if the split ratio appears in the price changes (suggesting unadjusted data)
// 5. Apply corrections by multiplying/dividing by the split ratio
func (r *Repairer) repairStockSplits(bars []models.Bar) []models.Bar {
	if len(bars) < 2 {
		return bars
	}

	// Find split events
	splitIndices := findSplitIndices(bars)
	if len(splitIndices) == 0 {
		return bars
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Process each split (from oldest to newest)
	for _, idx := range splitIndices {
		if idx == 0 {
			continue
		}

		splitRatio := result[idx].Splits

		// Skip trivial splits (1:1)
		if splitRatio == 0 || splitRatio == 1 {
			continue
		}

		// Analyze and repair data before this split
		result = repairSplitAtIndex(result, idx, splitRatio)
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

// repairSplitAtIndex repairs data around a specific split event.
func repairSplitAtIndex(bars []models.Bar, splitIdx int, splitRatio float64) []models.Bar {
	if splitIdx == 0 {
		return bars
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Calculate price changes leading up to the split
	// Look at a window of data before the split
	windowSize := minInt(splitIdx, 20) // Look at up to 20 days before split
	if windowSize < 3 {
		return bars // Not enough data
	}

	startIdx := splitIdx - windowSize

	// Calculate daily percentage changes using median of OHLC
	pctChanges := make([]float64, 0, windowSize)
	for i := startIdx + 1; i <= splitIdx; i++ {
		prevPrice := ohlcMedian(result[i-1])
		currPrice := ohlcMedian(result[i])
		if prevPrice != 0 {
			pctChange := (currPrice - prevPrice) / prevPrice
			pctChanges = append(pctChanges, pctChange)
		}
	}

	if len(pctChanges) < 3 {
		return bars
	}

	// Use IQR to estimate normal volatility
	q1, q3, iqr := stats.IQR(pctChanges)
	if math.IsNaN(iqr) || iqr == 0 {
		return bars
	}

	// Filter out outliers to get "normal" volatility
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	var normalChanges []float64
	for _, pct := range pctChanges {
		if pct >= lowerBound && pct <= upperBound {
			normalChanges = append(normalChanges, math.Abs(pct))
		}
	}

	if len(normalChanges) == 0 {
		return bars
	}

	// Calculate standard deviation of normal changes
	stdDev := stats.Std(normalChanges, 1)
	if math.IsNaN(stdDev) {
		stdDev = stats.Mean(normalChanges)
	}

	// Calculate the expected price change from split
	// For a n:1 split, price should be divided by n (ratio > 1)
	// For a 1:n reverse split, price should be multiplied by n (ratio < 1)
	var expectedChange float64
	if splitRatio > 1 {
		expectedChange = 1.0 - 1.0/splitRatio // e.g., 4:1 split -> -75%
	} else {
		expectedChange = splitRatio - 1.0 // e.g., 1:4 reverse -> -75%
	}

	// Set threshold: halfway between split change and largest normal change
	largestNormalChange := 5 * stdDev
	threshold := (math.Abs(expectedChange) + largestNormalChange) / 2

	// Check if the split appears unadjusted
	// Look at the price change on the split date
	splitDateChange := 0.0
	if splitIdx > 0 {
		prevPrice := ohlcMedian(result[splitIdx-1])
		currPrice := ohlcMedian(result[splitIdx])
		if prevPrice != 0 {
			splitDateChange = (currPrice - prevPrice) / prevPrice
		}
	}

	// If the change on split date is close to what we'd expect from an unadjusted split,
	// the data before the split needs to be adjusted
	if math.Abs(splitDateChange-expectedChange) < threshold {
		// Data appears to be unadjusted - apply correction
		result = applySplitCorrection(result, splitIdx, splitRatio)
	}

	return result
}

// applySplitCorrection adjusts historical prices for a split.
func applySplitCorrection(bars []models.Bar, splitIdx int, splitRatio float64) []models.Bar {
	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// For a n:1 split (ratio > 1), multiply pre-split prices by ratio
	// For a 1:n reverse split (ratio < 1), divide pre-split prices by ratio
	for i := 0; i < splitIdx; i++ {
		if splitRatio > 1 {
			// Normal split: historical prices should be lower
			result[i].Open /= splitRatio
			result[i].High /= splitRatio
			result[i].Low /= splitRatio
			result[i].Close /= splitRatio
			result[i].AdjClose /= splitRatio
			result[i].Volume = int64(float64(result[i].Volume) * splitRatio)
		} else if splitRatio > 0 {
			// Reverse split: historical prices should be higher
			result[i].Open *= (1 / splitRatio)
			result[i].High *= (1 / splitRatio)
			result[i].Low *= (1 / splitRatio)
			result[i].Close *= (1 / splitRatio)
			result[i].AdjClose *= (1 / splitRatio)
			result[i].Volume = int64(float64(result[i].Volume) / (1 / splitRatio))
		}
		result[i].Repaired = true
	}

	// Mark split date as repaired
	result[splitIdx].Repaired = true

	return result
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
	TotalSplits   int       // Number of split events found
	SplitsRepaired int      // Number of splits that were repaired
	BarsRepaired  int       // Total bars modified
	Splits        []SplitInfo // Details of each split
}

// SplitInfo contains information about a single split.
type SplitInfo struct {
	Date      time.Time
	Ratio     float64
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
