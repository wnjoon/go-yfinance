package repair

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// unadjustedOracleCases returns the corpus cases whose Adj Close equals
// Close on every row, i.e. frames without dividend adjustment.
func unadjustedOracleCases(t *testing.T, group string) []oracleCase {
	t.Helper()
	var out []oracleCase
	for _, c := range loadOracleCorpus(t, group) {
		ok := c.Bars.A != nil
		for i := range c.Bars.T {
			if !ok {
				break
			}
			a, cl := c.Bars.A[i], c.Bars.C[i]
			ok = a == cl && !math.IsNaN(a)
		}
		if ok {
			out = append(out, c)
		}
	}
	return out
}

// TestUnsetAdjCloseRepairsLikeUnadjusted pins that bars built without
// AdjClose (Go's zero value, e.g. by direct users of Repair) are repaired
// exactly like bars whose Adj Close equals Close. Upstream always has an
// Adj Close column; a zero AdjClose used to scale every dividend-adjusted
// price to 0 and silently disable split, unit-switch and level-shift
// detection.
func TestUnsetAdjCloseRepairsLikeUnadjusted(t *testing.T) {
	for _, group := range []string{"split", "unit_switch", "level_shift", "fixtures", "mined"} {
		group := group
		t.Run(group, func(t *testing.T) {
			cases := unadjustedOracleCases(t, group)
			if len(cases) == 0 {
				t.Skip("no unadjusted cases")
			}
			repairedCases, failed := 0, 0
			for _, c := range cases {
				loc := oracleLocation(t, c.TZ)
				want := c.wantBars(t, loc)
				for i := range want {
					want[i].AdjClose = 0
				}
				in := c
				in.Bars.A = make(nanFloats, len(c.Bars.T))
				got := runOracleCase(t, in, nil)
				if CountRepaired(want) > 0 {
					repairedCases++
				}
				if d := diffBars(got, want); d != "" {
					failed++
					t.Errorf("%s: %s", c.Name, d)
				}
			}
			t.Logf("%s: %d/%d unadjusted cases (%d repaired) match with AdjClose unset", group, len(cases)-failed, len(cases), repairedCases)
		})
	}
}

// TestDetectPriceShiftsAdjClose pins the adjustment factor used by the
// detector: unset (0) AdjClose means unadjusted, NaN keeps upstream's NaN
// propagation (that row's change counts as no change).
func TestDetectPriceShiftsAdjClose(t *testing.T) {
	mk := func(adj float64, closes ...float64) []models.Bar {
		out := make([]models.Bar, len(closes))
		for i, c := range closes {
			out[i] = models.Bar{Open: c, High: c, Low: c, Close: c, AdjClose: adj * c}
		}
		return out
	}
	// Newest first: the older rows are 10x the newer rows (an up signal).
	rows := mk(0, 10, 10, 100, 100)
	det := detectPriceShifts(rows, "1d", 10, false, 2)
	if det == nil || !det.up[0][2] {
		t.Fatalf("10x shift with AdjClose unset not detected: %+v", det)
	}
	adjusted := mk(1, 10, 10, 100, 100)
	want := detectPriceShifts(adjusted, "1d", 10, false, 2)
	if got, w := goMaskCells(det.up), goMaskCells(want.up); strings.Join(got, ",") != strings.Join(w, ",") {
		t.Fatalf("unset AdjClose masks %v, Adj Close = Close masks %v", got, w)
	}
	// NaN Adj Close: upstream's adj is NaN, so the changes into and out of
	// that row are NaN and denoise to 1 (no signal).
	rows = mk(1, 10, 10, 100, 100)
	rows[2].AdjClose = math.NaN()
	if det := detectPriceShifts(rows, "1d", 10, false, 2); det != nil && rowSignal(det.up, 2) {
		t.Fatalf("NaN Adj Close row should not signal: %+v", det)
	}
}

// TestRandomUnitMixupsUnsetAdjClose checks that sporadic 100x errors are
// still found when AdjClose is unset. Upstream skips rows with a zero price
// cell; a zero AdjClose here means "no Adj Close", so the row is checked as
// if Adj Close equalled Close, and AdjClose stays unset.
func TestRandomUnitMixupsUnsetAdjClose(t *testing.T) {
	mk := func(withAdj bool) []models.Bar {
		closes := []float64{10, 11, 10.5, 1050, 10.8, 11.2, 10.9}
		bars := make([]models.Bar, len(closes))
		for i, c := range closes {
			bars[i] = models.Bar{
				Date: time.Date(2024, 1, 1+i, 0, 0, 0, 0, time.UTC),
				Open: c * 0.99, High: c * 1.02, Low: c * 0.98, Close: c, Volume: 1000,
			}
			if withAdj {
				bars[i].AdjClose = c
			}
		}
		return bars
	}
	r := New(DefaultOptions())
	want := r.repairRandomUnitMixups(mk(true))
	got := r.repairRandomUnitMixups(mk(false))
	if !want[3].Repaired || math.Abs(want[3].Close-10.5) > 1e-9 {
		t.Fatalf("baseline not repaired: %+v", want[3])
	}
	for i := range want {
		w := want[i]
		w.AdjClose = 0
		if got[i] != w {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], w)
		}
	}
	if idx := DetectUnitMixups(mk(false)); len(idx) != 1 || idx[0] != 3 {
		t.Errorf("DetectUnitMixups with AdjClose unset = %v, want [3]", idx)
	}
}
