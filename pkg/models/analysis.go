// Package models provides data structures for Yahoo Finance API responses.
package models

import "time"

// RecommendationTrend represents analyst recommendations over time.
type RecommendationTrend struct {
	Trend []Recommendation `json:"trend"`
}

// Recommendation represents analyst recommendation counts for a period.
type Recommendation struct {
	Period     string `json:"period"` // "0m", "-1m", "-2m", "-3m"
	StrongBuy  int    `json:"strongBuy"`
	Buy        int    `json:"buy"`
	Hold       int    `json:"hold"`
	Sell       int    `json:"sell"`
	StrongSell int    `json:"strongSell"`
}

// Total returns total number of recommendations.
func (r *Recommendation) Total() int {
	return r.StrongBuy + r.Buy + r.Hold + r.Sell + r.StrongSell
}

// PriceTarget represents analyst price targets.
type PriceTarget struct {
	Current         float64 `json:"current"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	Mean            float64 `json:"mean"`
	Median          float64 `json:"median"`
	NumberOfAnalysts int    `json:"numberOfAnalysts"`
	// Additional recommendation data
	RecommendationKey  string  `json:"recommendationKey"`  // "buy", "hold", "sell"
	RecommendationMean float64 `json:"recommendationMean"` // 1.0 (strong buy) to 5.0 (strong sell)
}

// EarningsEstimate represents earnings estimates for a period.
type EarningsEstimate struct {
	Period           string  `json:"period"` // "0q", "+1q", "0y", "+1y"
	EndDate          string  `json:"endDate"`
	NumberOfAnalysts int     `json:"numberOfAnalysts"`
	Avg              float64 `json:"avg"`
	Low              float64 `json:"low"`
	High             float64 `json:"high"`
	YearAgoEPS       float64 `json:"yearAgoEps"`
	Growth           float64 `json:"growth"` // as decimal (0.15 = 15%)
}

// RevenueEstimate represents revenue estimates for a period.
type RevenueEstimate struct {
	Period           string  `json:"period"`
	EndDate          string  `json:"endDate"`
	NumberOfAnalysts int     `json:"numberOfAnalysts"`
	Avg              float64 `json:"avg"`
	Low              float64 `json:"low"`
	High             float64 `json:"high"`
	YearAgoRevenue   float64 `json:"yearAgoRevenue"`
	Growth           float64 `json:"growth"`
}

// EPSTrend represents EPS trend data for a period.
type EPSTrend struct {
	Period    string  `json:"period"`
	Current   float64 `json:"current"`
	SevenDays float64 `json:"7daysAgo"`
	ThirtyDays float64 `json:"30daysAgo"`
	SixtyDays float64 `json:"60daysAgo"`
	NinetyDays float64 `json:"90daysAgo"`
}

// EPSRevision represents EPS revision data for a period.
type EPSRevision struct {
	Period        string `json:"period"`
	UpLast7Days   int    `json:"upLast7days"`
	UpLast30Days  int    `json:"upLast30days"`
	DownLast7Days int    `json:"downLast7days"`
	DownLast30Days int   `json:"downLast30days"`
}

// EarningsHistory represents historical earnings data.
type EarningsHistory struct {
	History []EarningsHistoryItem `json:"history"`
}

// EarningsHistoryItem represents a single earnings report.
type EarningsHistoryItem struct {
	Period          string    `json:"period"` // "-1q", "-2q", etc.
	Quarter         time.Time `json:"quarter"`
	EPSActual       float64   `json:"epsActual"`
	EPSEstimate     float64   `json:"epsEstimate"`
	EPSDifference   float64   `json:"epsDifference"`
	SurprisePercent float64   `json:"surprisePercent"` // as decimal
}

// GrowthEstimate represents growth estimates from various sources.
type GrowthEstimate struct {
	Period        string   `json:"period"`
	StockGrowth   *float64 `json:"stockGrowth,omitempty"`   // nil if not available
	IndustryGrowth *float64 `json:"industryGrowth,omitempty"`
	SectorGrowth  *float64 `json:"sectorGrowth,omitempty"`
	IndexGrowth   *float64 `json:"indexGrowth,omitempty"`
}

// AnalysisData holds all analysis data for a ticker.
type AnalysisData struct {
	Recommendations   *RecommendationTrend   `json:"recommendations,omitempty"`
	PriceTarget       *PriceTarget           `json:"priceTarget,omitempty"`
	EarningsEstimates []EarningsEstimate     `json:"earningsEstimates,omitempty"`
	RevenueEstimates  []RevenueEstimate      `json:"revenueEstimates,omitempty"`
	EPSTrends         []EPSTrend             `json:"epsTrends,omitempty"`
	EPSRevisions      []EPSRevision          `json:"epsRevisions,omitempty"`
	EarningsHistory   *EarningsHistory       `json:"earningsHistory,omitempty"`
	GrowthEstimates   []GrowthEstimate       `json:"growthEstimates,omitempty"`
}
