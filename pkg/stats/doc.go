// Package stats provides statistical utility functions for price repair operations.
//
// This package includes functions for percentile calculations, z-score computations,
// median filtering, and outlier detection. These utilities are essential for
// detecting and correcting data quality issues in financial time series.
//
// # Percentile Functions
//
// The package provides percentile calculation using linear interpolation:
//
//	p50 := stats.Percentile(data, 50.0)  // Median
//	q1, q3, iqr := stats.IQR(data)       // Interquartile range
//
// # Z-Score Functions
//
// Z-score calculations for standardization and outlier detection:
//
//	z := stats.ZScore(value, mean, std)
//	zScores := stats.ZScoreSlice(data)
//
// # Filtering Functions
//
// Median filter and outlier detection for noise reduction:
//
//	filtered := stats.MedianFilter(data, windowSize)
//	mask := stats.OutlierMask(data, multiplier)
//
// These functions are designed to match the behavior of numpy and scipy
// functions used in the Python yfinance implementation.
package stats
