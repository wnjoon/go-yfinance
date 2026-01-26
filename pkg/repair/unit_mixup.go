package repair

import (
	"math"

	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/stats"
)

// repairUnitMixups fixes 100x currency errors ($/cents, £/pence mixups).
//
// Yahoo Finance sometimes returns prices in the wrong currency unit,
// causing values to be 100x too high or too low. This function detects
// and corrects these errors using two methods:
//
// 1. Unit Switch: A sudden permanent change in currency unit at some date
// 2. Random Mixups: Sporadic 100x errors scattered throughout the data
//
// Algorithm for random mixups:
// 1. Apply 2D median filter to OHLC prices
// 2. Calculate ratio of actual to median-filtered prices
// 3. Round ratio to nearest 20 and check if ~100
// 4. Correct identified outliers by dividing/multiplying by 100
func (r *Repairer) repairUnitMixups(bars []models.Bar) []models.Bar {
	if len(bars) < 2 {
		return bars
	}

	// First fix any unit switch (permanent currency change)
	result := r.repairUnitSwitch(bars)

	// Then fix random 100x errors
	result = r.repairRandomUnitMixups(result)

	return result
}

// repairUnitSwitch fixes a sudden permanent switch between currency units.
// For example, when Yahoo suddenly starts returning prices in cents instead of dollars.
func (r *Repairer) repairUnitSwitch(bars []models.Bar) []models.Bar {
	if len(bars) < 3 {
		return bars
	}

	// Determine multiplier based on currency
	n := 100.0
	if r.opts.Currency == "KWF" {
		// Kuwaiti Dinar divided into 1000, not 100
		n = 1000.0
	}

	return r.fixPricesSuddenChange(bars, n)
}

// repairRandomUnitMixups fixes sporadic 100x errors scattered through the data.
// Uses median filtering to detect outliers that are ~100x the local median.
func (r *Repairer) repairRandomUnitMixups(bars []models.Bar) []models.Bar {
	if len(bars) < 3 {
		return bars
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Extract OHLC data as 2D array
	// Order: High, Open, Low, Close, AdjClose (important: separate High from Low)
	data := make([][]float64, len(bars))
	for i, bar := range bars {
		data[i] = []float64{bar.High, bar.Open, bar.Low, bar.Close, bar.AdjClose}
	}

	// Check for and exclude zero values
	hasZeroes := false
	for i := range data {
		for j := range data[i] {
			if data[i][j] == 0 {
				hasZeroes = true
				break
			}
		}
		if hasZeroes {
			break
		}
	}

	if hasZeroes {
		// Filter out rows with zeros for analysis
		// but keep them in result
		nonZeroIndices := make([]int, 0)
		for i := range data {
			hasZero := false
			for j := range data[i] {
				if data[i][j] == 0 {
					hasZero = true
					break
				}
			}
			if !hasZero {
				nonZeroIndices = append(nonZeroIndices, i)
			}
		}

		if len(nonZeroIndices) < 3 {
			return bars // Not enough good data
		}

		// Create filtered dataset
		filteredData := make([][]float64, len(nonZeroIndices))
		for i, idx := range nonZeroIndices {
			filteredData[i] = data[idx]
		}
		data = filteredData

		// Process the filtered data
		corrections := detectAndCorrectMixups(filteredData)

		// Apply corrections to result using original indices
		for i, idx := range nonZeroIndices {
			if corrections[i] != 1.0 {
				result[idx].Open *= corrections[i]
				result[idx].High *= corrections[i]
				result[idx].Low *= corrections[i]
				result[idx].Close *= corrections[i]
				result[idx].AdjClose *= corrections[i]
				result[idx].Repaired = true
			}
		}

		return result
	}

	// No zeros, process all data
	corrections := detectAndCorrectMixups(data)

	for i := range result {
		if corrections[i] != 1.0 {
			result[i].Open *= corrections[i]
			result[i].High *= corrections[i]
			result[i].Low *= corrections[i]
			result[i].Close *= corrections[i]
			result[i].AdjClose *= corrections[i]
			result[i].Repaired = true
		}
	}

	return result
}

// detectAndCorrectMixups detects 100x errors using median filtering.
// Returns a slice of correction factors (1.0 means no correction, 0.01 or 100.0 for fixes).
func detectAndCorrectMixups(data [][]float64) []float64 {
	n := len(data)
	corrections := make([]float64, n)
	for i := range corrections {
		corrections[i] = 1.0
	}

	if n < 3 {
		return corrections
	}

	// Apply 2D median filter (3x3 window)
	median := stats.MedianFilter2D(data, 3)

	// Calculate ratio of actual to median
	for i := range data {
		for j := range data[i] {
			if median[i][j] == 0 {
				continue
			}

			ratio := data[i][j] / median[i][j]
			ratioRcp := 1.0 / ratio

			// Round ratio to nearest 20
			ratioRounded := math.Round(ratio/20) * 20
			ratioRcpRounded := math.Round(ratioRcp/20) * 20

			// Check if ratio is ~100 (either direction)
			if ratioRounded == 100 {
				// Price is 100x too high, need to divide by 100
				corrections[i] = 0.01
				break
			} else if ratioRcpRounded == 100 {
				// Price is 100x too low, need to multiply by 100
				corrections[i] = 100.0
				break
			}
		}
	}

	return corrections
}

// fixPricesSuddenChange detects and fixes a sudden permanent change in price level.
// This handles the case where Yahoo switches currency units at some date.
func (r *Repairer) fixPricesSuddenChange(bars []models.Bar, change float64) []models.Bar {
	if len(bars) < 3 {
		return bars
	}

	// Skip if change ratio is too close to 1.0
	if change > 0.8 && change < 1.25 {
		return bars
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)

	changeRcp := 1.0 / change

	// Calculate daily percentage changes using median of OHLC
	pctChanges := make([]float64, len(bars)-1)
	for i := 1; i < len(bars); i++ {
		prev := ohlcMedian(bars[i-1])
		curr := ohlcMedian(bars[i])
		if prev != 0 {
			pctChanges[i-1] = curr / prev
		} else {
			pctChanges[i-1] = 1.0
		}
	}

	// Use IQR to estimate normal volatility
	q1, q3, iqr := stats.IQR(pctChanges)
	if math.IsNaN(iqr) || iqr == 0 {
		return bars
	}

	// Filter outliers
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	var normalChanges []float64
	for _, pct := range pctChanges {
		if pct >= lowerBound && pct <= upperBound {
			normalChanges = append(normalChanges, pct)
		}
	}

	if len(normalChanges) == 0 {
		return bars
	}

	// Calculate mean and std of normal changes
	avg := stats.Mean(normalChanges)
	sd := stats.Std(normalChanges, 1)
	if math.IsNaN(sd) {
		sd = 0.01
	}

	// SD as percentage of mean
	sdPct := sd / avg

	// Only proceed if change far exceeds normal volatility
	largestChangePct := 5 * sdPct
	changeMax := math.Max(change, changeRcp)
	if changeMax < 1.0+largestChangePct {
		return bars
	}

	// Detect change points
	// Threshold is halfway between change ratio and largest normal change
	threshold := (changeMax + 1.0 + largestChangePct) * 0.5

	// Find where sudden change occurs
	for i := 1; i < len(bars); i++ {
		prev := ohlcMedian(result[i-1])
		curr := ohlcMedian(result[i])
		if prev == 0 {
			continue
		}

		dayChange := curr / prev

		// Check if this looks like a unit switch
		if dayChange >= threshold || dayChange <= 1.0/threshold {
			// Determine correction direction
			var correction float64
			if dayChange >= threshold {
				// Price jumped up, earlier prices need to be multiplied
				correction = change
			} else {
				// Price dropped, earlier prices need to be divided
				correction = changeRcp
			}

			// Apply correction to all bars before this point
			for j := 0; j < i; j++ {
				result[j].Open *= correction
				result[j].High *= correction
				result[j].Low *= correction
				result[j].Close *= correction
				result[j].AdjClose *= correction
				result[j].Volume = int64(float64(result[j].Volume) / correction)
				result[j].Repaired = true
			}

			// Continue checking rest of data for more switches
		}
	}

	return result
}

// UnitMixupStats contains statistics about unit mixup repairs.
type UnitMixupStats struct {
	TotalBars       int  // Total bars analyzed
	BarsRepaired    int  // Bars with 100x errors fixed
	HasUnitSwitch   bool // Whether a permanent unit switch was detected
	SwitchIndex     int  // Index where unit switch occurred (-1 if none)
	RandomMixupCount int // Number of random 100x errors found
}

// AnalyzeUnitMixups analyzes bars for 100x errors without modifying.
func (r *Repairer) AnalyzeUnitMixups(bars []models.Bar) UnitMixupStats {
	stats := UnitMixupStats{
		TotalBars:   len(bars),
		SwitchIndex: -1,
	}

	if len(bars) < 3 {
		return stats
	}

	// Extract OHLC data
	data := make([][]float64, len(bars))
	for i, bar := range bars {
		data[i] = []float64{bar.High, bar.Open, bar.Low, bar.Close, bar.AdjClose}
	}

	// Check for random mixups
	corrections := detectAndCorrectMixups(data)
	for _, c := range corrections {
		if c != 1.0 {
			stats.RandomMixupCount++
		}
	}

	stats.BarsRepaired = stats.RandomMixupCount

	return stats
}

// DetectUnitMixups checks if there are 100x errors in the data.
// Returns indices of bars with suspected 100x errors.
func DetectUnitMixups(bars []models.Bar) []int {
	var badIndices []int

	if len(bars) < 3 {
		return badIndices
	}

	// Extract OHLC data
	data := make([][]float64, len(bars))
	for i, bar := range bars {
		data[i] = []float64{bar.High, bar.Open, bar.Low, bar.Close, bar.AdjClose}
	}

	corrections := detectAndCorrectMixups(data)
	for i, c := range corrections {
		if c != 1.0 {
			badIndices = append(badIndices, i)
		}
	}

	return badIndices
}
