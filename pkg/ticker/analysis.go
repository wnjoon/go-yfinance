package ticker

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// analysisCache stores cached analysis data.
type analysisCache struct {
	recommendations   *models.RecommendationTrend
	priceTarget       *models.PriceTarget
	earningsEstimates []models.EarningsEstimate
	revenueEstimates  []models.RevenueEstimate
	epsTrends         []models.EPSTrend
	epsRevisions      []models.EPSRevision
	earningsHistory   *models.EarningsHistory
	growthEstimates   []models.GrowthEstimate
	// Raw cached data from API
	earningsTrendRaw map[string]interface{}
}

// Recommendations returns analyst recommendation trends.
func (t *Ticker) Recommendations() (*models.RecommendationTrend, error) {
	if t.analysisCache != nil && t.analysisCache.recommendations != nil {
		return t.analysisCache.recommendations, nil
	}

	data, err := t.fetchQuoteSummary([]string{"recommendationTrend"})
	if err != nil {
		return nil, err
	}

	result, err := t.parseRecommendations(data)
	if err != nil {
		return nil, err
	}

	t.initAnalysisCache()
	t.analysisCache.recommendations = result
	return result, nil
}

// PriceTarget returns analyst price targets.
func (t *Ticker) PriceTarget() (*models.PriceTarget, error) {
	if t.analysisCache != nil && t.analysisCache.priceTarget != nil {
		return t.analysisCache.priceTarget, nil
	}

	data, err := t.fetchQuoteSummary([]string{"financialData"})
	if err != nil {
		return nil, err
	}

	result, err := t.parsePriceTarget(data)
	if err != nil {
		return nil, err
	}

	t.initAnalysisCache()
	t.analysisCache.priceTarget = result
	return result, nil
}

// EarningsEstimates returns earnings estimates for upcoming periods.
func (t *Ticker) EarningsEstimates() ([]models.EarningsEstimate, error) {
	if t.analysisCache != nil && t.analysisCache.earningsEstimates != nil {
		return t.analysisCache.earningsEstimates, nil
	}

	if err := t.ensureEarningsTrend(); err != nil {
		return nil, err
	}

	result := t.parseEarningsEstimates()

	t.initAnalysisCache()
	t.analysisCache.earningsEstimates = result
	return result, nil
}

// RevenueEstimates returns revenue estimates for upcoming periods.
func (t *Ticker) RevenueEstimates() ([]models.RevenueEstimate, error) {
	if t.analysisCache != nil && t.analysisCache.revenueEstimates != nil {
		return t.analysisCache.revenueEstimates, nil
	}

	if err := t.ensureEarningsTrend(); err != nil {
		return nil, err
	}

	result := t.parseRevenueEstimates()

	t.initAnalysisCache()
	t.analysisCache.revenueEstimates = result
	return result, nil
}

// EPSTrend returns EPS trend data.
func (t *Ticker) EPSTrend() ([]models.EPSTrend, error) {
	if t.analysisCache != nil && t.analysisCache.epsTrends != nil {
		return t.analysisCache.epsTrends, nil
	}

	if err := t.ensureEarningsTrend(); err != nil {
		return nil, err
	}

	result := t.parseEPSTrends()

	t.initAnalysisCache()
	t.analysisCache.epsTrends = result
	return result, nil
}

// EPSRevisions returns EPS revision data.
func (t *Ticker) EPSRevisions() ([]models.EPSRevision, error) {
	if t.analysisCache != nil && t.analysisCache.epsRevisions != nil {
		return t.analysisCache.epsRevisions, nil
	}

	if err := t.ensureEarningsTrend(); err != nil {
		return nil, err
	}

	result := t.parseEPSRevisions()

	t.initAnalysisCache()
	t.analysisCache.epsRevisions = result
	return result, nil
}

// EarningsHistory returns historical earnings data.
func (t *Ticker) EarningsHistory() (*models.EarningsHistory, error) {
	if t.analysisCache != nil && t.analysisCache.earningsHistory != nil {
		return t.analysisCache.earningsHistory, nil
	}

	data, err := t.fetchQuoteSummary([]string{"earningsHistory"})
	if err != nil {
		return nil, err
	}

	result, err := t.parseEarningsHistory(data)
	if err != nil {
		return nil, err
	}

	t.initAnalysisCache()
	t.analysisCache.earningsHistory = result
	return result, nil
}

// GrowthEstimates returns growth estimates comparing stock to industry/sector/index.
func (t *Ticker) GrowthEstimates() ([]models.GrowthEstimate, error) {
	if t.analysisCache != nil && t.analysisCache.growthEstimates != nil {
		return t.analysisCache.growthEstimates, nil
	}

	if err := t.ensureEarningsTrend(); err != nil {
		return nil, err
	}

	// Fetch additional trend data
	trendData, err := t.fetchQuoteSummary([]string{"industryTrend", "sectorTrend", "indexTrend"})
	if err != nil {
		return nil, err
	}

	result := t.parseGrowthEstimates(trendData)

	t.initAnalysisCache()
	t.analysisCache.growthEstimates = result
	return result, nil
}

// initAnalysisCache initializes the analysis cache if nil.
func (t *Ticker) initAnalysisCache() {
	if t.analysisCache == nil {
		t.analysisCache = &analysisCache{}
	}
}

// ensureEarningsTrend fetches earningsTrend data if not cached.
func (t *Ticker) ensureEarningsTrend() error {
	t.initAnalysisCache()
	if t.analysisCache.earningsTrendRaw != nil {
		return nil
	}

	data, err := t.fetchQuoteSummary([]string{"earningsTrend"})
	if err != nil {
		return err
	}

	t.analysisCache.earningsTrendRaw = data
	return nil
}

// fetchQuoteSummary fetches data from quoteSummary API.
func (t *Ticker) fetchQuoteSummary(modules []string) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/%s", endpoints.QuoteSummaryURL, t.symbol)

	params := url.Values{}
	params.Set("modules", strings.Join(modules, ","))
	params.Set("corsDomain", "finance.yahoo.com")
	params.Set("formatted", "false")

	params, err := t.auth.AddCrumbToParams(params)
	if err != nil {
		return nil, fmt.Errorf("failed to add crumb: %w", err)
	}

	resp, err := t.client.Get(apiURL, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quoteSummary: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	var rawResp map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Body), &rawResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	quoteSummary, ok := rawResp["quoteSummary"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response: missing quoteSummary")
	}

	results, ok := quoteSummary["result"].([]interface{})
	if !ok || len(results) == 0 {
		return nil, fmt.Errorf("invalid response: missing result")
	}

	result, ok := results[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response: result is not an object")
	}

	return result, nil
}

// parseRecommendations parses recommendationTrend data.
func (t *Ticker) parseRecommendations(data map[string]interface{}) (*models.RecommendationTrend, error) {
	recTrend, ok := data["recommendationTrend"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("recommendationTrend not found")
	}

	trend, ok := recTrend["trend"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("trend data not found")
	}

	result := &models.RecommendationTrend{
		Trend: make([]models.Recommendation, 0, len(trend)),
	}

	for _, item := range trend {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		rec := models.Recommendation{
			Period:     getString(itemMap, "period"),
			StrongBuy:  getInt(itemMap, "strongBuy"),
			Buy:        getInt(itemMap, "buy"),
			Hold:       getInt(itemMap, "hold"),
			Sell:       getInt(itemMap, "sell"),
			StrongSell: getInt(itemMap, "strongSell"),
		}
		result.Trend = append(result.Trend, rec)
	}

	return result, nil
}

// parsePriceTarget parses financialData for price targets.
func (t *Ticker) parsePriceTarget(data map[string]interface{}) (*models.PriceTarget, error) {
	finData, ok := data["financialData"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("financialData not found")
	}

	return &models.PriceTarget{
		Current:            getFloat64(finData, "currentPrice"),
		High:               getFloat64(finData, "targetHighPrice"),
		Low:                getFloat64(finData, "targetLowPrice"),
		Mean:               getFloat64(finData, "targetMeanPrice"),
		Median:             getFloat64(finData, "targetMedianPrice"),
		NumberOfAnalysts:   getInt(finData, "numberOfAnalystOpinions"),
		RecommendationKey:  getString(finData, "recommendationKey"),
		RecommendationMean: getFloat64(finData, "recommendationMean"),
	}, nil
}

// parseEarningsEstimates parses earnings estimates from earningsTrend.
func (t *Ticker) parseEarningsEstimates() []models.EarningsEstimate {
	if t.analysisCache == nil || t.analysisCache.earningsTrendRaw == nil {
		return nil
	}

	earningsTrend, ok := t.analysisCache.earningsTrendRaw["earningsTrend"].(map[string]interface{})
	if !ok {
		return nil
	}

	trend, ok := earningsTrend["trend"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]models.EarningsEstimate, 0)

	for _, item := range trend {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		earningsEst, ok := itemMap["earningsEstimate"].(map[string]interface{})
		if !ok {
			continue
		}

		est := models.EarningsEstimate{
			Period:           getString(itemMap, "period"),
			EndDate:          getString(itemMap, "endDate"),
			NumberOfAnalysts: getNestedInt(earningsEst, "numberOfAnalysts"),
			Avg:              getNestedFloat(earningsEst, "avg"),
			Low:              getNestedFloat(earningsEst, "low"),
			High:             getNestedFloat(earningsEst, "high"),
			YearAgoEPS:       getNestedFloat(earningsEst, "yearAgoEps"),
			Growth:           getNestedFloat(earningsEst, "growth"),
		}
		result = append(result, est)
	}

	return result
}

// parseRevenueEstimates parses revenue estimates from earningsTrend.
func (t *Ticker) parseRevenueEstimates() []models.RevenueEstimate {
	if t.analysisCache == nil || t.analysisCache.earningsTrendRaw == nil {
		return nil
	}

	earningsTrend, ok := t.analysisCache.earningsTrendRaw["earningsTrend"].(map[string]interface{})
	if !ok {
		return nil
	}

	trend, ok := earningsTrend["trend"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]models.RevenueEstimate, 0)

	for _, item := range trend {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		revenueEst, ok := itemMap["revenueEstimate"].(map[string]interface{})
		if !ok {
			continue
		}

		est := models.RevenueEstimate{
			Period:           getString(itemMap, "period"),
			EndDate:          getString(itemMap, "endDate"),
			NumberOfAnalysts: getNestedInt(revenueEst, "numberOfAnalysts"),
			Avg:              getNestedFloat(revenueEst, "avg"),
			Low:              getNestedFloat(revenueEst, "low"),
			High:             getNestedFloat(revenueEst, "high"),
			YearAgoRevenue:   getNestedFloat(revenueEst, "yearAgoRevenue"),
			Growth:           getNestedFloat(revenueEst, "growth"),
		}
		result = append(result, est)
	}

	return result
}

// parseEPSTrends parses EPS trends from earningsTrend.
func (t *Ticker) parseEPSTrends() []models.EPSTrend {
	if t.analysisCache == nil || t.analysisCache.earningsTrendRaw == nil {
		return nil
	}

	earningsTrend, ok := t.analysisCache.earningsTrendRaw["earningsTrend"].(map[string]interface{})
	if !ok {
		return nil
	}

	trend, ok := earningsTrend["trend"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]models.EPSTrend, 0)

	for _, item := range trend {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		epsTrend, ok := itemMap["epsTrend"].(map[string]interface{})
		if !ok {
			continue
		}

		est := models.EPSTrend{
			Period:     getString(itemMap, "period"),
			Current:    getNestedFloat(epsTrend, "current"),
			SevenDays:  getNestedFloat(epsTrend, "7daysAgo"),
			ThirtyDays: getNestedFloat(epsTrend, "30daysAgo"),
			SixtyDays:  getNestedFloat(epsTrend, "60daysAgo"),
			NinetyDays: getNestedFloat(epsTrend, "90daysAgo"),
		}
		result = append(result, est)
	}

	return result
}

// parseEPSRevisions parses EPS revisions from earningsTrend.
func (t *Ticker) parseEPSRevisions() []models.EPSRevision {
	if t.analysisCache == nil || t.analysisCache.earningsTrendRaw == nil {
		return nil
	}

	earningsTrend, ok := t.analysisCache.earningsTrendRaw["earningsTrend"].(map[string]interface{})
	if !ok {
		return nil
	}

	trend, ok := earningsTrend["trend"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]models.EPSRevision, 0)

	for _, item := range trend {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		epsRevisions, ok := itemMap["epsRevisions"].(map[string]interface{})
		if !ok {
			continue
		}

		rev := models.EPSRevision{
			Period:         getString(itemMap, "period"),
			UpLast7Days:    getNestedInt(epsRevisions, "upLast7days"),
			UpLast30Days:   getNestedInt(epsRevisions, "upLast30days"),
			DownLast7Days:  getNestedInt(epsRevisions, "downLast7Days"),
			DownLast30Days: getNestedInt(epsRevisions, "downLast30days"),
		}
		result = append(result, rev)
	}

	return result
}

// parseEarningsHistory parses earningsHistory data.
func (t *Ticker) parseEarningsHistory(data map[string]interface{}) (*models.EarningsHistory, error) {
	ehData, ok := data["earningsHistory"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("earningsHistory not found")
	}

	history, ok := ehData["history"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("history data not found")
	}

	result := &models.EarningsHistory{
		History: make([]models.EarningsHistoryItem, 0, len(history)),
	}

	for _, item := range history {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Parse quarter date
		var quarterTime time.Time
		if quarterData, ok := itemMap["quarter"].(map[string]interface{}); ok {
			if raw, ok := quarterData["raw"].(float64); ok {
				quarterTime = time.Unix(int64(raw), 0)
			}
		}

		histItem := models.EarningsHistoryItem{
			Period:          getString(itemMap, "period"),
			Quarter:         quarterTime,
			EPSActual:       getNestedFloat(itemMap, "epsActual"),
			EPSEstimate:     getNestedFloat(itemMap, "epsEstimate"),
			EPSDifference:   getNestedFloat(itemMap, "epsDifference"),
			SurprisePercent: getNestedFloat(itemMap, "surprisePercent"),
		}
		result.History = append(result.History, histItem)
	}

	return result, nil
}

// parseGrowthEstimates parses growth estimates from multiple sources.
func (t *Ticker) parseGrowthEstimates(trendData map[string]interface{}) []models.GrowthEstimate {
	if t.analysisCache == nil || t.analysisCache.earningsTrendRaw == nil {
		return nil
	}

	earningsTrend, ok := t.analysisCache.earningsTrendRaw["earningsTrend"].(map[string]interface{})
	if !ok {
		return nil
	}

	trend, ok := earningsTrend["trend"].([]interface{})
	if !ok {
		return nil
	}

	// Build map of period -> growth estimate
	estimateMap := make(map[string]*models.GrowthEstimate)

	// Parse stock growth from earningsTrend
	for _, item := range trend {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		period := getString(itemMap, "period")
		if period == "" {
			continue
		}

		growth := getNestedFloatPtr(itemMap, "growth")
		estimateMap[period] = &models.GrowthEstimate{
			Period:      period,
			StockGrowth: growth,
		}
	}

	// Parse industry/sector/index trends
	trendTypes := []struct {
		key   string
		field string
	}{
		{"industryTrend", "IndustryGrowth"},
		{"sectorTrend", "SectorGrowth"},
		{"indexTrend", "IndexGrowth"},
	}

	for _, tt := range trendTypes {
		trendInfo, ok := trendData[tt.key].(map[string]interface{})
		if !ok {
			continue
		}

		estimates, ok := trendInfo["estimates"].([]interface{})
		if !ok {
			continue
		}

		for _, est := range estimates {
			estMap, ok := est.(map[string]interface{})
			if !ok {
				continue
			}

			period := getString(estMap, "period")
			if period == "" {
				continue
			}

			growth := estMap["growth"]

			ge, exists := estimateMap[period]
			if !exists {
				ge = &models.GrowthEstimate{Period: period}
				estimateMap[period] = ge
			}

			if growth != nil {
				growthVal, _ := growth.(float64)
				switch tt.key {
				case "industryTrend":
					ge.IndustryGrowth = &growthVal
				case "sectorTrend":
					ge.SectorGrowth = &growthVal
				case "indexTrend":
					ge.IndexGrowth = &growthVal
				}
			}
		}
	}

	// Convert map to slice
	result := make([]models.GrowthEstimate, 0, len(estimateMap))
	for _, ge := range estimateMap {
		result = append(result, *ge)
	}

	return result
}

// Helper functions for parsing (uses getString/getInt from info.go)

func getNestedFloat(m map[string]interface{}, key string) float64 {
	if nested, ok := m[key].(map[string]interface{}); ok {
		if v, ok := nested["raw"].(float64); ok {
			return v
		}
	}
	// Also try direct value
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func getNestedInt(m map[string]interface{}, key string) int {
	if nested, ok := m[key].(map[string]interface{}); ok {
		if v, ok := nested["raw"].(float64); ok {
			return int(v)
		}
	}
	// Also try direct value
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getNestedFloatPtr(m map[string]interface{}, key string) *float64 {
	if nested, ok := m[key].(map[string]interface{}); ok {
		if v, ok := nested["raw"].(float64); ok {
			return &v
		}
	}
	if v, ok := m[key].(float64); ok {
		return &v
	}
	return nil
}
