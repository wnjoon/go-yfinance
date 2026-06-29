package repair

import (
	"math"

	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/stats"
)

// repairZeroes fixes missing or zero price values.
//
// Yahoo Finance sometimes returns prices as 0 or NaN when trades actually occurred.
// However, most times when prices are 0 or NaN it's because no trading happened.
// This function attempts repair only when zeroes are few and rare.
//
// Algorithm:
// 1. Identify bars with zero/NaN prices where trading likely occurred
// 2. Use volume and price movement as indicators of actual trading
// 3. Interpolate or reconstruct missing values from surrounding data
// 4. Skip repair if too many zeroes (no good calibration data)
func (r *Repairer) repairZeroes(bars []models.Bar) []models.Bar {
	if len(bars) < 3 {
		return bars
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Find bars with zero/missing prices
	zeroIndices := findZeroIndices(result)
	if len(zeroIndices) == 0 {
		return bars
	}

	// Skip if too many zeroes (need good data for calibration)
	if float64(len(zeroIndices))/float64(len(bars)) > 0.5 {
		return bars
	}

	// For each zero bar, attempt repair
	for _, idx := range zeroIndices {
		if shouldRepairZero(result, idx) {
			result[idx] = repairZeroBar(result, idx)
		}
	}

	return result
}

// findZeroIndices returns indices of bars with zero or missing prices.
func findZeroIndices(bars []models.Bar) []int {
	var indices []int
	for i, bar := range bars {
		if isZeroBar(bar) {
			indices = append(indices, i)
		}
	}
	return indices
}

// isZeroBar checks if a bar has zero or invalid price data.
func isZeroBar(bar models.Bar) bool {
	return invalidPrice(bar.Open) || invalidPrice(bar.High) || invalidPrice(bar.Low) || invalidPrice(bar.Close)
}

// shouldRepairZero determines if a zero bar should be repaired.
// Returns true if there's evidence that trading occurred.
func shouldRepairZero(bars []models.Bar, idx int) bool {
	bar := bars[idx]

	// If volume > 0, trading likely occurred
	if bar.Volume > 0 {
		return true
	}

	// If there's a stock split, trading must have occurred
	if bar.Splits != 0 && bar.Splits != 1 {
		return true
	}

	// If there's a dividend, trading likely occurred
	if bar.Dividends > 0 {
		return true
	}

	// Check for price continuity gap
	if idx > 0 && idx < len(bars)-1 {
		prevClose := bars[idx-1].Close
		nextOpen := bars[idx+1].Open
		if prevClose > 0 && nextOpen > 0 {
			// If prices before and after are similar, likely just missing data
			change := math.Abs(nextOpen-prevClose) / prevClose
			if change < 0.1 { // Less than 10% gap
				return true
			}
		}
	}

	return false
}

// repairZeroBar attempts to repair a bar with zero values.
func repairZeroBar(bars []models.Bar, idx int) models.Bar {
	bar := bars[idx]
	bar.Repaired = true

	// Try to interpolate from surrounding bars
	var prevBar, nextBar *models.Bar

	if idx > 0 && !isZeroBar(bars[idx-1]) {
		prevBar = &bars[idx-1]
	}

	if idx < len(bars)-1 && !isZeroBar(bars[idx+1]) {
		nextBar = &bars[idx+1]
	}

	// Case 1: Both neighbors available - interpolate
	if prevBar != nil && nextBar != nil {
		bar.Open = (prevBar.Close + nextBar.Open) / 2
		bar.Close = (prevBar.Close + nextBar.Open) / 2
		bar.High = math.Max(prevBar.Close, nextBar.Open)
		bar.Low = math.Min(prevBar.Close, nextBar.Open)
		bar.AdjClose = bar.Close
		return bar
	}

	// Case 2: Only previous bar available - forward fill
	if prevBar != nil {
		bar.Open = prevBar.Close
		bar.High = prevBar.Close
		bar.Low = prevBar.Close
		bar.Close = prevBar.Close
		bar.AdjClose = prevBar.AdjClose
		return bar
	}

	// Case 3: Only next bar available - backward fill
	if nextBar != nil {
		bar.Open = nextBar.Open
		bar.High = nextBar.Open
		bar.Low = nextBar.Open
		bar.Close = nextBar.Open
		bar.AdjClose = nextBar.AdjClose
		return bar
	}

	// No neighbors, cannot repair
	bar.Repaired = false
	return bar
}

// repairPartialZeroes fixes bars where some but not all OHLC values are zero.
func repairPartialZeroes(bars []models.Bar) []models.Bar {
	result := make([]models.Bar, len(bars))
	copy(result, bars)

	for i := range result {
		bar := &result[i]

		zeroCount := invalidOHLCCount(*bar)
		if zeroCount == 0 || zeroCount == 4 {
			continue
		}

		goodValues := goodOHLCValues(*bar)
		if len(goodValues) == 0 {
			continue
		}

		avg := stats.Mean(goodValues)
		fillInvalidOHLC(bar, avg)
		normalizeOHLCBounds(bar)
		fillInvalidAdjClose(bar)
	}

	return result
}

func invalidOHLCCount(bar models.Bar) int {
	count := 0
	for _, value := range []float64{bar.Open, bar.High, bar.Low, bar.Close} {
		if invalidPrice(value) {
			count++
		}
	}
	return count
}

func goodOHLCValues(bar models.Bar) []float64 {
	values := make([]float64, 0, 4)
	for _, value := range []float64{bar.Open, bar.High, bar.Low, bar.Close} {
		if validPrice(value) {
			values = append(values, value)
		}
	}
	return values
}

func fillInvalidOHLC(bar *models.Bar, value float64) {
	if invalidPrice(bar.Open) {
		bar.Open = value
		bar.Repaired = true
	}
	if invalidPrice(bar.High) {
		bar.High = value
		bar.Repaired = true
	}
	if invalidPrice(bar.Low) {
		bar.Low = value
		bar.Repaired = true
	}
	if invalidPrice(bar.Close) {
		bar.Close = value
		bar.Repaired = true
	}
}

func normalizeOHLCBounds(bar *models.Bar) {
	bar.High = math.Max(bar.High, math.Max(bar.Open, bar.Close))
	bar.Low = math.Min(bar.Low, math.Min(bar.Open, bar.Close))
}

func fillInvalidAdjClose(bar *models.Bar) {
	if invalidPrice(bar.AdjClose) {
		bar.AdjClose = bar.Close
	}
}

func invalidPrice(value float64) bool {
	return !validPrice(value)
}

func validPrice(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

// repairVolumeZeroes fixes bars where volume is zero but price changed.
func repairVolumeZeroes(bars []models.Bar) []models.Bar {
	if len(bars) < 2 {
		return bars
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)

	for i := 1; i < len(result); i++ {
		bar := &result[i]
		prevBar := result[i-1]

		// Skip if volume is non-zero
		if bar.Volume > 0 {
			continue
		}

		// Check if price changed significantly
		if prevBar.Close == 0 {
			continue
		}

		priceChange := math.Abs(bar.Close-prevBar.Close) / prevBar.Close

		// If price changed more than 5% with zero volume, estimate volume
		if priceChange > 0.05 {
			// Look for nearby bars with similar price changes
			var similarVolumes []float64

			// Check surrounding bars
			for j := maxInt(0, i-5); j <= minInt(len(result)-1, i+5); j++ {
				if j == i || result[j].Volume == 0 {
					continue
				}
				if j > 0 && result[j-1].Close > 0 {
					otherChange := math.Abs(result[j].Close-result[j-1].Close) / result[j-1].Close
					if math.Abs(otherChange-priceChange) < 0.02 {
						similarVolumes = append(similarVolumes, float64(result[j].Volume))
					}
				}
			}

			if len(similarVolumes) > 0 {
				bar.Volume = int64(stats.Median(similarVolumes))
				bar.Repaired = true
			}
		}
	}

	return result
}

// ZeroRepairStats contains statistics about zero value repairs.
type ZeroRepairStats struct {
	TotalBars       int // Total bars analyzed
	ZeroBars        int // Bars with zero prices
	PartialZeroBars int // Bars with some zero values
	ZeroVolumeBars  int // Bars with zero volume but price changed
	BarsRepaired    int // Total bars repaired
}

// AnalyzeZeroes analyzes bars for zero/missing values without modifying.
func (r *Repairer) AnalyzeZeroes(bars []models.Bar) ZeroRepairStats {
	stats := ZeroRepairStats{
		TotalBars: len(bars),
	}

	for i, bar := range bars {
		if isZeroBar(bar) {
			stats.ZeroBars++
		} else {
			// Check for partial zeroes
			if bar.Open == 0 || bar.High == 0 || bar.Low == 0 || bar.Close == 0 {
				stats.PartialZeroBars++
			}
		}

		// Check for zero volume with price change
		if i > 0 && bar.Volume == 0 && bars[i-1].Close > 0 {
			priceChange := math.Abs(bar.Close-bars[i-1].Close) / bars[i-1].Close
			if priceChange > 0.05 {
				stats.ZeroVolumeBars++
			}
		}
	}

	return stats
}

// DetectZeroes checks for bars with zero/missing values.
// Returns indices of bars that may need repair.
func DetectZeroes(bars []models.Bar) []int {
	return findZeroIndices(bars)
}

// maxInt returns the maximum of two integers.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
