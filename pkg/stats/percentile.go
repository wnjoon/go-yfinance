package stats

import (
	"math"
	"sort"
)

// Percentile calculates the p-th percentile of the given data using linear interpolation.
// This matches numpy.percentile with default interpolation method.
//
// Parameters:
//   - data: slice of float64 values
//   - p: percentile to compute (0-100)
//
// Returns the percentile value. Returns NaN for empty data.
func Percentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return math.NaN()
	}

	// Make a sorted copy
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	n := float64(len(sorted))

	// Calculate the index using numpy's linear interpolation method
	// numpy uses: index = (n - 1) * p / 100
	index := (n - 1) * p / 100.0

	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation between lower and upper
	fraction := index - float64(lower)
	return sorted[lower]*(1-fraction) + sorted[upper]*fraction
}

// IQR calculates the interquartile range (Q3 - Q1).
// Returns Q1, Q3, and IQR.
//
// The interquartile range is used for outlier detection:
//   - Lower bound: Q1 - 1.5 * IQR
//   - Upper bound: Q3 + 1.5 * IQR
func IQR(data []float64) (q1, q3, iqr float64) {
	if len(data) == 0 {
		return math.NaN(), math.NaN(), math.NaN()
	}

	q1 = Percentile(data, 25.0)
	q3 = Percentile(data, 75.0)
	iqr = q3 - q1

	return q1, q3, iqr
}

// OutlierBounds calculates the lower and upper bounds for outlier detection
// using the IQR method with a configurable multiplier.
//
// Parameters:
//   - data: slice of float64 values
//   - multiplier: IQR multiplier (typically 1.5 for outliers, 3.0 for extreme outliers)
//
// Returns lower bound, upper bound.
func OutlierBounds(data []float64, multiplier float64) (lower, upper float64) {
	q1, q3, iqr := IQR(data)
	lower = q1 - multiplier*iqr
	upper = q3 + multiplier*iqr
	return lower, upper
}

// Mean calculates the arithmetic mean of the data.
// Returns NaN for empty data.
func Mean(data []float64) float64 {
	if len(data) == 0 {
		return math.NaN()
	}

	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

// Std calculates the standard deviation of the data.
// Uses n-1 denominator (sample standard deviation) by default.
//
// Parameters:
//   - data: slice of float64 values
//   - ddof: delta degrees of freedom (0 for population, 1 for sample)
func Std(data []float64, ddof int) float64 {
	if len(data) <= ddof {
		return math.NaN()
	}

	mean := Mean(data)
	sumSq := 0.0
	for _, v := range data {
		diff := v - mean
		sumSq += diff * diff
	}

	return math.Sqrt(sumSq / float64(len(data)-ddof))
}

// Median calculates the median (50th percentile) of the data.
func Median(data []float64) float64 {
	return Percentile(data, 50.0)
}

// RemoveNaN returns a new slice with NaN values removed.
func RemoveNaN(data []float64) []float64 {
	result := make([]float64, 0, len(data))
	for _, v := range data {
		if !math.IsNaN(v) {
			result = append(result, v)
		}
	}
	return result
}

// FilterByMask returns elements where mask is true.
func FilterByMask(data []float64, mask []bool) []float64 {
	result := make([]float64, 0)
	for i, v := range data {
		if i < len(mask) && mask[i] {
			result = append(result, v)
		}
	}
	return result
}

// Abs returns absolute values of the data.
func Abs(data []float64) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = math.Abs(v)
	}
	return result
}
