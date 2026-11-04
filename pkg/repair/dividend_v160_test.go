package repair

import (
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestIsDividendTooSmallUsesOpenCloseMove(t *testing.T) {
	status := dividendStatus{
		Dividend:   0.009,
		DivPct:     0.00009,
		PriceDrop:  1.0,
		Volatility: 0.1,
	}
	opts := Options{PrePost: true, Interval: "1h"}

	// Price recovered within the ex-div session (open ~ close): false positive.
	if isDividendTooSmall(status, 100.0, opts, 0.0) {
		t.Error("expected false when the session recovered (openCloseMove ~ 0)")
	}

	// Price stayed down through the session: genuine 100x-too-small dividend.
	if !isDividendTooSmall(status, 100.0, opts, 1.0) {
		t.Error("expected true when the session held the drop")
	}
}

func TestAnalyzeDividendPrePostGapDownNotTooSmall(t *testing.T) {
	// Gap-down open: prevClose-Close is large (old dayMove would pass), but the
	// bar itself recovered open-to-close, so v1.6.0 classifies it as noise.
	bars := make([]models.Bar, 8)
	for i := range bars {
		bars[i] = models.Bar{
			Date:     time.Date(2024, 1, 1, 9+i, 0, 0, 0, time.UTC),
			Open:     100,
			High:     100.1,
			Low:      99.9,
			Close:    100,
			AdjClose: 100,
			Volume:   1000,
		}
	}
	bars[4].Open = 99.0
	bars[4].Close = 99.0
	bars[4].Low = 98.95
	bars[4].High = 99.1
	bars[4].AdjClose = 99.0
	bars[4].Dividends = 0.009

	status := analyzeDividendWithOptions(bars, 4, 100.0, Options{PrePost: true, Interval: "1h"})

	if status.IsTooSmall {
		t.Error("expected IsTooSmall=false: the ex-div bar recovered open-to-close")
	}
}

func TestRepairDividendsTooSmallAndMissingAdj(t *testing.T) {
	// Dividend is 100x too small AND the adjustment is missing entirely:
	// the combined branch must scale the dividend first, then derive the
	// adjustment from the corrected value (upstream v1.6.0).
	bars := make([]models.Bar, 8)
	for i := range bars {
		bars[i] = models.Bar{
			Date:     time.Date(2024, 1, 1+i, 0, 0, 0, 0, time.UTC),
			Open:     100,
			High:     100.1,
			Low:      99.9,
			Close:    100,
			AdjClose: 100, // AdjClose == Close everywhere: adjustment missing
			Volume:   1000,
		}
	}
	bars[4].Open = 99.0
	bars[4].Close = 99.0
	bars[4].Low = 98.95
	bars[4].High = 99.1
	bars[4].AdjClose = 99.0
	bars[4].Dividends = 0.009 // should be 0.9

	repairer := New(DefaultOptions())
	result := repairer.repairDividends(bars)

	if math.Abs(result[4].Dividends-0.9) > 1e-9 {
		t.Errorf("expected dividend scaled to 0.9, got %v", result[4].Dividends)
	}

	// Adjustment derived from the corrected dividend: 1 - 0.9/100 = 0.991
	wantAdj := 100 * (1 - 0.9/100)
	if math.Abs(result[3].AdjClose-wantAdj) > 1e-9 {
		t.Errorf("expected pre-div AdjClose %v, got %v", wantAdj, result[3].AdjClose)
	}
	if !result[4].Repaired || !result[3].Repaired {
		t.Error("expected repaired flags on the dividend bar and earlier bars")
	}
}

// tinyDividendBars returns 380 flat daily bars with a correctly adjusted
// 0.005 dividend every 30 rows (0.005% yield, like NVDA's). The dividends
// listed in drops get a 0.6 ex-date drop, which on its own looks like a
// 100x-too-small dividend.
func tinyDividendBars(drops ...int) []models.Bar {
	const n, div = 380, 0.005
	bars := make([]models.Bar, n)
	for i := range bars {
		bars[i] = models.Bar{
			Date: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i),
			Open: 100, High: 100.05, Low: 99.95, Close: 100, Volume: 1000,
		}
	}
	for k := 0; k < 12; k++ {
		bars[10+30*k].Dividends = div
	}
	for _, k := range drops {
		bars[10+30*k].Low = 99.4
	}
	factor := 1.0
	for i := n - 1; i >= 0; i-- {
		bars[i].AdjClose = bars[i].Close * factor
		if bars[i].Dividends > 0 {
			factor *= 1 - bars[i].Dividends/bars[i-1].Close
		}
	}
	return bars
}

// TestRepairDividendsTooSmallClusterVote ports upstream's cluster check of
// div_too_small: with small dividends, price volatility can make a few of
// them look 100x too small (NVDA 2023-06/09), so a minority verdict within
// a cluster of similar dividends is discarded, and a majority applies to
// the whole cluster.
func TestRepairDividendsTooSmallClusterVote(t *testing.T) {
	r := New(DefaultOptions())
	count := func(bars []models.Bar) (repaired, scaled int) {
		for _, b := range bars {
			if b.Repaired {
				repaired++
			}
			if b.Dividends > 0.1 {
				scaled++
			}
		}
		return
	}

	// Sanity: each drop alone is flagged as too small.
	in := tinyDividendBars(3, 8)
	if st := analyzeDividend(in, 100, 100); !st.IsTooSmall || st.IsMissingAdj || st.AdjTooLarge {
		t.Fatalf("fixture: dividend 3 status %+v, want too small only", st)
	}

	// 2 of 12 flagged: false positives, nothing changes.
	if rep, scaled := count(r.repairDividends(in)); rep != 0 || scaled != 0 {
		t.Errorf("2/12 too small: %d bars repaired, %d dividends scaled, want none", rep, scaled)
	}
	// 7 of 12 flagged (>= 6/11): the whole cluster is too small.
	if _, scaled := count(r.repairDividends(tinyDividendBars(0, 1, 2, 3, 4, 5, 6))); scaled != 12 {
		t.Errorf("7/12 too small: %d dividends scaled, want 12", scaled)
	}
	// All flagged: kept.
	if _, scaled := count(r.repairDividends(tinyDividendBars(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11))); scaled != 12 {
		t.Errorf("12/12 too small: %d dividends scaled, want 12", scaled)
	}
}
