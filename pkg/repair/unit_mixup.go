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

	return r.fixPricesSuddenChange(bars, currencySubUnitDivisor(r.opts.Currency))
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
			applyCellCorrections(&result[idx], corrections[i])
		}

		return result
	}

	// No zeros, process all data
	corrections := detectAndCorrectMixups(data)

	for i := range result {
		applyCellCorrections(&result[i], corrections[i])
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

// applyCellCorrections applies per-cell 100x corrections to a bar. The
// corrections slice is aligned with ohlcMatrix column order: High, Open, Low,
// Close, AdjClose. After a partial repair, Low/High are recalculated because
// Yahoo derived them from the corrupt cells (upstream #2908).
func applyCellCorrections(bar *models.Bar, corrections []float64) {
	fields := []*float64{&bar.High, &bar.Open, &bar.Low, &bar.Close, &bar.AdjClose}

	anyUp, anyDown := false, false
	for k, field := range fields {
		c := corrections[k]
		if c == 1.0 || !validPrice(c) {
			continue
		}
		*field *= c
		bar.Repaired = true
		if c > 1 {
			anyUp = true
		} else {
			anyDown = true
		}
	}

	if anyUp {
		bar.Low = math.Min(bar.Open, bar.Close)
	}
	if anyDown {
		bar.High = math.Max(bar.Open, bar.Close)
	}
}

// detectAndCorrectMixups detects 100x errors per cell using median filtering.
// Returns one correction factor per cell, aligned with the input matrix
// (1.0 means no correction, 0.01 or 100.0 for fixes). Per-cell granularity
// matters: Yahoo can corrupt only some columns of a bar (upstream ASAI.L
// fixture: Low/Close/Adj Close 100x low while Open/High are fine).
func detectAndCorrectMixups(data [][]float64) [][]float64 {
	n := len(data)
	corrections := make([][]float64, n)
	for i := range corrections {
		corrections[i] = make([]float64, len(data[i]))
		for j := range corrections[i] {
			corrections[i][j] = 1.0
		}
	}

	if n < 3 {
		return corrections
	}

	// Apply 2D median filter (3x3 window)
	median := stats.MedianFilter2D(data, 3)

	// Coarse pass: flag rows containing at least one cell whose ratio to the
	// local median rounds to ~100 in either direction.
	for i := range data {
		flagged := false
		for j := range data[i] {
			if median[i][j] == 0 {
				continue
			}

			ratio := data[i][j] / median[i][j]
			ratioRcp := 1.0 / ratio

			// Round ratio to nearest 20
			ratioRounded := math.Round(ratio/20) * 20
			ratioRcpRounded := math.Round(ratioRcp/20) * 20

			if ratioRounded == 100 || ratioRcpRounded == 100 {
				flagged = true
				break
			}
		}
		if !flagged {
			continue
		}

		// Refinement pass: within a flagged row, correct every cell decisively
		// on the 100x side of its local median (geometric midpoint of 1 and
		// 100 is 10). Cells near ratio 1 are genuine and stay untouched —
		// Yahoo can corrupt only some columns of a bar.
		for j := range data[i] {
			if median[i][j] == 0 {
				continue
			}
			ratio := data[i][j] / median[i][j]
			if ratio > 10 {
				// Price is 100x too high, need to divide by 100
				corrections[i][j] = 0.01
			} else if ratio < 0.1 {
				// Price is 100x too low, need to multiply by 100
				corrections[i][j] = 100.0
			}
		}
	}

	return corrections
}

// rowHasCorrection reports whether any cell of a row needs correcting.
func rowHasCorrection(corrections []float64) bool {
	for _, c := range corrections {
		if c != 1.0 {
			return true
		}
	}
	return false
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

	// Volume is required to tell a unit switch from a real corporate action
	// (upstream #2908: "No Volume data, cannot repair").
	vol := barVolumes(bars)
	if allVolumesZero(vol) {
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

	// Calculate mean and std of normal changes (population std, matching
	// upstream np.std's ddof=0 default).
	avg := stats.Mean(normalChanges)
	sd := stats.Std(normalChanges, 0)
	if math.IsNaN(sd) {
		sd = 0.01
	}

	// SD as percentage of mean
	sdPct := sd / avg

	// Only proceed if change far exceeds normal volatility. Coarser
	// intervals aggregate more price noise (upstream: x3 for interday
	// above 1d, x2 more for months — 5d is NOT interday upstream).
	largestChangePct := 5 * sdPct
	largestChangePct *= intervalNoiseMultiplier(r.opts.Interval)
	changeMax := math.Max(change, changeRcp)
	if changeMax < 1.0+largestChangePct {
		return bars
	}

	// Detect change points (upstream 8b85f90:
	// threshold = 1 + (split_max - 1 + largest_change_pct) * 0.6)
	threshold := 1 + (changeMax-1+largestChangePct)*0.6

	// Abnormal-volume threshold for the boundary cross-check
	volThreshold := volumeChangeThreshold(vol, changeMax, r.opts.Interval)

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
			if !unitSwitchVolumeOK(vol, i, dayChange >= threshold, volThreshold) {
				// Volume says this is a real corporate action, not a unit
				// switch — keep scanning for a genuine switch point.
				continue
			}
			correction := switchCorrection(dayChange, threshold, change, changeRcp)
			applyUnitSwitchCorrection(result[:i], correction)
			break
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

// applyUnitSwitchCorrection rescales prices and dividends by the same factor
// (upstream unit-switch call passes correct_dividend=True but does NOT pass
// correct_volume: a currency-unit switch does not change the share count, so
// Volume must stay untouched — unlike a genuine stock split).
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
		bars[j].Dividends *= correction
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
		if rowHasCorrection(c) {
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
		if rowHasCorrection(c) {
			badIndices = append(badIndices, i)
		}
	}

	return badIndices
}
