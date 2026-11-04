package repair

import (
	"math"
	"sort"
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
	currencyDivide := currencySubUnitDivisor(r.opts.Currency)

	// Analyze every dividend first, as upstream does, so the too-small
	// verdicts can be cross-checked across similar dividends.
	statuses := make(map[int]dividendStatus, len(divIndices))
	var analyzed []dividendStatus
	for _, idx := range divIndices {
		if idx == 0 {
			continue
		}
		st := analyzeDividendWithOptions(result, idx, currencyDivide, r.opts)
		statuses[idx] = st
		analyzed = append(analyzed, st)
	}
	for _, st := range voteTooSmallByCluster(analyzed) {
		statuses[st.Index] = st
	}

	// Repair each dividend
	for _, idx := range divIndices {
		if idx == 0 {
			continue
		}

		status := statuses[idx]

		// Apply repairs based on analysis
		if status.IsMissingAdj && status.IsTooSmall {
			// Upstream v1.6.0 "too-small div and missing div-adjust": scale the
			// dividend up first so the adjustment is derived from the corrected
			// value, not the 100x-too-small one.
			result[idx].Dividends *= currencyDivide
			result = fixMissingDivAdj(result, idx)
		} else if status.IsMissingAdj {
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

// voteTooSmallByCluster ports upstream's cluster check of div_too_small in
// _fix_bad_div_adjust. Price volatility can make a small dividend look 100x
// too small, so dividends are clustered by yield (cluster_dividends) and,
// within a cluster where some but not all are flagged, the share flagged
// decides for the whole cluster: at least 6/11 (1.0 if any present
// adjustment exceeds its dividend) marks all too small, at most 1/2 (0.11)
// clears all, anything between keeps the individual verdicts.
func voteTooSmallByCluster(statuses []dividendStatus) []dividendStatus {
	out := append([]dividendStatus(nil), statuses...)
	if len(out) == 0 {
		return out
	}
	for _, cluster := range clusterDividendsByYield(out) {
		nFail, adjExceedsDiv := 0, false
		for _, i := range cluster {
			if out[i].IsTooSmall {
				nFail++
			}
			adjExceedsDiv = adjExceedsDiv || out[i].AdjTooLarge
		}
		if nFail == 0 || nFail == len(cluster) {
			continue
		}
		trueThreshold, falseThreshold := 1.0, 0.11
		if !adjExceedsDiv {
			// The adjustment confirms the dividends, so a too-small verdict
			// is more likely a false positive.
			trueThreshold, falseThreshold = 6.0/11, 0.5
		}
		pctFail := float64(nFail) / float64(len(cluster))
		switch {
		case pctFail >= trueThreshold:
			for _, i := range cluster {
				out[i].IsTooSmall = true
			}
		case pctFail <= falseThreshold:
			for _, i := range cluster {
				out[i].IsTooSmall = false
			}
		}
	}
	return out
}

// clusterDividendsByYield ports upstream's cluster_dividends on the '%'
// column: sorted by yield, a dividend joins the current cluster while its
// yield is less than 7x the cluster's mean. It returns index groups.
func clusterDividendsByYield(statuses []dividendStatus) [][]int {
	order := make([]int, len(statuses))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return statuses[order[a]].DivPct < statuses[order[b]].DivPct })
	var clusters [][]int
	current := []int{order[0]}
	sum := statuses[order[0]].DivPct
	for _, i := range order[1:] {
		if statuses[i].DivPct/(sum/float64(len(current))) < 7 {
			current = append(current, i)
			sum += statuses[i].DivPct
			continue
		}
		clusters = append(clusters, current)
		current, sum = []int{i}, statuses[i].DivPct
	}
	return append(clusters, current)
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
	return analyzeDividendWithOptions(bars, idx, currencyDivide, Options{})
}

func analyzeDividendWithOptions(bars []models.Bar, idx int, currencyDivide float64, opts Options) dividendStatus {
	status := dividendStatus{
		Index:    idx,
		Date:     bars[idx].Date,
		Dividend: bars[idx].Dividends,
	}

	if idx == 0 || idx >= len(bars) {
		return status
	}

	prevClose := bars[idx-1].Close
	if !validPrice(prevClose) || !validPrice(bars[idx].Low) || !validPrice(status.Dividend) {
		return status
	}

	// Calculate dividend percentage
	status.DivPct = status.Dividend / prevClose

	// Calculate price drop (close to low)
	status.PriceDrop = prevClose - bars[idx].Low
	// Intraday recovery is measured within the ex-div bar itself: open-to-close
	// (upstream v1.6.0: "price recovered by end of session").
	openCloseMove := bars[idx].Open - bars[idx].Close

	// Calculate typical volatility in surrounding window
	status.Volatility = calculateWindowVolatility(bars, idx)

	// Check if adjustment is missing
	status.IsMissingAdj = hasMissingDividendAdjustment(bars, idx, prevClose, status.DivPct)

	// Check if dividend is too large (100x error)
	status.IsTooLarge = isDividendTooLarge(status, currencyDivide)

	// Check if dividend is too small (0.01x error)
	status.IsTooSmall = isDividendTooSmall(status, currencyDivide, opts, openCloseMove)

	// Check for phantom dividend (duplicate within 7 days)
	status.IsPhantom = isPhantomDividend(bars, idx)

	// Check if present adjustment is too small/large
	if validPrice(bars[idx-1].AdjClose) && validPrice(bars[idx].AdjClose) && validPrice(bars[idx].Close) {
		presentAdj := (bars[idx-1].AdjClose / prevClose) / (bars[idx].AdjClose / bars[idx].Close)
		if !math.IsNaN(presentAdj) && !math.IsInf(presentAdj, 0) {
			impliedDivYield := 1.0 - presentAdj
			status.AdjTooSmall = impliedDivYield < (0.1 * status.DivPct)
			status.AdjTooLarge = impliedDivYield > (10 * status.DivPct)
		}
	}

	return status
}

// hasMissingDividendAdjustment reports whether Adj Close does not move
// across the ex-dividend date. Upstream tests pre_adj == post_adj; to allow
// for Yahoo's rounding, the factors may differ by less than a tenth of the
// dividend yield divPct. (A fixed tolerance flagged every correctly adjusted
// dividend below 0.1% of the price, so it was adjusted twice.)
func hasMissingDividendAdjustment(bars []models.Bar, idx int, prevClose, divPct float64) bool {
	if !validPrice(prevClose) || !validPrice(bars[idx].Close) ||
		!validPrice(bars[idx-1].AdjClose) || !validPrice(bars[idx].AdjClose) {
		return false
	}
	prevAdj := bars[idx-1].AdjClose / prevClose
	postAdj := bars[idx].AdjClose / bars[idx].Close
	return math.Abs(prevAdj/postAdj-1) < 0.1*divPct
}

func isDividendTooLarge(status dividendStatus, currencyDivide float64) bool {
	if status.DivPct <= 0.035 {
		return false
	}
	if status.PriceDrop <= 0 {
		return status.DivPct > 1.0
	}
	diff := math.Abs(status.Dividend - status.PriceDrop)
	diffFixed := math.Abs((status.Dividend / currencyDivide) - status.PriceDrop)
	return diffFixed*2 <= diff
}

func isDividendTooSmall(status dividendStatus, currencyDivide float64, opts Options, openCloseMove float64) bool {
	if math.IsNaN(status.Volatility) || status.Volatility <= 0 {
		return false
	}
	dropWoVol := status.PriceDrop - status.Volatility
	if dropWoVol <= 0 {
		return false
	}
	// Pre/post false-positive test: if the price recovered within the ex-div
	// session (small open-to-close move), the drop was noise, not a dividend.
	if opts.PrePost && isIntradayInterval(opts.Interval) && openCloseMove < 0.2*dropWoVol {
		return false
	}
	diff := math.Abs(status.Dividend - dropWoVol)
	diffFixed := math.Abs((status.Dividend * currencyDivide) - dropWoVol)
	return diffFixed <= diff && status.DivPct < 0.001
}

func isIntradayInterval(interval string) bool {
	switch interval {
	case "1m", "2m", "5m", "15m", "30m", "60m", "90m", "1h":
		return true
	default:
		return false
	}
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
	// An unset (0) or missing (NaN) Adj Close gives no present adjustment
	// to correct.
	if validPrice(prevClose) && validPrice(result[divIdx-1].AdjClose) {
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
	prev, cur := result[divIdx-1], result[divIdx]
	if validPrice(prev.Close) && validPrice(prev.AdjClose) && validPrice(cur.Close) && validPrice(cur.AdjClose) {
		presentAdj := (prev.AdjClose / prev.Close) / (cur.AdjClose / cur.Close)

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
	TotalDividends int            // Number of dividend events found
	MissingAdj     int            // Dividends with missing adjustment
	TooSmall       int            // Dividends 100x too small
	TooLarge       int            // Dividends 100x too big
	Phantoms       int            // Phantom (duplicate) dividends
	BarsRepaired   int            // Total bars modified
	Dividends      []DividendInfo // Details of each dividend
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

	currencyDivide := currencySubUnitDivisor(r.opts.Currency)

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

	currencyDivide := currencySubUnitDivisor(currency)

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
