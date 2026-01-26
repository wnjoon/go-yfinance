package repair

import (
	"math"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/stats"
)

// repairCapitalGains fixes capital gains double-counting in Adjusted Close.
//
// Yahoo Finance sometimes double-counts capital gains for ETFs and Mutual Funds:
// 1. Capital Gains is added to the Dividends column
// 2. Adj Close is also adjusted for Capital Gains separately
// 3. Result: Capital Gains counted twice, historical prices appear too low
//
// This function detects and corrects this issue.
func (r *Repairer) repairCapitalGains(bars []models.Bar) []models.Bar {
	if len(bars) < 2 {
		return bars
	}

	// Check if we have capital gains data
	if !HasCapitalGains(bars) {
		return bars
	}

	// Get indices of capital gains events
	cgIndices := findCapitalGainsIndices(bars)
	if len(cgIndices) == 0 {
		return bars
	}

	// Calculate normal price volatility (excluding distribution days)
	normalVolatility := calculateNormalVolatility(bars, cgIndices)

	// Check each capital gains event for double-counting
	doubleCountDetected := 0
	totalCGEvents := 0

	for _, idx := range cgIndices {
		if idx == 0 {
			continue // Need previous day
		}

		totalCGEvents++
		isDoubleCount := detectDoubleCount(bars, idx, normalVolatility)
		if isDoubleCount {
			doubleCountDetected++
		}
	}

	// If >= 66.67% of CG events appear double-counted, repair all
	if totalCGEvents == 0 {
		return bars
	}

	ratio := float64(doubleCountDetected) / float64(totalCGEvents)
	if ratio < 0.6667 {
		return bars // Not enough evidence of double-counting
	}

	// Apply repair
	return applyCapitalGainsRepair(bars, cgIndices)
}

// findCapitalGainsIndices returns indices of bars with capital gains.
func findCapitalGainsIndices(bars []models.Bar) []int {
	var indices []int
	for i, bar := range bars {
		if bar.CapitalGains > 0 {
			indices = append(indices, i)
		}
	}
	return indices
}

// calculateNormalVolatility calculates the mean absolute percentage change
// on days without distributions (dividends or capital gains).
func calculateNormalVolatility(bars []models.Bar, cgIndices []int) float64 {
	// Create a set of distribution dates for quick lookup
	distDates := make(map[time.Time]bool)
	for _, idx := range cgIndices {
		distDates[bars[idx].Date] = true
	}
	for _, bar := range bars {
		if bar.Dividends > 0 {
			distDates[bar.Date] = true
		}
	}

	// Calculate price changes on non-distribution days
	var changes []float64
	for i := 1; i < len(bars); i++ {
		// Skip distribution days
		if distDates[bars[i].Date] {
			continue
		}

		// Calculate percentage change using Close prices
		if bars[i-1].Close != 0 {
			pctChange := (bars[i].Close - bars[i-1].Close) / bars[i-1].Close
			changes = append(changes, pctChange)
		}
	}

	if len(changes) == 0 {
		return 0.01 // Default to 1% if no data
	}

	// Return mean of absolute changes
	absChanges := stats.Abs(changes)
	return stats.Mean(absChanges)
}

// detectDoubleCount checks if a specific capital gains event shows evidence
// of double-counting.
func detectDoubleCount(bars []models.Bar, idx int, normalVolatility float64) bool {
	if idx == 0 || idx >= len(bars) {
		return false
	}

	bar := bars[idx]
	prevBar := bars[idx-1]

	if prevBar.Close == 0 || bar.Close == 0 {
		return false
	}

	// Calculate actual price drop (as percentage)
	priceDrop := (prevBar.Close - bar.Close) / prevBar.Close

	// Adjust for normal volatility
	adjustedDrop := priceDrop - normalVolatility

	// Calculate expected drop from dividend only
	dividendOnly := bar.Dividends
	if bar.CapitalGains > 0 && bar.Dividends >= bar.CapitalGains {
		// If dividend includes capital gains (double-count scenario),
		// true dividend would be: dividend - capital_gains
		dividendOnly = bar.Dividends - bar.CapitalGains
	}
	expectedDropDivOnly := dividendOnly / prevBar.Close

	// Calculate expected drop from dividend + capital gains
	expectedDropDivPlusCG := (bar.Dividends + bar.CapitalGains) / prevBar.Close

	// Compare which is closer to actual drop
	diffDivOnly := math.Abs(adjustedDrop - expectedDropDivOnly)
	diffDivPlusCG := math.Abs(adjustedDrop - expectedDropDivPlusCG)

	// If drop matches dividend-only better, it means CG was already counted
	// (i.e., double-counted)
	return diffDivOnly < diffDivPlusCG
}

// applyCapitalGainsRepair corrects the Adjusted Close values.
func applyCapitalGainsRepair(bars []models.Bar, cgIndices []int) []models.Bar {
	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Process each capital gains event (from newest to oldest)
	for i := len(cgIndices) - 1; i >= 0; i-- {
		idx := cgIndices[i]
		if idx == 0 {
			continue
		}

		bar := &result[idx]
		prevBar := result[idx-1]

		if bar.Close == 0 || prevBar.Close == 0 || bar.AdjClose == 0 || prevBar.AdjClose == 0 {
			continue
		}

		// Calculate the true dividend (removing double-counted CG)
		trueDividend := bar.Dividends
		if bar.Dividends >= bar.CapitalGains {
			trueDividend = bar.Dividends - bar.CapitalGains
		}

		// Calculate current adjustment factor (before -> after distribution)
		adjBefore := prevBar.AdjClose / prevBar.Close
		adjAfter := bar.AdjClose / bar.Close

		if adjAfter == 0 {
			continue
		}

		// Current adjustment ratio
		currentAdj := adjBefore / adjAfter

		// Correct adjustment should account for true dividend + capital gains (once)
		totalDistribution := trueDividend + bar.CapitalGains
		correctAdj := 1.0 - totalDistribution/prevBar.Close

		if correctAdj <= 0 || currentAdj <= 0 {
			continue
		}

		// Correction factor
		correction := correctAdj / currentAdj

		// Apply correction to all bars before this date
		for j := 0; j < idx; j++ {
			if result[j].AdjClose != 0 {
				result[j].AdjClose *= correction
				result[j].Repaired = true
			}
		}

		// Mark this bar as repaired
		bar.Repaired = true
	}

	return result
}

// CapitalGainsRepairStats returns statistics about capital gains repair.
type CapitalGainsRepairStats struct {
	TotalEvents        int     // Number of capital gains events
	DoubleCountEvents  int     // Number detected as double-counted
	DoubleCountRatio   float64 // Ratio of double-counted events
	RepairApplied      bool    // Whether repair was applied
	BarsRepaired       int     // Number of bars that were repaired
}

// AnalyzeCapitalGains analyzes bars for capital gains issues without modifying.
// Useful for debugging and understanding the data.
func (r *Repairer) AnalyzeCapitalGains(bars []models.Bar) CapitalGainsRepairStats {
	stats := CapitalGainsRepairStats{}

	if len(bars) < 2 || !HasCapitalGains(bars) {
		return stats
	}

	cgIndices := findCapitalGainsIndices(bars)
	stats.TotalEvents = len(cgIndices)

	if stats.TotalEvents == 0 {
		return stats
	}

	normalVolatility := calculateNormalVolatility(bars, cgIndices)

	for _, idx := range cgIndices {
		if idx == 0 {
			continue
		}
		if detectDoubleCount(bars, idx, normalVolatility) {
			stats.DoubleCountEvents++
		}
	}

	stats.DoubleCountRatio = float64(stats.DoubleCountEvents) / float64(stats.TotalEvents)
	stats.RepairApplied = stats.DoubleCountRatio >= 0.6667

	if stats.RepairApplied {
		// Count how many bars would be repaired
		for _, idx := range cgIndices {
			if idx > 0 {
				stats.BarsRepaired += idx
			}
		}
	}

	return stats
}
