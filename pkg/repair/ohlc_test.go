package repair

import (
	"math"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestRepairInconsistentOHLCCloseAboveHigh(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 110, AdjClose: 110},
	}

	result := repairInconsistentOHLC(bars)

	bar := result[0]
	if !bar.Repaired {
		t.Fatal("expected Repaired=true for Close > High")
	}
	if bar.Close > bar.High || bar.Close < bar.Low {
		t.Errorf("Close %v still outside [Low %v, High %v]", bar.Close, bar.Low, bar.High)
	}
	if bar.Open != 100 || bar.Low != 98 {
		t.Errorf("good values must stay untouched: Open=%v Low=%v", bar.Open, bar.Low)
	}
}

func TestRepairInconsistentOHLCOpenBelowLow(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 50, High: 105, Low: 98, Close: 102, AdjClose: 102},
	}

	result := repairInconsistentOHLC(bars)

	bar := result[0]
	if !bar.Repaired {
		t.Fatal("expected Repaired=true for Open < Low")
	}
	if bar.Open > bar.High || bar.Open < bar.Low {
		t.Errorf("Open %v still outside [Low %v, High %v]", bar.Open, bar.Low, bar.High)
	}
	if bar.Close != 102 || bar.High != 105 {
		t.Errorf("good values must stay untouched: Close=%v High=%v", bar.Close, bar.High)
	}
}

func TestRepairInconsistentOHLCConsistentUntouched(t *testing.T) {
	bars := []models.Bar{
		{Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102},
		{Date: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Open: 0, High: 0, Low: 0, Close: 0, AdjClose: 0},
	}

	result := repairInconsistentOHLC(bars)

	for i, bar := range result {
		if bar.Repaired {
			t.Errorf("bar %d: expected no repair", i)
		}
	}
	if result[0].Close != 102 {
		t.Errorf("consistent bar changed: Close=%v", result[0].Close)
	}
}

// TestZeroRepairsKeepAdjCloseRatio checks that when zero/inconsistency
// repair rewrites Close, Adj Close moves with it (upstream treats Adj Close
// as corrupted with Close and rebuilds both), keeping the bar's dividend
// adjustment factor. Otherwise AutoAdjust, which now runs after repair,
// would map Close back to the unrepaired value.
func TestZeroRepairsKeepAdjCloseRatio(t *testing.T) {
	const ratio = 0.9
	day := func(d int) time.Time { return time.Date(2024, 1, d, 0, 0, 0, 0, time.UTC) }
	ratioOf := func(b models.Bar) float64 { return b.AdjClose / b.Close }

	bad := []models.Bar{{Date: day(1), Open: 100, High: 105, Low: 98, Close: 110, AdjClose: 110 * ratio}}
	got := repairInconsistentOHLC(bad)[0]
	if !got.Repaired || got.Close == 110 || math.Abs(ratioOf(got)-ratio) > 1e-12 {
		t.Errorf("inconsistent OHLC: %+v, Adj Close ratio %v, want %v", got, ratioOf(got), ratio)
	}
	// Unset Adj Close stays unset.
	bad[0].AdjClose = 0
	if got := repairInconsistentOHLC(bad)[0]; got.AdjClose != 0 {
		t.Errorf("unset Adj Close changed: %+v", got)
	}

	gap := []models.Bar{
		{Date: day(1), Open: 100, High: 105, Low: 98, Close: 102, AdjClose: 102 * ratio},
		{Date: day(2)},
		{Date: day(3), Open: 105, High: 110, Low: 103, Close: 108, AdjClose: 108 * ratio},
	}
	for name, idx := range map[string]int{"interpolate": 1} {
		got := repairZeroBar(gap, idx)
		if math.Abs(ratioOf(got)-ratio) > 1e-12 {
			t.Errorf("%s: Adj Close ratio %v, want %v (%+v)", name, ratioOf(got), ratio, got)
		}
	}
	if got := repairZeroBar(gap[1:], 0); math.Abs(ratioOf(got)-ratio) > 1e-12 {
		t.Errorf("backward fill: Adj Close ratio %v, want %v (%+v)", ratioOf(got), ratio, got)
	}
	if got := repairZeroBar(gap[:2], 1); math.Abs(ratioOf(got)-ratio) > 1e-12 {
		t.Errorf("forward fill: Adj Close ratio %v, want %v (%+v)", ratioOf(got), ratio, got)
	}
}
