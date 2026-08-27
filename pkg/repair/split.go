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
	if r.opts.Interval != "1d" && r.opts.Interval != "1wk" && r.opts.Interval != "1mo" && r.opts.Interval != "3mo" {
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
		result = repairSplitAtIndex(result, idx, splitRatio, r.opts.Interval)
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
func repairSplitAtIndex(bars []models.Bar, splitIdx int, splitRatio float64, interval string) []models.Bar {
	if splitIdx == 0 || splitRatio <= 0 || (splitRatio > 0.8 && splitRatio < 1.25) {
		return bars
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Match upstream's per-split scope: all older data plus a short look-ahead.
	// The look-ahead is essential when corruption starts or stops near a split.
	lookAhead := 5
	if interval == "1wk" || interval == "1mo" || interval == "3mo" {
		lookAhead = 1
	}
	cutoff := minInt(len(result), splitIdx+lookAhead+1)
	work := result[:cutoff]
	if len(work) < 3 {
		return bars
	}
	if splitVolumesAllZero(work) {
		return bars
	}

	// Upstream scans newest -> oldest. A signal marks a boundary between ranges,
	// rather than implying that every row before the split needs correction.
	ratio := make([]float64, len(work))
	ratio[0] = 1
	for i := 1; i < len(work); i++ {
		newer := adjustedOHLCMedian(work[len(work)-i])
		older := adjustedOHLCMedian(work[len(work)-1-i])
		if newer == 0 || older == 0 {
			ratio[i] = 1
		} else {
			ratio[i] = older / newer
		}
	}

	// Use IQR to estimate normal volatility
	q1, q3, iqr := stats.IQR(ratio)
	if math.IsNaN(iqr) {
		return bars
	}

	// Filter out outliers to get "normal" volatility
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	normalChanges := splitBoundedChanges(ratio, lowerBound, upperBound)
	if len(normalChanges) == 0 {
		return bars
	}

	// Calculate standard deviation of normal changes (population std,
	// matching upstream np.std's ddof=0 default).
	mean := stats.Mean(normalChanges)
	stdDev := stats.Std(normalChanges, 0)
	if math.IsNaN(stdDev) || mean == 0 {
		return bars
	}

	// Split repair shares its detection formula with unit-switch repair
	// upstream (both call the same _fix_prices_sudden_change with `change` =
	// the split ratio): threshold = 1 + (splitMax-1+largestNormalChange)*0.6,
	// compared against the ratio observed on the split date — not a
	// symmetric distance band around a single expected value.
	splitMax := math.Max(splitRatio, 1.0/splitRatio)
	largestNormalChange := 5 * stdDev / mean
	if interval != "1d" {
		largestNormalChange *= 3
	}
	if interval == "1mo" || interval == "3mo" {
		largestNormalChange *= 2
	}
	if splitMax < 1+largestNormalChange {
		return bars
	}
	threshold := 1 + (splitMax-1+largestNormalChange)*0.6
	up, down := make([]bool, len(ratio)), make([]bool, len(ratio))
	for i, v := range ratio {
		up[i] = v > threshold
		down[i] = v < 1/threshold
	}
	up[0], down[0] = false, false
	suppressSplitVolumeSpikes(work, up, down)
	suppressLocalVolatility(ratio, up, down, splitMax, interval)
	result = applySplitRanges(result, cutoff, splitRatio, up, down)

	return result
}

func adjustedOHLCMedian(bar models.Bar) float64 {
	if bar.Close == 0 {
		return ohlcMedian(bar)
	}
	adj := bar.AdjClose / bar.Close
	if math.IsNaN(adj) || math.IsInf(adj, 0) || adj == 0 {
		adj = 1
	}
	return stats.OHLCMedian(bar.Open*adj, bar.High*adj, bar.Low*adj, bar.Close*adj)
}

// suppressLocalVolatility applies upstream's second, candidate-local 0.5
// threshold. This prevents a globally unusual but locally ordinary move from
// being mistaken for a missing split adjustment.
func suppressLocalVolatility(ratio []float64, up, down []bool, splitMax float64, interval string) {
	for idx := range ratio {
		if !up[idx] && !down[idx] {
			continue
		}
		lookback := 3
		if len(interval) > 0 && interval[len(interval)-1] == 'd' {
			lookback = 10
		} else if len(interval) > 0 && interval[len(interval)-1] == 'm' {
			lookback = 100
		}
		start, end := maxInt(0, idx-lookback), minInt(len(ratio), idx+2)
		clean := make([]float64, 0, end-start)
		for j := start; j < end; j++ {
			if !up[j] && !down[j] {
				clean = append(clean, ratio[j])
			}
		}
		if len(clean) == 0 {
			continue
		}
		avg, sd := stats.Mean(clean), stats.Std(clean, 0)
		if avg == 0 || math.IsNaN(sd) {
			continue
		}
		localThreshold := 1 + (splitMax-1+5*sd/avg*intervalNoiseMultiplier(interval))*0.5
		if ratio[idx] < localThreshold && ratio[idx] > 1/localThreshold {
			up[idx], down[idx] = false, false
		}
	}
}

// suppressSplitVolumeSpikes ports the practical false-positive guard in
// Python 1.7.0: a catastrophic price drop on abnormally high volume is likely
// a real market event. Signals are newest-first, matching upstream df2.
func suppressSplitVolumeSpikes(work []models.Bar, up, down []bool) {
	n := len(work)
	vol := make([]float64, n)
	for i := range work {
		vol[i] = float64(work[n-1-i].Volume)
	}
	for idx := range up {
		if !up[idx] || idx == 0 {
			continue
		}
		dropIdx := idx - 1
		block := make([]float64, 0, 30)
		firstAfter := math.NaN()
		// Stop at the nearest newer down-signal, matching upstream's
		// block.loc[dt+1 : next_down_dt-1] exclusion.
		newerDown := -1
		for j := idx - 1; j >= 0; j-- {
			if down[j] {
				newerDown = j
				break
			}
		}
		for j := dropIdx - 1; j > newerDown && len(block) < 30; j-- {
			if up[j] || down[j] || vol[j] <= 0 {
				continue
			}
			if math.IsNaN(firstAfter) {
				firstAfter = vol[j]
			}
			block = append(block, vol[j])
		}
		if len(block) < 2 {
			continue
		}
		z := splitVolumeZScore(vol[dropIdx], block)
		zFirst := splitVolumeZScore(firstAfter, block)
		if math.Max(z, zFirst) > 2 {
			up[idx] = false
		}
	}
}

func splitVolumeZScore(v float64, sample []float64) float64 {
	if len(sample) < 2 || math.IsNaN(v) {
		return 0
	}
	mean, sd := stats.Mean(sample), stats.Std(sample, 1)
	if sd == 0 || math.IsNaN(sd) {
		return 0
	}
	return (v - mean) / sd
}

func splitVolumesAllZero(bars []models.Bar) bool {
	for _, b := range bars {
		if b.Volume != 0 {
			return false
		}
	}
	return true
}

func splitBoundedChanges(changes []float64, lower, upper float64) []float64 {
	var out []float64
	for _, v := range changes {
		if v >= lower && v <= upper {
			out = append(out, v)
		}
	}
	return out
}

// applySplitRanges ports upstream's map_signals_to_ranges. Signal indexes are
// newest-first; only alternating ranges are corrected, preserving already-good
// islands (the NRDY regression contains several such islands).
func applySplitRanges(bars []models.Bar, cutoff int, split float64, up, down []bool) []models.Bar {
	indices := make([]int, 0)
	for i := range up {
		if up[i] || down[i] {
			indices = append(indices, i)
		}
	}
	for pair := 0; pair < len(indices); pair += 2 {
		start, end := indices[pair], len(up)
		if pair+1 < len(indices) {
			end = indices[pair+1]
		}
		factor := split
		if (split > 1 && up[start]) || (split < 1 && down[start]) {
			factor = 1 / split
		}
		for di := start; di < end; di++ {
			i := cutoff - 1 - di
			bars[i].Open *= factor
			bars[i].High *= factor
			bars[i].Low *= factor
			bars[i].Close *= factor
			bars[i].AdjClose *= factor
			bars[i].Dividends *= factor
			bars[i].Volume = int64(math.RoundToEven(float64(bars[i].Volume) / factor))
			bars[i].Repaired = true
		}
	}
	return bars
}

func splitWindowChanges(bars []models.Bar, startIdx, splitIdx, windowSize int) []float64 {
	pctChanges := make([]float64, 0, windowSize)
	for i := startIdx + 1; i <= splitIdx; i++ {
		prevPrice := ohlcMedian(bars[i-1])
		currPrice := ohlcMedian(bars[i])
		if prevPrice != 0 {
			pctChanges = append(pctChanges, (currPrice-prevPrice)/prevPrice)
		}
	}
	return pctChanges
}

func absBoundedChanges(changes []float64, lowerBound, upperBound float64) []float64 {
	var normalChanges []float64
	for _, pct := range changes {
		if pct >= lowerBound && pct <= upperBound {
			normalChanges = append(normalChanges, math.Abs(pct))
		}
	}
	return normalChanges
}

// expectedSplitChange returns the signed price change an UNADJUSTED split
// produces on the split date: a 2:1 forward split halves the price (-0.5), a
// 1:4 reverse split quadruples it (+3.0). Matches DetectBadSplits.
func expectedSplitChange(splitRatio float64) float64 {
	if splitRatio > 1 {
		return -(1.0 - 1.0/splitRatio)
	}
	return 1.0/splitRatio - 1.0
}

func splitDateChange(bars []models.Bar, splitIdx int) float64 {
	if splitIdx == 0 {
		return 0
	}
	prevPrice := ohlcMedian(bars[splitIdx-1])
	currPrice := ohlcMedian(bars[splitIdx])
	if prevPrice == 0 {
		return 0
	}
	return (currPrice - prevPrice) / prevPrice
}

// applySplitCorrection adjusts historical prices, dividends, and volume for a
// split. Dividends scale by the same factor as prices, and volume by the
// reciprocal factor (upstream: correct_dividend=True, correct_volume=True are
// both passed to the shared _fix_prices_sudden_change call for split repair).
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
			result[i].Dividends /= splitRatio
			result[i].Volume = int64(float64(result[i].Volume) * splitRatio)
		} else if splitRatio > 0 {
			// Reverse split: historical prices should be higher
			result[i].Open *= (1 / splitRatio)
			result[i].High *= (1 / splitRatio)
			result[i].Low *= (1 / splitRatio)
			result[i].Close *= (1 / splitRatio)
			result[i].AdjClose *= (1 / splitRatio)
			result[i].Dividends *= (1 / splitRatio)
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
