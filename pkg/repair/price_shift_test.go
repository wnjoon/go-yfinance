package repair

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

type reconcileOracleSample struct {
	rng             [2]int
	during, outside []float64
}

type reconcileOracleCase struct {
	name        string
	vol         []float64
	ranges      [][2]int
	up, down    []int
	threshold   float64
	unitSwitch  bool
	wantKept    [][2]int
	wantSamples []reconcileOracleSample
}

type thresholdOracleCase struct {
	name     string
	vol      []float64
	up, down []int
	change   float64
	interval string
	want     float64
	wantOK   bool
}

func signalMask(n int, idx []int) []bool {
	m := make([]bool, n)
	for _, i := range idx {
		m[i] = true
	}
	return m
}

func testRanges(spans [][2]int) []priceShiftRange {
	out := make([]priceShiftRange, len(spans))
	for i, s := range spans {
		out[i] = priceShiftRange{start: s[0], end: s[1], multiplier: 2, volumeMultiplier: 0.5}
	}
	return out
}

func rangeSpans(ranges []priceShiftRange) [][2]int {
	out := make([][2]int, len(ranges))
	for i, r := range ranges {
		out[i] = [2]int{r.start, r.end}
	}
	return out
}

// TestReconcilePriceShiftsWithVolumeOracle replays upstream-verified cases:
// the ported upstream tests (split vs unit switch, stdev fallback, missing
// volume), reverse-order deletion, pooling across ranges and gaps (including
// zero/negative volume and the exactly-10 quirk), fewer than 4 samples,
// threshold boundaries, and both signal directions for split and unit switch.
func TestReconcilePriceShiftsWithVolumeOracle(t *testing.T) {
	for _, tc := range reconcileOracleCases {
		t.Run(tc.name, func(t *testing.T) {
			n := len(tc.vol)
			up, down := signalMask(n, tc.up), signalMask(n, tc.down)
			ranges := testRanges(tc.ranges)
			before := append([]priceShiftRange(nil), ranges...)

			got := reconcilePriceShiftsWithVolume(tc.vol, ranges, up, down, tc.threshold, tc.unitSwitch)
			if !reflect.DeepEqual(rangeSpans(got), tc.wantKept) {
				t.Fatalf("kept %v, want %v", rangeSpans(got), tc.wantKept)
			}
			if !reflect.DeepEqual(ranges, before) {
				t.Fatalf("input ranges mutated: %v", ranges)
			}

			// Replay the visit order, deleting exactly what upstream deleted,
			// and check the pooled samples at every step.
			live := append([]priceShiftRange(nil), ranges...)
			kept := map[[2]int]bool{}
			for _, s := range tc.wantKept {
				kept[s] = true
			}
			visit := 0
			for i := len(live) - 1; i >= 0; i-- {
				during, outside := rangeVolumeSamples(tc.vol, live, i)
				if visit >= len(tc.wantSamples) {
					t.Fatalf("visited more ranges than upstream")
				}
				want := tc.wantSamples[visit]
				visit++
				if [2]int{live[i].start, live[i].end} != want.rng {
					t.Fatalf("visit %d: range %v, want %v", visit, live[i], want.rng)
				}
				if !reflect.DeepEqual(during, want.during) {
					t.Errorf("range %v during = %v, want %v", want.rng, during, want.during)
				}
				if !reflect.DeepEqual(outside, want.outside) {
					t.Errorf("range %v outside = %v, want %v", want.rng, outside, want.outside)
				}
				if !kept[want.rng] {
					live = append(live[:i], live[i+1:]...)
				}
			}
			if visit != len(tc.wantSamples) {
				t.Fatalf("visited %d ranges, upstream visited %d", visit, len(tc.wantSamples))
			}
		})
	}
}

// TestReconcileReverseOrderDeletionChangesPooling pins the 1.8.0 behaviour
// that a rejected range stops feeding samples: evaluating the same ranges
// without deleting would reject the first range too.
func TestReconcileReverseOrderDeletionChangesPooling(t *testing.T) {
	var tc reconcileOracleCase
	for _, c := range reconcileOracleCases {
		if c.name == "deletion" {
			tc = c
		}
	}
	n := len(tc.vol)
	up, down := signalMask(n, tc.up), signalMask(n, tc.down)
	ranges := testRanges(tc.ranges)

	// Without deletion the first range pools the later ranges and fails.
	during, outside := rangeVolumeSamples(tc.vol, ranges, 0)
	if rangeVolumeConfirms(denoiseVolume(during), denoiseVolume(outside), up[4], down[4], tc.threshold, false) {
		t.Fatal("fixture no longer distinguishes deletion order")
	}
	got := reconcilePriceShiftsWithVolume(tc.vol, ranges, up, down, tc.threshold, false)
	if !reflect.DeepEqual(rangeSpans(got), [][2]int{{4, 13}}) {
		t.Fatalf("kept %v, want only the first range", rangeSpans(got))
	}
}

func TestReconcileNoRanges(t *testing.T) {
	got := reconcilePriceShiftsWithVolume([]float64{1, 2, 3}, nil, make([]bool, 3), make([]bool, 3), 2, false)
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestRangeVolumeConfirmsStdDevFallback(t *testing.T) {
	tight := []float64{2000, 2010, 2000, 2010} // mean 2005, sd 5
	noisy := []float64{500, 1500, 500, 1500}   // mean 1000, sd 500
	// change ~2.0 misses the threshold of 3 in both directions, so only the
	// fallback can keep the range: |Δmean| = 1005 is 201 sd of the tight
	// sample and 2.01 sd of the noisy one. Either side may supply it.
	if !rangeVolumeConfirms(tight, noisy, false, true, 3, false) {
		t.Error("tight during-sample should satisfy the >15 sd fallback")
	}
	if !rangeVolumeConfirms(noisy, tight, true, false, 3, false) {
		t.Error("tight outside-sample should satisfy the >15 sd fallback")
	}
	// Both noisy: 2.01 sd on each side is not enough.
	noisy2 := []float64{1505, 2505, 1505, 2505}
	if rangeVolumeConfirms(noisy2, noisy, false, true, 3, false) {
		t.Error("fallback must not apply below 15 sd")
	}
	// Fewer than 4 values on either side disables the fallback.
	if rangeVolumeConfirms(tight[:3], noisy, false, true, 3, false) {
		t.Error("fallback must require at least 4 during values")
	}
	if rangeVolumeConfirms(tight, noisy[:3], false, true, 3, false) {
		t.Error("fallback must require at least 4 outside values")
	}
	// Unit switch never uses the fallback.
	if !rangeVolumeConfirms(tight, noisy, false, true, 3, true) {
		t.Error("unit switch without an inverse volume move must be kept")
	}
}

func TestVolumeStdDevMultipleZeroDeviation(t *testing.T) {
	flat := []float64{10, 10, 10, 10}
	// Equal constant samples: 0/0 is NaN on both sides, so no fallback.
	if got := volumeStdDevMultiple(flat, flat, 10, 10); !math.IsNaN(got) {
		t.Errorf("0/0 multiple = %v, want NaN", got)
	}
	// Different constant samples: x/0 is +Inf, so the fallback keeps them,
	// exactly like NumPy.
	other := []float64{20, 20, 20, 20}
	if got := volumeStdDevMultiple(flat, other, 10, 20); !math.IsInf(got, 1) {
		t.Errorf("x/0 multiple = %v, want +Inf", got)
	}
	// Python's max(nan, x) returns nan; max(x, nan) returns x.
	noisy := []float64{5, 15, 5, 15}
	if got := volumeStdDevMultiple(flat, noisy, 10, 10); !math.IsNaN(got) {
		t.Errorf("max(nan, 0) = %v, want NaN", got)
	}
	if got := volumeStdDevMultiple(noisy, flat, 10, 10); got != 0 {
		t.Errorf("max(0, nan) = %v, want 0", got)
	}
}

func TestDenoiseVolumeOracle(t *testing.T) {
	for _, tc := range denoiseOracleCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := denoiseVolume(tc.vol); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("denoise(%v) = %v, want %v", tc.vol, got, tc.want)
			}
		})
	}
}

func TestEstimateVolumeShiftThresholdOracle(t *testing.T) {
	for _, tc := range thresholdOracleCases {
		t.Run(tc.name, func(t *testing.T) {
			n := len(tc.vol)
			got, ok := estimateVolumeShiftThreshold(tc.vol, signalMask(n, tc.up), signalMask(n, tc.down), tc.change, tc.interval)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && !closeTo(got, tc.want, 1e-12) {
				t.Fatalf("threshold = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMapSignalsToRanges(t *testing.T) {
	up := signalMask(10, []int{6, 8})
	down := signalMask(10, []int{0, 2}) // position 0 is ignored
	for _, tc := range []struct {
		change  float64
		wantM   []float64
		wantVol []float64
	}{
		// Forward split: down -> 'split', up -> '1.0/split'.
		{2, []float64{2, 0.5}, []float64{0.5, 2}},
		// Reverse split: down -> '1.0/split', up -> 'split'.
		{0.25, []float64{4, 0.25}, []float64{0.25, 4}},
	} {
		got := mapSignalsToRanges(up, down, tc.change)
		if !reflect.DeepEqual(rangeSpans(got), [][2]int{{2, 6}, {8, 10}}) {
			t.Fatalf("change=%v spans %v", tc.change, rangeSpans(got))
		}
		for i, r := range got {
			if r.multiplier != tc.wantM[i] || r.volumeMultiplier != tc.wantVol[i] {
				t.Errorf("change=%v range %d: m=%v rcp=%v, want %v %v", tc.change, i, r.multiplier, r.volumeMultiplier, tc.wantM[i], tc.wantVol[i])
			}
		}
	}
	if got := mapSignalsToRanges(make([]bool, 4), signalMask(4, []int{0}), 2); len(got) != 0 {
		t.Fatalf("signal at position 0 produced %v", got)
	}
}

// splitScenarioBars builds an oldest-first daily series whose rows before
// splitIdx are quoted at oldPrice/oldVolume and from splitIdx on at
// newPrice/newVolume, with mild deterministic noise.
func splitScenarioBars(n, splitIdx int, split, oldPrice, newPrice float64, oldVolume, newVolume int64) []models.Bar {
	bars := make([]models.Bar, n)
	noise := []float64{1, 1.01, 0.99, 1.005, 0.995}
	vnoise := []float64{1, 1.04, 0.98, 1.01, 0.97}
	for i := range bars {
		p, v := oldPrice, oldVolume
		if i >= splitIdx {
			p, v = newPrice, newVolume
		}
		p *= noise[i%len(noise)]
		bars[i] = models.Bar{
			Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i),
			Open: p, High: p * 1.01, Low: p * 0.99, Close: p, AdjClose: p,
			Volume: int64(float64(v) * vnoise[i%len(vnoise)]),
		}
	}
	bars[splitIdx].Splits = split
	return bars
}

// TestRepairStockSplitsVolumeDirections covers both split directions end to
// end: the price signal is kept only when volume confirms it (inverse move or
// >15 sd shift); a flat-volume signal is rejected and nothing changes.
func TestRepairStockSplitsVolumeDirections(t *testing.T) {
	for _, tc := range []struct {
		name               string
		split              float64
		oldPrice, newPrice float64
		oldVol, newVol     int64
		wantRepaired       bool
		wantFactor         float64
	}{
		// 2:1 forward split not applied: old prices 2x too high (up
		// signal), old volume half the new volume.
		{"forward/volume confirms", 2, 100, 50, 1000, 2000, true, 0.5},
		{"forward/flat volume", 2, 100, 50, 1000, 1000, false, 1},
		// 1:4 reverse split not applied: old prices 4x too low (down
		// signal), old volume 4x the new volume.
		{"reverse/volume confirms", 0.25, 25, 100, 4000, 1000, true, 4},
		{"reverse/flat volume", 0.25, 25, 100, 1000, 1000, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bars := splitScenarioBars(30, 20, tc.split, tc.oldPrice, tc.newPrice, tc.oldVol, tc.newVol)
			got := New(DefaultOptions()).repairStockSplits(bars)
			for i := range got {
				want := bars[i]
				if tc.wantRepaired && i < 20 {
					want.Open *= tc.wantFactor
					want.High *= tc.wantFactor
					want.Low *= tc.wantFactor
					want.Close *= tc.wantFactor
					want.AdjClose *= tc.wantFactor
					want.Volume = int64(math.RoundToEven(float64(want.Volume) / tc.wantFactor))
					want.Repaired = true
				}
				if !closeTo(got[i].Close, want.Close, 1e-12) || got[i].Volume != want.Volume || got[i].Repaired != want.Repaired {
					t.Fatalf("row %d: got close=%v vol=%d repaired=%v, want close=%v vol=%d repaired=%v",
						i, got[i].Close, got[i].Volume, got[i].Repaired, want.Close, want.Volume, want.Repaired)
				}
			}
		})
	}
}

// TestRepairStockSplitsAlreadyAdjustedUnchanged: correctly adjusted data has
// no price signal, so volume (which legitimately jumps at the split) must not
// trigger any change.
func TestRepairStockSplitsAlreadyAdjustedUnchanged(t *testing.T) {
	bars := splitScenarioBars(30, 20, 2, 50, 50, 1000, 2000)
	got := New(DefaultOptions()).repairStockSplits(bars)
	if !reflect.DeepEqual(got, bars) {
		t.Fatal("already-adjusted data changed")
	}
}

// TestRepairStockSplitsSkipsMissingOHLCRows mirrors upstream dropping rows
// with missing prices before detection and restoring them untouched. Repaired
// row counts match upstream _fix_bad_stock_splits on the same series
// (including the NaN-on-split-day case, which upstream also leaves alone
// because the volume check then fails).
func TestRepairStockSplitsSkipsMissingOHLCRows(t *testing.T) {
	for _, tc := range []struct {
		nanRow       int
		wantRepaired int
	}{{10, 19}, {19, 19}, {20, 0}, {21, 20}} {
		bars := splitScenarioBars(30, 20, 2, 100, 50, 1000, 2000)
		nan := math.NaN()
		r := tc.nanRow
		bars[r].Open, bars[r].High, bars[r].Low, bars[r].Close, bars[r].AdjClose = nan, nan, nan, nan, nan
		got := New(DefaultOptions()).repairStockSplits(bars)
		if n := CountRepaired(got); n != tc.wantRepaired {
			t.Fatalf("nan row %d: repaired %d rows, want %d", r, n, tc.wantRepaired)
		}
		if got[r].Repaired || !math.IsNaN(got[r].Close) || got[r].Volume != bars[r].Volume {
			t.Fatalf("nan row %d touched: %+v", r, got[r])
		}
		for i := 0; i < 20 && tc.wantRepaired > 0; i++ {
			if i != r && (!got[i].Repaired || !closeTo(got[i].Close, bars[i].Close/2, 1e-12)) {
				t.Fatalf("nan row %d: row %d not repaired: %+v", r, i, got[i])
			}
		}
	}
}

// TestFixPricesSuddenChangeUnitSwitchVolume covers the unit-switch reconcile:
// an inverse volume move in either direction drops the candidate, while flat
// volume or volume moving with price keeps it.
func TestFixPricesSuddenChangeUnitSwitchVolume(t *testing.T) {
	for _, tc := range []struct {
		name               string
		oldPrice, newPrice float64
		oldVol, newVol     int64
		wantRepaired       bool
	}{
		{"price up, volume flat", 1, 100, 100000, 100000, true},
		{"price up, volume collapses", 1, 100, 100000, 1000, false},
		{"price up, volume rises", 1, 100, 1000, 100000, true},
		{"price down, volume flat", 100, 1, 100000, 100000, true},
		{"price down, volume surges", 100, 1, 1000, 100000, false},
		{"price down, volume falls", 100, 1, 100000, 1000, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bars := splitScenarioBars(30, 20, 0, tc.oldPrice, tc.newPrice, tc.oldVol, tc.newVol)
			got := New(DefaultOptions()).fixPricesSuddenChange(bars, 100)
			if n := CountRepaired(got); (n == 20) != tc.wantRepaired || (n != 0 && n != 20) {
				t.Fatalf("repaired %d rows, want repaired=%v", n, tc.wantRepaired)
			}
		})
	}
}
