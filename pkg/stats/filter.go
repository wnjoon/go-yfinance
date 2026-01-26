package stats

import (
	"math"
	"sort"
)

// MedianFilter applies a 1D median filter to the data.
// This is similar to scipy.ndimage.median_filter for 1D arrays.
//
// Parameters:
//   - data: input slice
//   - windowSize: filter window size (should be odd)
//
// Returns filtered data with same length as input.
// Edge values use smaller windows.
func MedianFilter(data []float64, windowSize int) []float64 {
	n := len(data)
	if n == 0 {
		return nil
	}

	result := make([]float64, n)
	halfWindow := windowSize / 2

	for i := range data {
		// Determine window bounds
		start := i - halfWindow
		end := i + halfWindow + 1

		if start < 0 {
			start = 0
		}
		if end > n {
			end = n
		}

		// Collect window values
		window := make([]float64, 0, end-start)
		for j := start; j < end; j++ {
			if !math.IsNaN(data[j]) {
				window = append(window, data[j])
			}
		}

		if len(window) > 0 {
			result[i] = Median(window)
		} else {
			result[i] = math.NaN()
		}
	}

	return result
}

// MedianFilter2D applies a 2D median filter to the data matrix.
// This is similar to scipy.ndimage.median_filter for 2D arrays.
//
// Parameters:
//   - data: 2D slice [rows][cols]
//   - windowSize: filter window size for both dimensions
//
// Returns filtered 2D data.
func MedianFilter2D(data [][]float64, windowSize int) [][]float64 {
	if len(data) == 0 || len(data[0]) == 0 {
		return nil
	}

	rows := len(data)
	cols := len(data[0])
	result := make([][]float64, rows)

	halfWindow := windowSize / 2

	for i := range data {
		result[i] = make([]float64, cols)

		for j := range data[i] {
			// Determine window bounds
			rowStart := i - halfWindow
			rowEnd := i + halfWindow + 1
			colStart := j - halfWindow
			colEnd := j + halfWindow + 1

			if rowStart < 0 {
				rowStart = 0
			}
			if rowEnd > rows {
				rowEnd = rows
			}
			if colStart < 0 {
				colStart = 0
			}
			if colEnd > cols {
				colEnd = cols
			}

			// Collect window values
			window := make([]float64, 0)
			for r := rowStart; r < rowEnd; r++ {
				for c := colStart; c < colEnd; c++ {
					if !math.IsNaN(data[r][c]) {
						window = append(window, data[r][c])
					}
				}
			}

			if len(window) > 0 {
				result[i][j] = Median(window)
			} else {
				result[i][j] = math.NaN()
			}
		}
	}

	return result
}

// OutlierMask creates a boolean mask for outliers using the IQR method.
// Returns true for values that are outliers.
//
// Parameters:
//   - data: slice of float64 values
//   - multiplier: IQR multiplier (typically 1.5)
func OutlierMask(data []float64, multiplier float64) []bool {
	lower, upper := OutlierBounds(data, multiplier)
	mask := make([]bool, len(data))

	for i, v := range data {
		mask[i] = v < lower || v > upper
	}
	return mask
}

// InlierMask creates a boolean mask for inliers (non-outliers).
// Returns true for values that are NOT outliers.
func InlierMask(data []float64, multiplier float64) []bool {
	mask := OutlierMask(data, multiplier)
	for i := range mask {
		mask[i] = !mask[i]
	}
	return mask
}

// ClipOutliers replaces outliers with boundary values.
func ClipOutliers(data []float64, multiplier float64) []float64 {
	lower, upper := OutlierBounds(data, multiplier)
	result := make([]float64, len(data))

	for i, v := range data {
		if v < lower {
			result[i] = lower
		} else if v > upper {
			result[i] = upper
		} else {
			result[i] = v
		}
	}
	return result
}

// Diff calculates the difference between consecutive elements.
// Returns slice of length n-1.
func Diff(data []float64) []float64 {
	if len(data) < 2 {
		return nil
	}

	result := make([]float64, len(data)-1)
	for i := 1; i < len(data); i++ {
		result[i-1] = data[i] - data[i-1]
	}
	return result
}

// PctChange calculates the percentage change between consecutive elements.
// Returns slice of length n-1.
func PctChange(data []float64) []float64 {
	if len(data) < 2 {
		return nil
	}

	result := make([]float64, len(data)-1)
	for i := 1; i < len(data); i++ {
		if data[i-1] != 0 {
			result[i-1] = (data[i] - data[i-1]) / data[i-1]
		} else {
			result[i-1] = math.NaN()
		}
	}
	return result
}

// FindBlocks identifies contiguous blocks of True values in a boolean mask.
// Returns slice of [start, end) pairs.
func FindBlocks(mask []bool) [][2]int {
	var blocks [][2]int
	inBlock := false
	start := 0

	for i, v := range mask {
		if v && !inBlock {
			start = i
			inBlock = true
		} else if !v && inBlock {
			blocks = append(blocks, [2]int{start, i})
			inBlock = false
		}
	}

	// Handle block at end
	if inBlock {
		blocks = append(blocks, [2]int{start, len(mask)})
	}

	return blocks
}

// MedianOfSlice calculates the median without sorting the original slice.
func MedianOfSlice(data []float64) float64 {
	if len(data) == 0 {
		return math.NaN()
	}

	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// OHLC calculates the median of Open, High, Low, Close values.
// This provides a robust estimate of the "typical" price.
func OHLCMedian(open, high, low, close float64) float64 {
	values := []float64{open, high, low, close}
	return MedianOfSlice(values)
}

// CountTrue counts the number of true values in a boolean slice.
func CountTrue(mask []bool) int {
	count := 0
	for _, v := range mask {
		if v {
			count++
		}
	}
	return count
}

// All returns true if all values in the mask are true.
func All(mask []bool) bool {
	for _, v := range mask {
		if !v {
			return false
		}
	}
	return true
}

// Any returns true if any value in the mask is true.
func Any(mask []bool) bool {
	for _, v := range mask {
		if v {
			return true
		}
	}
	return false
}
