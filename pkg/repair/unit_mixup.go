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

	data := ohlcMatrix(bars)

	if hasMatrixZeroes(data) {
		// Filter out rows with zeros for analysis
		// but keep them in result
		nonZeroIndices := nonZeroMatrixRows(data)
		if len(nonZeroIndices) < 3 {
			return bars // Not enough good data
		}

		// Process the filtered data
		corrections := detectAndCorrectMixups(filteredMatrix(data, nonZeroIndices))

		// Apply corrections to result using original indices
		for i, idx := range nonZeroIndices {
			applyUnitCorrection(&result[idx], corrections[i])
		}

		return result
	}

	// No zeros, process all data
	corrections := detectAndCorrectMixups(data)

	for i := range result {
		applyUnitCorrection(&result[i], corrections[i])
	}

	return result
}

func ohlcMatrix(bars []models.Bar) [][]float64 {
	data := make([][]float64, len(bars))
	for i, bar := range bars {
		data[i] = []float64{bar.High, bar.Open, bar.Low, bar.Close, bar.AdjClose}
	}
	return data
}

func hasMatrixZeroes(data [][]float64) bool {
	for i := range data {
		if rowHasZero(data[i]) {
			return true
		}
	}
	return false
}

func nonZeroMatrixRows(data [][]float64) []int {
	indices := make([]int, 0)
	for i := range data {
		if !rowHasZero(data[i]) {
			indices = append(indices, i)
		}
	}
	return indices
}

func rowHasZero(row []float64) bool {
	for _, value := range row {
		if value == 0 {
			return true
		}
	}
	return false
}

func filteredMatrix(data [][]float64, indices []int) [][]float64 {
	filteredData := make([][]float64, len(indices))
	for i, idx := range indices {
		filteredData[i] = data[idx]
	}
	return filteredData
}

func applyUnitCorrection(bar *models.Bar, correction float64) {
	if correction == 1.0 || !validPrice(correction) {
		return
	}
	bar.Open *= correction
	bar.High *= correction
	bar.Low *= correction
	bar.Close *= correction
	bar.AdjClose *= correction
	bar.Repaired = true
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
	pctChanges := dailyOHLCChanges(bars)

	// Use IQR to estimate normal volatility
	q1, q3, iqr := stats.IQR(pctChanges)
	if math.IsNaN(iqr) || iqr == 0 {
		return bars
	}

	// Filter outliers
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	normalChanges := boundedChanges(pctChanges, lowerBound, upperBound)

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
			correction := switchCorrection(dayChange, threshold, change, changeRcp)
			applyUnitSwitchCorrection(result[:i], correction)
		}
	}

	return result
}

func dailyOHLCChanges(bars []models.Bar) []float64 {
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
	return pctChanges
}

func boundedChanges(changes []float64, lowerBound, upperBound float64) []float64 {
	var normalChanges []float64
	for _, pct := range changes {
		if pct >= lowerBound && pct <= upperBound {
			normalChanges = append(normalChanges, pct)
		}
	}
	return normalChanges
}

func switchCorrection(dayChange, threshold, change, changeRcp float64) float64 {
	if dayChange >= threshold {
		return change
	}
	return changeRcp
}

func applyUnitSwitchCorrection(bars []models.Bar, correction float64) {
	if !validPrice(correction) {
		return
	}
	for j := range bars {
		bars[j].Open *= correction
		bars[j].High *= correction
		bars[j].Low *= correction
		bars[j].Close *= correction
		bars[j].AdjClose *= correction
		bars[j].Volume = int64(float64(bars[j].Volume) / correction)
		bars[j].Repaired = true
	}
}

// UnitMixupStats contains statistics about unit mixup repairs.
type UnitMixupStats struct {
	TotalBars        int  // Total bars analyzed
	BarsRepaired     int  // Bars with 100x errors fixed
	HasUnitSwitch    bool // Whether a permanent unit switch was detected
	SwitchIndex      int  // Index where unit switch occurred (-1 if none)
	RandomMixupCount int  // Number of random 100x errors found
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
