package stats

import (
	"math"
	"testing"
)

const tolerance = 1e-10

func almostEqual(a, b float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	return math.Abs(a-b) < tolerance
}

// TestPercentile tests the Percentile function
func TestPercentile(t *testing.T) {
	tests := []struct {
		name     string
		data     []float64
		p        float64
		expected float64
	}{
		{
			name:     "median of odd length",
			data:     []float64{1, 2, 3, 4, 5},
			p:        50,
			expected: 3,
		},
		{
			name:     "median of even length",
			data:     []float64{1, 2, 3, 4},
			p:        50,
			expected: 2.5,
		},
		{
			name:     "Q1",
			data:     []float64{1, 2, 3, 4, 5, 6, 7, 8},
			p:        25,
			expected: 2.75,
		},
		{
			name:     "Q3",
			data:     []float64{1, 2, 3, 4, 5, 6, 7, 8},
			p:        75,
			expected: 6.25,
		},
		{
			name:     "0th percentile",
			data:     []float64{1, 2, 3, 4, 5},
			p:        0,
			expected: 1,
		},
		{
			name:     "100th percentile",
			data:     []float64{1, 2, 3, 4, 5},
			p:        100,
			expected: 5,
		},
		{
			name:     "empty data",
			data:     []float64{},
			p:        50,
			expected: math.NaN(),
		},
		{
			name:     "single element",
			data:     []float64{42},
			p:        50,
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Percentile(tt.data, tt.p)
			if !almostEqual(result, tt.expected) {
				t.Errorf("Percentile(%v, %v) = %v, want %v", tt.data, tt.p, result, tt.expected)
			}
		})
	}
}

// TestIQR tests the IQR function
func TestIQR(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	q1, q3, iqr := IQR(data)

	expectedQ1 := 3.25
	expectedQ3 := 7.75
	expectedIQR := 4.5

	if !almostEqual(q1, expectedQ1) {
		t.Errorf("Q1 = %v, want %v", q1, expectedQ1)
	}
	if !almostEqual(q3, expectedQ3) {
		t.Errorf("Q3 = %v, want %v", q3, expectedQ3)
	}
	if !almostEqual(iqr, expectedIQR) {
		t.Errorf("IQR = %v, want %v", iqr, expectedIQR)
	}
}

// TestMean tests the Mean function
func TestMean(t *testing.T) {
	tests := []struct {
		name     string
		data     []float64
		expected float64
	}{
		{
			name:     "simple mean",
			data:     []float64{1, 2, 3, 4, 5},
			expected: 3,
		},
		{
			name:     "single element",
			data:     []float64{10},
			expected: 10,
		},
		{
			name:     "empty data",
			data:     []float64{},
			expected: math.NaN(),
		},
		{
			name:     "negative values",
			data:     []float64{-1, -2, -3},
			expected: -2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Mean(tt.data)
			if !almostEqual(result, tt.expected) {
				t.Errorf("Mean(%v) = %v, want %v", tt.data, result, tt.expected)
			}
		})
	}
}

// TestStd tests the Std function
func TestStd(t *testing.T) {
	// Test sample standard deviation (ddof=1)
	data := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	result := Std(data, 1)
	expected := 2.138089935299395 // calculated separately

	if math.Abs(result-expected) > 0.0001 {
		t.Errorf("Std(%v, 1) = %v, want %v", data, result, expected)
	}

	// Test population standard deviation (ddof=0)
	result = Std(data, 0)
	expected = 2.0 // calculated separately

	if math.Abs(result-expected) > 0.0001 {
		t.Errorf("Std(%v, 0) = %v, want %v", data, result, expected)
	}
}

// TestZScore tests the ZScore function
func TestZScore(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		mean     float64
		std      float64
		expected float64
	}{
		{
			name:     "positive z-score",
			value:    15,
			mean:     10,
			std:      2.5,
			expected: 2.0,
		},
		{
			name:     "negative z-score",
			value:    5,
			mean:     10,
			std:      2.5,
			expected: -2.0,
		},
		{
			name:     "zero z-score",
			value:    10,
			mean:     10,
			std:      2.5,
			expected: 0.0,
		},
		{
			name:     "zero std",
			value:    10,
			mean:     10,
			std:      0,
			expected: math.NaN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ZScore(tt.value, tt.mean, tt.std)
			if !almostEqual(result, tt.expected) {
				t.Errorf("ZScore(%v, %v, %v) = %v, want %v", tt.value, tt.mean, tt.std, result, tt.expected)
			}
		})
	}
}

// TestMedianFilter tests the MedianFilter function
func TestMedianFilter(t *testing.T) {
	data := []float64{1, 2, 100, 4, 5} // 100 is an outlier
	result := MedianFilter(data, 3)

	// With window size 3:
	// Position 0: window [1, 2] -> median = 1.5
	// Position 1: window [1, 2, 100] -> median = 2
	// Position 2: window [2, 100, 4] -> sorted [2, 4, 100] -> median = 4
	// Position 3: window [100, 4, 5] -> sorted [4, 5, 100] -> median = 5
	// Position 4: window [4, 5] -> median = 4.5
	expected := []float64{1.5, 2, 4, 5, 4.5}

	for i, v := range result {
		if !almostEqual(v, expected[i]) {
			t.Errorf("MedianFilter result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestOutlierMask tests the OutlierMask function
func TestOutlierMask(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 100} // 100 is an outlier
	mask := OutlierMask(data, 1.5)

	// Only 100 should be detected as outlier
	if !mask[5] {
		t.Error("100 should be detected as outlier")
	}

	for i := 0; i < 5; i++ {
		if mask[i] {
			t.Errorf("data[%d] = %v should not be an outlier", i, data[i])
		}
	}
}

// TestDiff tests the Diff function
func TestDiff(t *testing.T) {
	data := []float64{1, 3, 6, 10}
	result := Diff(data)
	expected := []float64{2, 3, 4}

	if len(result) != len(expected) {
		t.Errorf("Diff length = %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if !almostEqual(v, expected[i]) {
			t.Errorf("Diff result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestPctChange tests the PctChange function
func TestPctChange(t *testing.T) {
	data := []float64{100, 110, 99, 100}
	result := PctChange(data)
	expected := []float64{0.1, -0.1, 0.01010101010101}

	if len(result) != len(expected) {
		t.Errorf("PctChange length = %d, want %d", len(result), len(expected))
	}

	for i, v := range result {
		if math.Abs(v-expected[i]) > 0.0001 {
			t.Errorf("PctChange result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestFindBlocks tests the FindBlocks function
func TestFindBlocks(t *testing.T) {
	mask := []bool{false, true, true, false, true, true, true, false}
	blocks := FindBlocks(mask)

	expected := [][2]int{{1, 3}, {4, 7}}

	if len(blocks) != len(expected) {
		t.Errorf("FindBlocks found %d blocks, want %d", len(blocks), len(expected))
	}

	for i, block := range blocks {
		if block != expected[i] {
			t.Errorf("Block %d = %v, want %v", i, block, expected[i])
		}
	}
}

// TestRemoveNaN tests the RemoveNaN function
func TestRemoveNaN(t *testing.T) {
	data := []float64{1, math.NaN(), 2, math.NaN(), 3}
	result := RemoveNaN(data)

	if len(result) != 3 {
		t.Errorf("RemoveNaN length = %d, want 3", len(result))
	}

	expected := []float64{1, 2, 3}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("RemoveNaN result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestAbs tests the Abs function
func TestAbs(t *testing.T) {
	data := []float64{-1, 2, -3, 4, -5}
	result := Abs(data)
	expected := []float64{1, 2, 3, 4, 5}

	for i, v := range result {
		if v != expected[i] {
			t.Errorf("Abs result[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestOHLCMedian tests the OHLCMedian function
func TestOHLCMedian(t *testing.T) {
	// Open=10, High=15, Low=8, Close=12
	// Sorted: 8, 10, 12, 15 -> Median = (10+12)/2 = 11
	result := OHLCMedian(10, 15, 8, 12)
	expected := 11.0

	if !almostEqual(result, expected) {
		t.Errorf("OHLCMedian(10, 15, 8, 12) = %v, want %v", result, expected)
	}
}

// TestCountTrue tests the CountTrue function
func TestCountTrue(t *testing.T) {
	mask := []bool{true, false, true, true, false}
	result := CountTrue(mask)

	if result != 3 {
		t.Errorf("CountTrue = %d, want 3", result)
	}
}

// TestAllAny tests the All and Any functions
func TestAllAny(t *testing.T) {
	allTrue := []bool{true, true, true}
	allFalse := []bool{false, false, false}
	mixed := []bool{true, false, true}

	if !All(allTrue) {
		t.Error("All(allTrue) should be true")
	}
	if All(mixed) {
		t.Error("All(mixed) should be false")
	}

	if !Any(mixed) {
		t.Error("Any(mixed) should be true")
	}
	if Any(allFalse) {
		t.Error("Any(allFalse) should be false")
	}
}

// TestWeightedMean tests the WeightedMean function
func TestWeightedMean(t *testing.T) {
	data := []float64{10, 20, 30}
	weights := []float64{1, 2, 3}
	result := WeightedMean(data, weights)
	// (10*1 + 20*2 + 30*3) / (1+2+3) = 140/6 = 23.333...
	expected := 140.0 / 6.0

	if math.Abs(result-expected) > 0.0001 {
		t.Errorf("WeightedMean = %v, want %v", result, expected)
	}
}

// TestMedianFilter2D tests the MedianFilter2D function
func TestMedianFilter2D(t *testing.T) {
	data := [][]float64{
		{1, 2, 3},
		{4, 100, 6}, // 100 is an outlier
		{7, 8, 9},
	}

	result := MedianFilter2D(data, 3)

	// The center value (100) should be filtered to approximately 5-6
	// as it's the median of surrounding values [1,2,3,4,6,7,8,9] = 5
	if result[1][1] > 10 {
		t.Errorf("MedianFilter2D center = %v, should be filtered (was 100)", result[1][1])
	}
}

// Benchmark tests
func BenchmarkPercentile(b *testing.B) {
	data := make([]float64, 1000)
	for i := range data {
		data[i] = float64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Percentile(data, 50)
	}
}

func BenchmarkMedianFilter(b *testing.B) {
	data := make([]float64, 1000)
	for i := range data {
		data[i] = float64(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MedianFilter(data, 5)
	}
}
