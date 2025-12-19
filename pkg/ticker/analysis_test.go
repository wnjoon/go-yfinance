package ticker

import (
	"testing"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestRecommendationTotal(t *testing.T) {
	rec := models.Recommendation{
		Period:     "0m",
		StrongBuy:  5,
		Buy:        24,
		Hold:       15,
		Sell:       1,
		StrongSell: 3,
	}

	total := rec.Total()
	expected := 48

	if total != expected {
		t.Errorf("Expected total %d, got %d", expected, total)
	}
}

func TestAnalysisCacheInitialization(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	defer tkr.Close()

	if tkr.analysisCache != nil {
		t.Error("Expected analysisCache to be nil initially")
	}
}

func TestClearCacheIncludesAnalysis(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	defer tkr.Close()

	// Manually set analysisCache
	tkr.analysisCache = &analysisCache{
		recommendations: &models.RecommendationTrend{
			Trend: []models.Recommendation{{Period: "0m"}},
		},
	}

	// Clear cache
	tkr.ClearCache()

	if tkr.analysisCache != nil {
		t.Error("Expected analysisCache to be nil after ClearCache")
	}
}

func TestAnalysisModels(t *testing.T) {
	// Test PriceTarget
	pt := models.PriceTarget{
		Current:          250.00,
		High:             350.00,
		Low:              200.00,
		Mean:             287.50,
		Median:           290.00,
		NumberOfAnalysts: 40,
		RecommendationKey: "buy",
		RecommendationMean: 2.1,
	}

	if pt.Current != 250.00 {
		t.Errorf("Expected current 250.00, got %f", pt.Current)
	}

	if pt.NumberOfAnalysts != 40 {
		t.Errorf("Expected 40 analysts, got %d", pt.NumberOfAnalysts)
	}

	// Test EarningsEstimate
	est := models.EarningsEstimate{
		Period:           "0q",
		EndDate:          "2024-12-31",
		NumberOfAnalysts: 30,
		Avg:              2.50,
		Low:              2.30,
		High:             2.70,
		YearAgoEPS:       2.20,
		Growth:           0.136,
	}

	if est.Period != "0q" {
		t.Errorf("Expected period '0q', got '%s'", est.Period)
	}

	if est.Growth != 0.136 {
		t.Errorf("Expected growth 0.136, got %f", est.Growth)
	}
}

func TestGrowthEstimatePointers(t *testing.T) {
	// Test nil pointer handling for growth estimates
	ge := models.GrowthEstimate{
		Period: "0q",
	}

	if ge.StockGrowth != nil {
		t.Error("Expected StockGrowth to be nil")
	}

	// Test with value
	val := 0.15
	ge.StockGrowth = &val

	if ge.StockGrowth == nil || *ge.StockGrowth != 0.15 {
		t.Error("Expected StockGrowth to be 0.15")
	}
}

func TestEarningsHistoryItem(t *testing.T) {
	item := models.EarningsHistoryItem{
		Period:          "-1q",
		EPSActual:       2.50,
		EPSEstimate:     2.40,
		EPSDifference:   0.10,
		SurprisePercent: 0.0417,
	}

	if item.EPSActual != 2.50 {
		t.Errorf("Expected EPSActual 2.50, got %f", item.EPSActual)
	}

	if item.EPSDifference != 0.10 {
		t.Errorf("Expected EPSDifference 0.10, got %f", item.EPSDifference)
	}
}

// Integration test - commented out for CI, run manually
// func TestAnalysisLive(t *testing.T) {
// 	tkr, err := New("AAPL")
// 	if err != nil {
// 		t.Fatalf("Failed to create ticker: %v", err)
// 	}
// 	defer tkr.Close()
//
// 	// Test Recommendations
// 	rec, err := tkr.Recommendations()
// 	if err != nil {
// 		t.Fatalf("Failed to get recommendations: %v", err)
// 	}
//
// 	if rec == nil || len(rec.Trend) == 0 {
// 		t.Fatal("Expected non-empty recommendations")
// 	}
//
// 	t.Logf("Recommendations: %d periods", len(rec.Trend))
// 	for _, r := range rec.Trend {
// 		t.Logf("  %s: Buy=%d, Hold=%d, Sell=%d (Total=%d)",
// 			r.Period, r.StrongBuy+r.Buy, r.Hold, r.Sell+r.StrongSell, r.Total())
// 	}
//
// 	// Test Price Targets
// 	pt, err := tkr.PriceTarget()
// 	if err != nil {
// 		t.Fatalf("Failed to get price targets: %v", err)
// 	}
//
// 	t.Logf("Price Targets: Current=$%.2f, Mean=$%.2f, High=$%.2f, Low=$%.2f",
// 		pt.Current, pt.Mean, pt.High, pt.Low)
// 	t.Logf("Recommendation: %s (%.2f)", pt.RecommendationKey, pt.RecommendationMean)
//
// 	// Test Earnings Estimates
// 	ee, err := tkr.EarningsEstimates()
// 	if err != nil {
// 		t.Fatalf("Failed to get earnings estimates: %v", err)
// 	}
//
// 	t.Logf("Earnings Estimates: %d periods", len(ee))
// 	for _, e := range ee {
// 		t.Logf("  %s: Avg=%.2f, Growth=%.2f%%", e.Period, e.Avg, e.Growth*100)
// 	}
//
// 	// Test Earnings History
// 	eh, err := tkr.EarningsHistory()
// 	if err != nil {
// 		t.Fatalf("Failed to get earnings history: %v", err)
// 	}
//
// 	t.Logf("Earnings History: %d quarters", len(eh.History))
// 	for _, h := range eh.History {
// 		t.Logf("  %s (%s): Actual=%.2f, Est=%.2f, Surprise=%.2f%%",
// 			h.Period, h.Quarter.Format("2006-01-02"), h.EPSActual, h.EPSEstimate, h.SurprisePercent*100)
// 	}
// }
