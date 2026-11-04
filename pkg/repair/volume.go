package repair

import (
	"math"

	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/stats"
)

// barVolumes extracts the volume series as floats.
func barVolumes(bars []models.Bar) []float64 {
	vol := make([]float64, len(bars))
	for i, bar := range bars {
		vol[i] = float64(bar.Volume)
	}
	return vol
}

func allVolumesZero(vol []float64) bool {
	for _, v := range vol {
		if v != 0 {
			return false
		}
	}
	return true
}

// intervalNoiseMultiplier scales an estimated "normal" volatility for coarser
// intervals. Matches upstream's interday definition exactly:
// interday = interval in {1d, 1wk, 1mo, 3mo} — note "5d" is NOT interday and
// gets no multiplier — with an extra factor for monthly bars.
func intervalNoiseMultiplier(interval string) float64 {
	switch interval {
	case "1wk":
		return 3
	case "1mo", "3mo":
		return 6
	default:
		return 1
	}
}

// denoiseVolume smooths a volume series for range comparison: zeros are
// backward- then forward-filled, then a centred sliding median with an odd
// window of at most 9 rows is taken (edge windows are truncated, matching
// upstream's NaN padding + nanmedian). Port of upstream _denoise_volume.
func denoiseVolume(vol []float64) []float64 {
	n := len(vol)
	if n == 0 {
		return nil
	}

	filled := make([]float64, n)
	copy(filled, vol)
	for i := n - 2; i >= 0; i-- {
		if filled[i] == 0 {
			filled[i] = filled[i+1]
		}
	}
	for i := 1; i < n; i++ {
		if filled[i] == 0 {
			filled[i] = filled[i-1]
		}
	}

	w := n
	if w > 9 {
		w = 9
	}
	if w%2 == 0 {
		w--
	}
	pad := w / 2

	out := make([]float64, n)
	window := make([]float64, 0, w)
	for i := 0; i < n; i++ {
		lo := maxInt(0, i-pad)
		hi := minInt(n, i+pad+1)
		window = window[:0]
		for j := lo; j < hi; j++ {
			if filled[j] > 0 && !math.IsNaN(filled[j]) {
				window = append(window, filled[j])
			}
		}
		if len(window) == 0 {
			out[i] = 0
			continue
		}
		out[i] = stats.Median(window)
	}
	return out
}

func positiveValues(values []float64) []float64 {
	out := make([]float64, 0, len(values))
	for _, v := range values {
		if v > 0 {
			out = append(out, v)
		}
	}
	return out
}
