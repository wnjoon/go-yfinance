package stats

import "math"

// ZScore calculates the z-score (standard score) for a single value.
//
// Z-score = (value - mean) / std
//
// Returns NaN if std is zero or NaN.
func ZScore(value, mean, std float64) float64 {
	if std == 0 || math.IsNaN(std) {
		return math.NaN()
	}
	return (value - mean) / std
}

// ZScoreSlice calculates z-scores for all values in the data.
// Uses sample standard deviation (ddof=1).
func ZScoreSlice(data []float64) []float64 {
	if len(data) == 0 {
		return nil
	}

	mean := Mean(data)
	std := Std(data, 1)

	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = ZScore(v, mean, std)
	}
	return result
}

// ZScoreWithParams calculates z-scores using provided mean and std.
func ZScoreWithParams(data []float64, mean, std float64) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		result[i] = ZScore(v, mean, std)
	}
	return result
}

// DetectOutliersByZScore identifies outliers based on z-score threshold.
// Returns a boolean mask where true indicates an outlier.
//
// Parameters:
//   - data: slice of float64 values
//   - threshold: z-score threshold (typically 2.0 or 3.0)
func DetectOutliersByZScore(data []float64, threshold float64) []bool {
	zScores := ZScoreSlice(data)
	mask := make([]bool, len(data))

	for i, z := range zScores {
		mask[i] = math.Abs(z) > threshold
	}
	return mask
}

// WeightedMean calculates the weighted arithmetic mean.
// Returns NaN if weights sum to zero or if slices have different lengths.
func WeightedMean(data, weights []float64) float64 {
	if len(data) != len(weights) || len(data) == 0 {
		return math.NaN()
	}

	sumWeights := 0.0
	sumWeighted := 0.0

	for i, v := range data {
		sumWeights += weights[i]
		sumWeighted += v * weights[i]
	}

	if sumWeights == 0 {
		return math.NaN()
	}

	return sumWeighted / sumWeights
}

// RollingMean calculates a rolling (moving) mean with the specified window size.
// Uses center alignment. Returns NaN for positions where window is incomplete.
func RollingMean(data []float64, windowSize int) []float64 {
	n := len(data)
	result := make([]float64, n)

	halfWindow := windowSize / 2

	for i := range data {
		start := i - halfWindow
		end := i + halfWindow + 1

		if start < 0 || end > n {
			result[i] = math.NaN()
			continue
		}

		sum := 0.0
		count := 0
		for j := start; j < end; j++ {
			if !math.IsNaN(data[j]) {
				sum += data[j]
				count++
			}
		}

		if count > 0 {
			result[i] = sum / float64(count)
		} else {
			result[i] = math.NaN()
		}
	}

	return result
}

// RollingStd calculates a rolling (moving) standard deviation.
// Uses center alignment and sample std (ddof=1).
func RollingStd(data []float64, windowSize int) []float64 {
	n := len(data)
	result := make([]float64, n)

	halfWindow := windowSize / 2

	for i := range data {
		start := i - halfWindow
		end := i + halfWindow + 1

		if start < 0 || end > n {
			result[i] = math.NaN()
			continue
		}

		window := make([]float64, 0, end-start)
		for j := start; j < end; j++ {
			if !math.IsNaN(data[j]) {
				window = append(window, data[j])
			}
		}

		if len(window) > 1 {
			result[i] = Std(window, 1)
		} else {
			result[i] = math.NaN()
		}
	}

	return result
}
