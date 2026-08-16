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
