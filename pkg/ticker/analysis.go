package ticker

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/wnjoon/go-yfinance/internal/endpoints"
	"github.com/wnjoon/go-yfinance/pkg/config"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// epsMethodologyModuleSuffix maps an EPS methodology to the quoteSummary
// module-name suffix Yahoo uses for it, e.g. "earningsTrendGaap" and
// "earningsTrendNonGaap". The default methodology uses the unsuffixed module.
var epsMethodologyModuleSuffix = map[models.EPSMethodology]string{
	models.EPSMethodologyDefault:    "",
	models.EPSMethodologyGAAP:       "Gaap",
	models.EPSMethodologyNormalized: "NonGaap",
}

// analysisCache stores cached analysis data.
//
// All fields are guarded by Ticker.mu. EPS tables are cached per normalized
// EPS methodology, mirroring Python yfinance's per-methodology dict caches.
type analysisCache struct {
	recommendations  *models.RecommendationTrend
	priceTarget      *models.PriceTarget
	revenueEstimates []models.RevenueEstimate
	growthEstimates  []models.GrowthEstimate

	earningsEstimates map[models.EPSMethodology][]models.EarningsEstimate
	epsTrends         map[models.EPSMethodology][]models.EPSTrend
	epsRevisions      map[models.EPSMethodology][]models.EPSRevision
	earningsHistory   map[models.EPSMethodology]*models.EarningsHistory

	// earningsTrend caches the whole earningsTrend{,Gaap,NonGaap} module
	// (the "trend" list plus "defaultMethodology") per methodology.
	earningsTrend map[models.EPSMethodology]map[string]interface{}
}

// readAnalysisCache runs read against the analysis cache (if any) under the
// read lock and returns the cache generation observed at that moment.
func (t *Ticker) readAnalysisCache(read func(c *analysisCache)) uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.analysisCache != nil {
		read(t.analysisCache)
	}
	return t.cacheGeneration
}

// writeAnalysisCache runs write against the analysis cache under the write
// lock, unless ClearCache ran since generation gen was observed.
func (t *Ticker) writeAnalysisCache(gen uint64, write func(c *analysisCache)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cacheGeneration != gen {
		return
	}
	if t.analysisCache == nil {
		t.analysisCache = &analysisCache{}
	}
	write(t.analysisCache)
}

// cachedEPSTable returns the value cached for methodology in the map selected
// by slot, building and caching it on a miss. The methodology is validated
// (case-insensitively) before any cache lookup or request.
func cachedEPSTable[T any](
	t *Ticker,
	methodology models.EPSMethodology,
	slot func(c *analysisCache) *map[models.EPSMethodology]T,
	build func(m models.EPSMethodology) (T, error),
) (T, error) {
	var zero T
	m, err := models.ParseEPSMethodology(string(methodology))
	if err != nil {
		return zero, err
	}

	var cached T
	var hit bool
	gen := t.readAnalysisCache(func(c *analysisCache) {
		cached, hit = (*slot(c))[m]
	})
	if hit {
		return cached, nil
	}

	result, err := build(m)
	if err != nil {
		return zero, err
	}

	t.writeAnalysisCache(gen, func(c *analysisCache) {
		tables := slot(c)
		if *tables == nil {
			*tables = make(map[models.EPSMethodology]T)
		}
		(*tables)[m] = result
	})
	return result, nil
}

// Recommendations returns analyst recommendation trends.
func (t *Ticker) Recommendations() (*models.RecommendationTrend, error) {
	var cached *models.RecommendationTrend
	gen := t.readAnalysisCache(func(c *analysisCache) { cached = c.recommendations })
	if cached != nil {
		return cached, nil
	}

	data, err := t.fetchQuoteSummary([]string{"recommendationTrend"})
	if err != nil {
		return nil, err
	}

	result, err := t.parseRecommendations(data)
	if err != nil {
		return nil, err
	}

	t.writeAnalysisCache(gen, func(c *analysisCache) { c.recommendations = result })
	return result, nil
}

// AnalystPriceTargets returns analyst price targets.
// This method name matches Python yfinance's ticker.analyst_price_targets property.
func (t *Ticker) AnalystPriceTargets() (*models.PriceTarget, error) {
	var cached *models.PriceTarget
	gen := t.readAnalysisCache(func(c *analysisCache) { cached = c.priceTarget })
	if cached != nil {
		return cached, nil
	}

	data, err := t.fetchQuoteSummary([]string{"financialData"})
	if err != nil {
		return nil, err
	}

	result, err := t.parsePriceTarget(data)
	if err != nil {
		return nil, err
	}

	t.writeAnalysisCache(gen, func(c *analysisCache) { c.priceTarget = result })
	return result, nil
}

// PriceTarget is a deprecated alias for AnalystPriceTargets.
// Deprecated: Use AnalystPriceTargets() instead for Python yfinance compatibility.
func (t *Ticker) PriceTarget() (*models.PriceTarget, error) {
	return t.AnalystPriceTargets()
}

// EarningsEstimate returns earnings estimates for upcoming periods using
// Yahoo's default EPS methodology for the ticker.
// This method name matches Python yfinance's ticker.earnings_estimate property.
//
// It is equivalent to EarningsEstimateWithMethodology(models.EPSMethodologyDefault).
func (t *Ticker) EarningsEstimate() ([]models.EarningsEstimate, error) {
	return t.EarningsEstimateWithMethodology(models.EPSMethodologyDefault)
}

// EarningsEstimateWithMethodology returns earnings estimates for upcoming
// periods using the given EPS methodology.
// This matches Python yfinance's ticker.get_earnings_estimate(methodology=...).
//
// methodology is matched case-insensitively; an unknown value returns an error
// wrapping [models.ErrInvalidEPSMethodology] without making a request. Each
// row's Methodology is the requested methodology, or, for the default, the
// one Yahoo reports for the ticker. Results are cached per methodology.
func (t *Ticker) EarningsEstimateWithMethodology(methodology models.EPSMethodology) ([]models.EarningsEstimate, error) {
	return cachedEPSTable(t, methodology,
		func(c *analysisCache) *map[models.EPSMethodology][]models.EarningsEstimate {
			return &c.earningsEstimates
		},
		func(m models.EPSMethodology) ([]models.EarningsEstimate, error) {
			module, err := t.earningsTrendModule(m)
			if err != nil {
				return nil, err
			}
			return parseEarningsEstimates(module, m), nil
		})
}

// EarningsEstimates is a deprecated alias for EarningsEstimate.
// Deprecated: Use EarningsEstimate() instead for Python yfinance compatibility.
func (t *Ticker) EarningsEstimates() ([]models.EarningsEstimate, error) {
	return t.EarningsEstimate()
}

// RevenueEstimate returns revenue estimates for upcoming periods.
// This method name matches Python yfinance's ticker.revenue_estimate property.
//
// Revenue estimates have no EPS methodology; they always come from the
// default earningsTrend module.
func (t *Ticker) RevenueEstimate() ([]models.RevenueEstimate, error) {
	var cached []models.RevenueEstimate
	gen := t.readAnalysisCache(func(c *analysisCache) { cached = c.revenueEstimates })
	if cached != nil {
		return cached, nil
	}

	module, err := t.earningsTrendModule(models.EPSMethodologyDefault)
	if err != nil {
		return nil, err
	}

	result := parseRevenueEstimates(module)

	t.writeAnalysisCache(gen, func(c *analysisCache) { c.revenueEstimates = result })
	return result, nil
}

// RevenueEstimates is a deprecated alias for RevenueEstimate.
// Deprecated: Use RevenueEstimate() instead for Python yfinance compatibility.
func (t *Ticker) RevenueEstimates() ([]models.RevenueEstimate, error) {
	return t.RevenueEstimate()
}

// EPSTrend returns EPS trend data using Yahoo's default EPS methodology.
//
// It is equivalent to EPSTrendWithMethodology(models.EPSMethodologyDefault).
func (t *Ticker) EPSTrend() ([]models.EPSTrend, error) {
	return t.EPSTrendWithMethodology(models.EPSMethodologyDefault)
}

// EPSTrendWithMethodology returns EPS trend data using the given EPS
// methodology.
// This matches Python yfinance's ticker.get_eps_trend(methodology=...).
//
// See [Ticker.EarningsEstimateWithMethodology] for methodology handling.
func (t *Ticker) EPSTrendWithMethodology(methodology models.EPSMethodology) ([]models.EPSTrend, error) {
	return cachedEPSTable(t, methodology,
		func(c *analysisCache) *map[models.EPSMethodology][]models.EPSTrend { return &c.epsTrends },
		func(m models.EPSMethodology) ([]models.EPSTrend, error) {
			module, err := t.earningsTrendModule(m)
			if err != nil {
				return nil, err
			}
			return parseEPSTrends(module, m), nil
		})
}

// EPSRevisions returns EPS revision data using Yahoo's default EPS methodology.
//
// It is equivalent to EPSRevisionsWithMethodology(models.EPSMethodologyDefault).
func (t *Ticker) EPSRevisions() ([]models.EPSRevision, error) {
	return t.EPSRevisionsWithMethodology(models.EPSMethodologyDefault)
}

// EPSRevisionsWithMethodology returns EPS revision data using the given EPS
// methodology.
// This matches Python yfinance's ticker.get_eps_revisions(methodology=...).
//
// See [Ticker.EarningsEstimateWithMethodology] for methodology handling.
func (t *Ticker) EPSRevisionsWithMethodology(methodology models.EPSMethodology) ([]models.EPSRevision, error) {
	return cachedEPSTable(t, methodology,
		func(c *analysisCache) *map[models.EPSMethodology][]models.EPSRevision { return &c.epsRevisions },
		func(m models.EPSMethodology) ([]models.EPSRevision, error) {
			module, err := t.earningsTrendModule(m)
			if err != nil {
				return nil, err
			}
			return parseEPSRevisions(module, m), nil
		})
}

// EarningsHistory returns historical earnings data using Yahoo's default EPS
// methodology.
//
// It is equivalent to EarningsHistoryWithMethodology(models.EPSMethodologyDefault).
func (t *Ticker) EarningsHistory() (*models.EarningsHistory, error) {
	return t.EarningsHistoryWithMethodology(models.EPSMethodologyDefault)
}

// EarningsHistoryWithMethodology returns historical earnings data using the
// given EPS methodology, fetched from the earningsHistory, earningsHistoryGaap,
// or earningsHistoryNonGaap quoteSummary module.
// This matches Python yfinance's ticker.get_earnings_history(methodology=...).
//
// See [Ticker.EarningsEstimateWithMethodology] for methodology handling.
func (t *Ticker) EarningsHistoryWithMethodology(methodology models.EPSMethodology) (*models.EarningsHistory, error) {
	return cachedEPSTable(t, methodology,
		func(c *analysisCache) *map[models.EPSMethodology]*models.EarningsHistory { return &c.earningsHistory },
		func(m models.EPSMethodology) (*models.EarningsHistory, error) {
			module := epsModuleName("earningsHistory", m)
			data, err := t.fetchQuoteSummary([]string{module})
			if err != nil {
				return nil, err
			}
			return parseEarningsHistory(data, module, m)
		})
}

// GrowthEstimates returns growth estimates comparing stock to industry/sector/index.
//
// Growth estimates have no EPS methodology; the stock growth always comes
// from the default earningsTrend module.
func (t *Ticker) GrowthEstimates() ([]models.GrowthEstimate, error) {
	var cached []models.GrowthEstimate
	gen := t.readAnalysisCache(func(c *analysisCache) { cached = c.growthEstimates })
	if cached != nil {
		return cached, nil
	}

	module, err := t.earningsTrendModule(models.EPSMethodologyDefault)
	if err != nil {
		return nil, err
	}

	// Fetch additional trend data
	trendData, err := t.fetchQuoteSummary([]string{"industryTrend", "sectorTrend", "indexTrend"})
	if err != nil {
		return nil, err
	}

	result := parseGrowthEstimates(module, trendData)

	t.writeAnalysisCache(gen, func(c *analysisCache) { c.growthEstimates = result })
	return result, nil
}

// epsModuleName returns the quoteSummary module name for base
// ("earningsTrend" or "earningsHistory") and a normalized methodology.
func epsModuleName(base string, m models.EPSMethodology) string {
	return base + epsMethodologyModuleSuffix[m]
}

// earningsTrendModule returns the cached earningsTrend module for a normalized
// methodology, fetching earningsTrend, earningsTrendGaap, or
// earningsTrendNonGaap on a miss. A response without the module is cached as
// an empty module, so the EPS tables come back empty without refetching.
func (t *Ticker) earningsTrendModule(m models.EPSMethodology) (map[string]interface{}, error) {
	var module map[string]interface{}
	var hit bool
	gen := t.readAnalysisCache(func(c *analysisCache) { module, hit = c.earningsTrend[m] })
	if hit {
		return module, nil
	}

	name := epsModuleName("earningsTrend", m)
	data, err := t.fetchQuoteSummary([]string{name})
	if err != nil {
		return nil, err
	}

	module, ok := data[name].(map[string]interface{})
	if !ok {
		module = map[string]interface{}{}
	}

	t.writeAnalysisCache(gen, func(c *analysisCache) {
		if c.earningsTrend == nil {
			c.earningsTrend = make(map[models.EPSMethodology]map[string]interface{})
		}
		c.earningsTrend[m] = module
	})
	return module, nil
}

// epsMethodologyLabel returns the methodology reported on EPS rows: the
// requested methodology, else the module's defaultMethodology with Yahoo's
// "nongaap" reported as "normalized". Unknown defaults pass through
// unchanged; a missing or non-string default yields "".
func epsMethodologyLabel(m models.EPSMethodology, module map[string]interface{}) string {
	if m != models.EPSMethodologyDefault {
		return string(m)
	}
	def, _ := module["defaultMethodology"].(string)
	switch def {
	case "nongaap":
		return string(models.EPSMethodologyNormalized)
	case "gaap":
		return string(models.EPSMethodologyGAAP)
	default:
		return def
	}
}

// trendItems returns the "trend" list of an earningsTrend module, or nil.
func trendItems(module map[string]interface{}) ([]interface{}, bool) {
	trend, ok := module["trend"].([]interface{})
	return trend, ok
}

// estimateHasData reports whether an earningsTrend sub-object carries at least
// one value besides its currency key. Python yfinance only adds the
// methodology column when the table has data columns.
func estimateHasData(est map[string]interface{}, currencyKey string) bool {
	for k, v := range est {
		if k == currencyKey {
			continue
		}
		switch val := v.(type) {
		case map[string]interface{}:
			if len(val) > 0 {
				return true
			}
		case float64:
			return true
		}
	}
	return false
}

// fetchQuoteSummary fetches data from quoteSummary API.
func (t *Ticker) fetchQuoteSummary(modules []string) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("%s/%s", endpoints.QuoteSummaryURL, t.symbol)

	params := url.Values{}
	params.Set("modules", strings.Join(modules, ","))
	params.Set("corsDomain", "finance.yahoo.com")
	params.Set("formatted", "false")
	lang, region := config.Get().GetLocale()
	params.Set("lang", lang)
	params.Set("region", region)

	get := t.auth.GetWithCrumb
	if t.quoteSummaryGetter != nil {
		get = t.quoteSummaryGetter
	}

	resp, err := get(apiURL, params)
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

// parseEarningsEstimates parses earnings estimates from an earningsTrend module.
func parseEarningsEstimates(module map[string]interface{}, m models.EPSMethodology) []models.EarningsEstimate {
	trend, ok := trendItems(module)
	if !ok {
		return nil
	}

	result := make([]models.EarningsEstimate, 0)
	hasData := false

	for _, item := range trend {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		earningsEst, ok := itemMap["earningsEstimate"].(map[string]interface{})
		if !ok {
			continue
		}
		hasData = hasData || estimateHasData(earningsEst, "earningsCurrency")

		est := models.EarningsEstimate{
			Period:           getString(itemMap, "period"),
			EndDate:          getString(itemMap, "endDate"),
			Currency:         getNestedString(earningsEst, "earningsCurrency"),
			NumberOfAnalysts: getNestedInt(earningsEst, "numberOfAnalysts"),
			Avg:              getNestedFloat(earningsEst, "avg"),
			Low:              getNestedFloat(earningsEst, "low"),
			High:             getNestedFloat(earningsEst, "high"),
			YearAgoEPS:       getNestedFloat(earningsEst, "yearAgoEps"),
			Growth:           getNestedFloat(earningsEst, "growth"),
		}
		result = append(result, est)
	}

	if hasData {
		label := epsMethodologyLabel(m, module)
		for i := range result {
			result[i].Methodology = label
		}
	}

	return result
}

// parseRevenueEstimates parses revenue estimates from an earningsTrend module.
func parseRevenueEstimates(module map[string]interface{}) []models.RevenueEstimate {
	trend, ok := trendItems(module)
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
			Currency:         getNestedString(revenueEst, "revenueCurrency"),
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

// parseEPSTrends parses EPS trends from an earningsTrend module.
func parseEPSTrends(module map[string]interface{}, m models.EPSMethodology) []models.EPSTrend {
	trend, ok := trendItems(module)
	if !ok {
		return nil
	}

	result := make([]models.EPSTrend, 0)
	hasData := false

	for _, item := range trend {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		epsTrend, ok := itemMap["epsTrend"].(map[string]interface{})
		if !ok {
			continue
		}
		hasData = hasData || estimateHasData(epsTrend, "epsTrendCurrency")

		est := models.EPSTrend{
			Period:     getString(itemMap, "period"),
			Currency:   getNestedString(epsTrend, "epsTrendCurrency"),
			Current:    getNestedFloat(epsTrend, "current"),
			SevenDays:  getNestedFloat(epsTrend, "7daysAgo"),
			ThirtyDays: getNestedFloat(epsTrend, "30daysAgo"),
			SixtyDays:  getNestedFloat(epsTrend, "60daysAgo"),
			NinetyDays: getNestedFloat(epsTrend, "90daysAgo"),
		}
		result = append(result, est)
	}

	if hasData {
		label := epsMethodologyLabel(m, module)
		for i := range result {
			result[i].Methodology = label
		}
	}

	return result
}

// parseEPSRevisions parses EPS revisions from an earningsTrend module.
func parseEPSRevisions(module map[string]interface{}, m models.EPSMethodology) []models.EPSRevision {
	trend, ok := trendItems(module)
	if !ok {
		return nil
	}

	result := make([]models.EPSRevision, 0)
	hasData := false

	for _, item := range trend {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		epsRevisions, ok := itemMap["epsRevisions"].(map[string]interface{})
		if !ok {
			continue
		}
		hasData = hasData || estimateHasData(epsRevisions, "epsRevisionsCurrency")

		rev := models.EPSRevision{
			Period:         getString(itemMap, "period"),
			Currency:       getNestedString(epsRevisions, "epsRevisionsCurrency"),
			UpLast7Days:    getNestedInt(epsRevisions, "upLast7days"),
			UpLast30Days:   getNestedInt(epsRevisions, "upLast30days"),
			DownLast7Days:  getNestedInt(epsRevisions, "downLast7days"),
			DownLast30Days: getNestedInt(epsRevisions, "downLast30days"),
		}
		result = append(result, rev)
	}

	if hasData {
		label := epsMethodologyLabel(m, module)
		for i := range result {
			result[i].Methodology = label
		}
	}

	return result
}

// parseEarningsHistory parses an earningsHistory{,Gaap,NonGaap} module.
func parseEarningsHistory(data map[string]interface{}, moduleName string, m models.EPSMethodology) (*models.EarningsHistory, error) {
	ehData, ok := data[moduleName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s not found", moduleName)
	}

	history, ok := ehData["history"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("history data not found")
	}

	result := &models.EarningsHistory{
		History: make([]models.EarningsHistoryItem, 0, len(history)),
	}
	label := epsMethodologyLabel(m, ehData)

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
			Methodology:     label,
		}
		result.History = append(result.History, histItem)
	}

	return result, nil
}

// parseGrowthEstimates parses growth estimates from the default earningsTrend
// module and the industry/sector/index trend modules.
func parseGrowthEstimates(module map[string]interface{}, trendData map[string]interface{}) []models.GrowthEstimate {
	trend, ok := trendItems(module)
	if !ok {
		return nil
	}

	// Build map of period -> growth estimate
	estimateMap := make(map[string]*models.GrowthEstimate)

	// Parse stock growth from earningsTrend
	parseStockGrowthEstimates(trend, estimateMap)

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
		parsePeerGrowthEstimates(trendData, tt.key, tt.field, estimateMap)
	}

	// Convert map to slice
	result := make([]models.GrowthEstimate, 0, len(estimateMap))
	for _, ge := range estimateMap {
		result = append(result, *ge)
	}

	return result
}

func parseStockGrowthEstimates(trend []interface{}, estimateMap map[string]*models.GrowthEstimate) {
	for _, item := range trend {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		period := getString(itemMap, "period")
		if period == "" {
			continue
		}
		estimateMap[period] = &models.GrowthEstimate{
			Period:      period,
			StockGrowth: getNestedFloatPtr(itemMap, "growth"),
		}
	}
}

func parsePeerGrowthEstimates(trendData map[string]interface{}, trendKey, field string, estimateMap map[string]*models.GrowthEstimate) {
	trendInfo, ok := trendData[trendKey].(map[string]interface{})
	if !ok {
		return
	}
	estimates, ok := trendInfo["estimates"].([]interface{})
	if !ok {
		return
	}
	for _, est := range estimates {
		applyPeerGrowthEstimate(est, field, estimateMap)
	}
}

func applyPeerGrowthEstimate(est interface{}, field string, estimateMap map[string]*models.GrowthEstimate) {
	estMap, ok := est.(map[string]interface{})
	if !ok {
		return
	}
	period := getString(estMap, "period")
	if period == "" {
		return
	}
	growthVal, ok := estMap["growth"].(float64)
	if !ok {
		return
	}
	ge, exists := estimateMap[period]
	if !exists {
		ge = &models.GrowthEstimate{Period: period}
		estimateMap[period] = ge
	}
	switch field {
	case "IndustryGrowth":
		ge.IndustryGrowth = &growthVal
	case "SectorGrowth":
		ge.SectorGrowth = &growthVal
	case "IndexGrowth":
		ge.IndexGrowth = &growthVal
	}
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

func getNestedString(m map[string]interface{}, key string) string {
	if nested, ok := m[key].(map[string]interface{}); ok {
		if v, ok := nested["raw"].(string); ok {
			return v
		}
		if v, ok := nested["fmt"].(string); ok {
			return v
		}
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
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
