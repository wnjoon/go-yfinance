package repair

import (
	"math"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/stats"
)

// repairDividends fixes bad dividend adjustments.
//
// Yahoo Finance dividend data can have several issues:
// 1. Missing dividend adjustment (Adj Close = Close)
// 2. Dividend 100x too big (currency unit error)
// 3. Dividend 100x too small (currency unit error)
// 4. Duplicate/phantom dividends (within 7 days)
// 5. Wrong ex-dividend date (price drop is days/weeks after)
//
// Algorithm:
// 1. For each dividend event, calculate expected vs actual price drop
// 2. Compare with typical volatility to identify anomalies
// 3. Determine if dividend is too big, too small, or adjustment is missing
// 4. Apply corrections to Adj Close and/or Dividends
func (r *Repairer) repairDividends(bars []models.Bar) []models.Bar {
	if len(bars) < 3 {
		return bars
	}

	// Skip weekly/monthly intervals - too volatile for reliable detection
	interval := r.opts.Interval
	if interval == "1wk" || interval == "1mo" || interval == "3mo" {
		return bars
	}

	// Skip if capital gains present (complex interaction)
	if HasCapitalGains(bars) {
		return bars
	}

	// Find dividend events
	divIndices := findDividendIndices(bars)
	if len(divIndices) == 0 {
		return bars
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Determine currency divisor
	currencyDivide := 100.0
	if r.opts.Currency == "KWF" {
		currencyDivide = 1000.0
	}

	// Analyze each dividend
	for _, idx := range divIndices {
		if idx == 0 {
			continue
		}

		status := analyzeDividend(result, idx, currencyDivide)

		// Apply repairs based on analysis
		if status.IsMissingAdj {
			result = fixMissingDivAdj(result, idx)
		} else if status.IsTooSmall {
			result = fixDivTooSmall(result, idx, currencyDivide)
		} else if status.IsTooLarge {
			result = fixDivTooLarge(result, idx, currencyDivide)
		} else if status.IsPhantom {
			result = removePhantomDiv(result, idx)
		}
	}

	return result
}

// dividendStatus contains analysis results for a dividend event.
type dividendStatus struct {
	Index        int
	Date         time.Time
	Dividend     float64
	DivPct       float64 // Dividend as % of prev close
	PriceDrop    float64 // Close-to-Low drop
	Volatility   float64 // Typical price volatility
	IsMissingAdj bool    // Adj Close = Close (no adjustment)
	IsTooSmall   bool    // Dividend 100x too small
	IsTooLarge   bool    // Dividend 100x too big
	IsPhantom    bool    // Duplicate dividend
	AdjTooSmall  bool    // Present adjustment is too small
	AdjTooLarge  bool    // Present adjustment is too large
}

// findDividendIndices returns indices of bars with dividends.
func findDividendIndices(bars []models.Bar) []int {
	var indices []int
	for i, bar := range bars {
		if bar.Dividends > 0 {
			indices = append(indices, i)
		}
	}
	return indices
}

// analyzeDividend analyzes a single dividend event.
func analyzeDividend(bars []models.Bar, idx int, currencyDivide float64) dividendStatus {
	status := dividendStatus{
		Index:    idx,
		Date:     bars[idx].Date,
		Dividend: bars[idx].Dividends,
	}

	if idx == 0 || idx >= len(bars) {
		return status
	}

	prevClose := bars[idx-1].Close
	if prevClose == 0 {
		return status
	}

	// Calculate dividend percentage
	status.DivPct = status.Dividend / prevClose

	// Calculate price drop (close to low)
	status.PriceDrop = prevClose - bars[idx].Low

	// Calculate typical volatility in surrounding window
	status.Volatility = calculateWindowVolatility(bars, idx)

	// Check if adjustment is missing
	if prevClose > 0 && bars[idx].Close > 0 {
		prevAdj := bars[idx-1].AdjClose / prevClose
		postAdj := bars[idx].AdjClose / bars[idx].Close
		if math.Abs(prevAdj-postAdj) < 0.001 {
			status.IsMissingAdj = true
		}
	}

	// Check if dividend is too large (100x error)
	if status.DivPct > 0.035 { // Only check if significant dividend
		if status.PriceDrop > 0 {
			diff := math.Abs(status.Dividend - status.PriceDrop)
			diffFixed := math.Abs((status.Dividend/currencyDivide) - status.PriceDrop)
			if diffFixed*2 <= diff {
				status.IsTooLarge = true
			}
		} else if status.DivPct > 1.0 {
			// Dividend > 100% of price, almost certainly wrong
			status.IsTooLarge = true
		}
	}

	// Check if dividend is too small (0.01x error)
	if !math.IsNaN(status.Volatility) && status.Volatility > 0 {
		dropWoVol := status.PriceDrop - status.Volatility
		if dropWoVol > 0 {
			diff := math.Abs(status.Dividend - dropWoVol)
			diffFixed := math.Abs((status.Dividend*currencyDivide) - dropWoVol)
			if diffFixed*1 <= diff && status.DivPct < 0.001 {
				status.IsTooSmall = true
			}
		}
	}

	// Check for phantom dividend (duplicate within 7 days)
	status.IsPhantom = isPhantomDividend(bars, idx)

	// Check if present adjustment is too small/large
	presentAdj := (bars[idx-1].AdjClose / prevClose) / (bars[idx].AdjClose / bars[idx].Close)
	impliedDivYield := 1.0 - presentAdj
	status.AdjTooSmall = impliedDivYield < (0.1 * status.DivPct)
	status.AdjTooLarge = impliedDivYield > (10 * status.DivPct)

	return status
}

// calculateWindowVolatility calculates typical price volatility in a window around idx.
func calculateWindowVolatility(bars []models.Bar, idx int) float64 {
	start := maxInt(0, idx-4)
	end := minInt(len(bars), idx+4)

	if end-start < 4 {
		return math.NaN()
	}

	var diffs []float64
	for i := start; i < end-1; i++ {
		diff := bars[i].Close - bars[i+1].Low
		diffs = append(diffs, math.Abs(diff))
	}

	if len(diffs) == 0 {
		return math.NaN()
	}

	return stats.Mean(diffs)
}

// isPhantomDividend checks if dividend is a duplicate within 17 days.
func isPhantomDividend(bars []models.Bar, idx int) bool {
	const proximityDays = 17
	div := bars[idx].Dividends
	divDate := bars[idx].Date

	// Check nearby dividends
	for i, bar := range bars {
		if i == idx || bar.Dividends == 0 {
			continue
		}

		daysDiff := math.Abs(divDate.Sub(bar.Date).Hours() / 24)
		if daysDiff <= proximityDays {
			ratio := div / bar.Dividends
			if math.Abs(ratio-1.0) < 0.08 {
				// Similar dividend within proximity - one is phantom
				// The one with smaller price drop is phantom
				drop := bars[idx-1].Close - bars[idx].Low
				otherIdx := i
				if otherIdx > 0 {
					otherDrop := bars[otherIdx-1].Close - bars[otherIdx].Low
					if drop < otherDrop*0.66 {
						return true
					}
				}
			}
		}
	}

	return false
}

// fixMissingDivAdj fixes missing dividend adjustment.
func fixMissingDivAdj(bars []models.Bar, divIdx int) []models.Bar {
	result := make([]models.Bar, len(bars))
	copy(result, bars)

	div := result[divIdx].Dividends
	prevClose := result[divIdx-1].Close
	if prevClose == 0 {
		return result
	}

	// Calculate true adjustment factor
	trueAdj := 1.0 - div/prevClose

	// Apply adjustment to all bars before dividend
	for i := 0; i < divIdx; i++ {
		result[i].AdjClose *= trueAdj
		result[i].Repaired = true
	}

	result[divIdx].Repaired = true
	return result
}

// fixDivTooSmall fixes dividend that is 100x too small.
func fixDivTooSmall(bars []models.Bar, divIdx int, currencyDivide float64) []models.Bar {
	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Multiply dividend by 100
	result[divIdx].Dividends *= currencyDivide
	result[divIdx].Repaired = true

	// Recalculate adjustment
	div := result[divIdx].Dividends
	prevClose := result[divIdx-1].Close
	if prevClose > 0 {
		trueAdj := 1.0 - div/prevClose
		for i := 0; i < divIdx; i++ {
			result[i].AdjClose *= trueAdj
			result[i].Repaired = true
		}
	}

	return result
}

// fixDivTooLarge fixes dividend that is 100x too big.
func fixDivTooLarge(bars []models.Bar, divIdx int, currencyDivide float64) []models.Bar {
	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Divide dividend by 100
	result[divIdx].Dividends /= currencyDivide
	result[divIdx].Repaired = true

	// Recalculate adjustment
	div := result[divIdx].Dividends
	prevClose := result[divIdx-1].Close
	if prevClose > 0 {
		trueAdj := 1.0 - div/prevClose
		currentAdj := result[divIdx-1].AdjClose / prevClose

		// Only fix if adjustment is significantly different
		if math.Abs(currentAdj-trueAdj) > 0.01 {
			adjustmentRatio := trueAdj / currentAdj
			for i := 0; i < divIdx; i++ {
				result[i].AdjClose *= adjustmentRatio
				result[i].Repaired = true
			}
		}
	}

	return result
}

// removePhantomDiv removes a phantom (duplicate) dividend.
func removePhantomDiv(bars []models.Bar, divIdx int) []models.Bar {
	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Remove the adjustment for this dividend
	prevClose := result[divIdx-1].Close
	if prevClose > 0 {
		presentAdj := (result[divIdx-1].AdjClose / prevClose) / (result[divIdx].AdjClose / result[divIdx].Close)

		// Reverse the adjustment
		for i := 0; i < divIdx; i++ {
			result[i].AdjClose /= presentAdj
			result[i].Repaired = true
		}
	}

	// Zero out the phantom dividend
	result[divIdx].Dividends = 0
	result[divIdx].Repaired = true

	return result
}

// DividendRepairStats contains statistics about dividend repairs.
type DividendRepairStats struct {
	TotalDividends   int       // Number of dividend events found
	MissingAdj       int       // Dividends with missing adjustment
	TooSmall         int       // Dividends 100x too small
	TooLarge         int       // Dividends 100x too big
	Phantoms         int       // Phantom (duplicate) dividends
	BarsRepaired     int       // Total bars modified
	Dividends        []DividendInfo // Details of each dividend
}

// DividendInfo contains information about a single dividend.
type DividendInfo struct {
	Date         time.Time
	Amount       float64
	IsMissingAdj bool
	IsTooSmall   bool
	IsTooLarge   bool
	IsPhantom    bool
	WasRepaired  bool
}

// AnalyzeDividends analyzes bars for dividend issues without modifying.
func (r *Repairer) AnalyzeDividends(bars []models.Bar) DividendRepairStats {
	result := DividendRepairStats{}

	divIndices := findDividendIndices(bars)
	result.TotalDividends = len(divIndices)

	currencyDivide := 100.0
	if r.opts.Currency == "KWF" {
		currencyDivide = 1000.0
	}

	for _, idx := range divIndices {
		info := DividendInfo{
			Date:   bars[idx].Date,
			Amount: bars[idx].Dividends,
		}

		if idx > 0 {
			status := analyzeDividend(bars, idx, currencyDivide)
			info.IsMissingAdj = status.IsMissingAdj
			info.IsTooSmall = status.IsTooSmall
			info.IsTooLarge = status.IsTooLarge
			info.IsPhantom = status.IsPhantom

			if status.IsMissingAdj {
				result.MissingAdj++
			}
			if status.IsTooSmall {
				result.TooSmall++
			}
			if status.IsTooLarge {
				result.TooLarge++
			}
			if status.IsPhantom {
				result.Phantoms++
			}
		}

		result.Dividends = append(result.Dividends, info)
	}

	return result
}

// DetectBadDividends checks if there are dividend issues in the data.
// Returns indices of bars with suspected dividend problems.
func DetectBadDividends(bars []models.Bar, currency string) []int {
	var badIndices []int

	divIndices := findDividendIndices(bars)

	currencyDivide := 100.0
	if currency == "KWF" {
		currencyDivide = 1000.0
	}

	for _, idx := range divIndices {
		if idx == 0 {
			continue
		}

		status := analyzeDividend(bars, idx, currencyDivide)
		if status.IsMissingAdj || status.IsTooSmall || status.IsTooLarge || status.IsPhantom {
			badIndices = append(badIndices, idx)
		}
	}

	return badIndices
}
