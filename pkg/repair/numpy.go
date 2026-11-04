package repair

import (
	"math"
	"sort"
)

// The price-shift detector compares statistics against thresholds, so a
// one-ulp difference can flip a signal. These helpers reproduce NumPy's
// float64 results bit for bit (verified against NumPy 2.x on random inputs):
// np.sum's pairwise summation, np.mean, np.std, np.percentile's "linear"
// method, np.median and np.round.

// npPairwiseBlock is NumPy's PW_BLOCKSIZE.
const npPairwiseBlock = 128

// npSum is np.sum of a float64 array: pairwise summation with eight
// accumulators per block of at most 128 values.
func npSum(a []float64) float64 {
	n := len(a)
	switch {
	case n < 8:
		res := 0.0
		for _, v := range a {
			res += v
		}
		return res
	case n <= npPairwiseBlock:
		var r [8]float64
		copy(r[:], a[:8])
		i := 8
		for ; i < n-n%8; i += 8 {
			for j := range r {
				r[j] += a[i+j]
			}
		}
		res := ((r[0] + r[1]) + (r[2] + r[3])) + ((r[4] + r[5]) + (r[6] + r[7]))
		for ; i < n; i++ {
			res += a[i]
		}
		return res
	default:
		n2 := n / 2
		n2 -= n2 % 8
		return npSum(a[:n2]) + npSum(a[n2:])
	}
}

// npMean is np.mean: NaN for an empty array.
func npMean(a []float64) float64 {
	if len(a) == 0 {
		return math.NaN()
	}
	return npSum(a) / float64(len(a))
}

// npStd is np.std(a, ddof=ddof): NaN when len(a) <= ddof.
func npStd(a []float64, ddof int) float64 {
	if len(a) <= ddof {
		return math.NaN()
	}
	mean := npMean(a)
	sq := make([]float64, len(a))
	for i, v := range a {
		d := v - mean
		sq[i] = d * d
	}
	return math.Sqrt(npSum(sq) / float64(len(a)-ddof))
}

// npPercentile is np.percentile(a, q) with the default "linear" method,
// including NumPy's two-sided lerp. NaN for an empty array.
func npPercentile(a []float64, q float64) float64 {
	n := len(a)
	if n == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), a...)
	sort.Float64s(s)
	idx := float64(n-1) * (q / 100)
	lo := int(math.Floor(idx))
	hi := minInt(lo+1, n-1)
	g := idx - float64(lo)
	diff := s[hi] - s[lo]
	if g >= 0.5 {
		return s[hi] - diff*(1-g)
	}
	return s[lo] + diff*g
}

// npMedian is np.median: NaN if any value is NaN or a is empty.
func npMedian(a []float64) float64 {
	n := len(a)
	if n == 0 {
		return math.NaN()
	}
	for _, v := range a {
		if math.IsNaN(v) {
			return math.NaN()
		}
	}
	s := append([]float64(nil), a...)
	sort.Float64s(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

// npRound3 is np.round(x, 3): scale, round half to even, unscale.
func npRound3(x float64) float64 {
	return math.RoundToEven(x*1000) / 1000
}

// pyMax is Python's builtin max(a, b): a unless b > a, so a NaN first
// argument wins and a NaN second argument is ignored.
func pyMax(a, b float64) float64 {
	if b > a {
		return b
	}
	return a
}
