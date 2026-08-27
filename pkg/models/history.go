package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// Bar represents a single OHLCV bar (candlestick).
type Bar struct {
	Date      time.Time `json:"date"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	AdjClose  float64   `json:"adjClose"`
	Volume    int64     `json:"volume"`
	Dividends float64   `json:"dividends,omitempty"`
	// DividendCurrency is present when Yahoo returns dividend events in a separate currency.
	DividendCurrency string  `json:"dividendCurrency,omitempty"`
	Splits           float64 `json:"splits,omitempty"`
	CapitalGains     float64 `json:"capitalGains,omitempty"` // Capital gains distribution (ETF/MutualFund)
	Repaired         bool    `json:"repaired,omitempty"`     // True if this bar was repaired
}

// History represents historical price data.
type History struct {
	Symbol   string `json:"symbol"`
	Currency string `json:"currency"`
	Bars     []Bar  `json:"bars"`
}

// HistoryParams represents parameters for fetching historical data.
type HistoryParams struct {
	// Period: 1d, 5d, 1mo, 3mo, 6mo, 1y, 2y, 5y, 10y, ytd, max
	Period string `json:"period,omitempty"`

	// Interval: 1m, 2m, 5m, 15m, 30m, 60m, 90m, 1h, 1d, 5d, 1wk, 1mo, 3mo
	Interval string `json:"interval,omitempty"`

	// Start date (YYYY-MM-DD or time.Time)
	Start *time.Time `json:"start,omitempty"`

	// End date (YYYY-MM-DD or time.Time)
	End *time.Time `json:"end,omitempty"`

	// Include pre/post market data
	PrePost bool `json:"prepost,omitempty"`

	// Automatically adjust OHLC for splits/dividends
	AutoAdjust bool `json:"autoAdjust,omitempty"`

	// Include dividend and split events
	Actions bool `json:"actions,omitempty"`

	// Repair bad data (100x errors, missing data)
	Repair bool `json:"repair,omitempty"`

	// RepairOptions provides fine-grained control over repair operations.
	// If nil, all repairs are enabled when Repair is true.
	RepairOptions *RepairOptions `json:"repairOptions,omitempty"`

	// Keep NaN rows
	KeepNA bool `json:"keepna,omitempty"`
}

// RepairOptions provides fine-grained control over which repairs to apply.
type RepairOptions struct {
	// FixUnitMixups repairs 100x currency errors ($/cents, £/pence)
	FixUnitMixups bool `json:"fixUnitMixups,omitempty"`

	// FixZeroes repairs missing/zero price values
	FixZeroes bool `json:"fixZeroes,omitempty"`

	// FixSplits repairs bad stock split adjustments
	FixSplits bool `json:"fixSplits,omitempty"`

	// FixDividends repairs bad dividend adjustments
	FixDividends bool `json:"fixDividends,omitempty"`

	// FixCapitalGains repairs capital gains double-counting (ETF/MutualFund only)
	FixCapitalGains bool `json:"fixCapitalGains,omitempty"`
}

// DefaultRepairOptions returns options with all repairs enabled.
func DefaultRepairOptions() RepairOptions {
	return RepairOptions{
		FixUnitMixups:   true,
		FixZeroes:       true,
		FixSplits:       true,
		FixDividends:    true,
		FixCapitalGains: true,
	}
}

// DefaultHistoryParams returns default history parameters.
func DefaultHistoryParams() HistoryParams {
	return HistoryParams{
		Period:     "1mo",
		Interval:   "1d",
		PrePost:    false,
		AutoAdjust: true,
		Actions:    true,
		Repair:     false,
		KeepNA:     false,
	}
}

// ChartMeta represents metadata from chart API response.
type ChartMeta struct {
	Currency             string   `json:"currency"`
	Symbol               string   `json:"symbol"`
	ExchangeName         string   `json:"exchangeName"`
	ExchangeTimezoneName string   `json:"exchangeTimezoneName"`
	InstrumentType       string   `json:"instrumentType"`
	FirstTradeDate       int64    `json:"firstTradeDate"`
	RegularMarketTime    int64    `json:"regularMarketTime"`
	GMTOffset            int      `json:"gmtoffset"`
	Timezone             string   `json:"timezone"`
	RegularMarketPrice   float64  `json:"regularMarketPrice"`
	ChartPreviousClose   float64  `json:"chartPreviousClose"`
	PreviousClose        float64  `json:"previousClose"`
	Scale                int      `json:"scale"`
	PriceHint            int      `json:"priceHint"`
	DataGranularity      string   `json:"dataGranularity"`
	Range                string   `json:"range"`
	ValidRanges          []string `json:"validRanges"`
	// TradingPeriods contains the exchange's regular sessions and, when Yahoo
	// supplies them, the corresponding pre- and post-market sessions.
	TradingPeriods []TradingPeriod `json:"tradingPeriods,omitempty"`
}

// TradingPeriod represents one exchange trading session. Start and End are
// Unix seconds. PreStart/PreEnd and PostStart/PostEnd are nil when Yahoo does
// not provide extended-hours sessions for that day.
type TradingPeriod struct {
	Start     int64  `json:"start"`
	End       int64  `json:"end"`
	PreStart  *int64 `json:"preStart,omitempty"`
	PreEnd    *int64 `json:"preEnd,omitempty"`
	PostStart *int64 `json:"postStart,omitempty"`
	PostEnd   *int64 `json:"postEnd,omitempty"`
}

// HasTradingPeriods reports whether Yahoo included a decodable
// tradingPeriods value, including an explicitly empty value.
func (m ChartMeta) HasTradingPeriods() bool { return m.TradingPeriods != nil }

// WithTradingPeriods returns a metadata copy with trading periods populated.
func (m ChartMeta) WithTradingPeriods(periods []TradingPeriod) ChartMeta {
	m.TradingPeriods = periods
	if periods == nil {
		m.TradingPeriods = []TradingPeriod{}
	}
	return m
}

// UnmarshalJSON accepts both Yahoo tradingPeriods encodings: regular-only
// list-of-lists and grouped pre/regular/post list-of-lists.
func (m *ChartMeta) UnmarshalJSON(data []byte) error {
	type chartMetaAlias ChartMeta
	var wire struct {
		chartMetaAlias
		TradingPeriods json.RawMessage `json:"tradingPeriods"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*m = ChartMeta(wire.chartMetaAlias)
	if len(wire.TradingPeriods) == 0 || string(wire.TradingPeriods) == "null" {
		return nil
	}
	periods, err := decodeTradingPeriods(wire.TradingPeriods)
	if err != nil {
		return fmt.Errorf("decode tradingPeriods: %w", err)
	}
	m.TradingPeriods = periods
	return nil
}

// MarshalJSON emits the normalized flat Go representation. UnmarshalJSON also
// accepts this representation so populated metadata round-trips without loss.
func (m ChartMeta) MarshalJSON() ([]byte, error) {
	type chartMetaAlias ChartMeta
	return json.Marshal(chartMetaAlias(m))
}

type yahooTradingPeriod struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

func decodeTradingPeriods(raw json.RawMessage) ([]TradingPeriod, error) {
	var normalized []TradingPeriod
	if err := json.Unmarshal(raw, &normalized); err == nil {
		return normalized, nil
	}

	var regularOnly [][]yahooTradingPeriod
	if err := json.Unmarshal(raw, &regularOnly); err == nil {
		flat := flattenYahooPeriods(regularOnly)
		out := make([]TradingPeriod, 0, len(flat))
		for _, p := range flat {
			if p.Start == 0 && p.End == 0 {
				continue
			}
			out = append(out, TradingPeriod{Start: p.Start, End: p.End})
		}
		return out, nil
	}

	var grouped map[string][][]yahooTradingPeriod
	if err := json.Unmarshal(raw, &grouped); err != nil {
		return nil, err
	}
	regularGroups := grouped["regular"]
	out := make([]TradingPeriod, 0, len(flattenYahooPeriods(regularGroups)))
	for groupIndex, regularGroup := range regularGroups {
		for periodIndex, p := range regularGroup {
			if p.Start == 0 && p.End == 0 {
				continue
			}
			tp := TradingPeriod{Start: p.Start, End: p.End}
			if pre, ok := yahooPeriodAt(grouped["pre"], groupIndex, periodIndex); ok {
				tp.PreStart, tp.PreEnd = int64Ptr(pre.Start), int64Ptr(pre.End)
			}
			if post, ok := yahooPeriodAt(grouped["post"], groupIndex, periodIndex); ok {
				tp.PostStart, tp.PostEnd = int64Ptr(post.Start), int64Ptr(post.End)
			}
			out = append(out, tp)
		}
	}
	return out, nil
}

func yahooPeriodAt(groups [][]yahooTradingPeriod, groupIndex, periodIndex int) (yahooTradingPeriod, bool) {
	if groupIndex >= len(groups) || periodIndex >= len(groups[groupIndex]) {
		return yahooTradingPeriod{}, false
	}
	period := groups[groupIndex][periodIndex]
	return period, period.Start != 0 || period.End != 0
}

func flattenYahooPeriods(groups [][]yahooTradingPeriod) []yahooTradingPeriod {
	var out []yahooTradingPeriod
	for _, group := range groups {
		out = append(out, group...)
	}
	return out
}

func int64Ptr(v int64) *int64 { return &v }

// Dividend represents a dividend payment.
type Dividend struct {
	Date     time.Time `json:"date"`
	Amount   float64   `json:"amount"`
	Currency string    `json:"currency,omitempty"`
}

// Split represents a stock split.
type Split struct {
	Date        time.Time `json:"date"`
	Numerator   float64   `json:"numerator"`
	Denominator float64   `json:"denominator"`
	Ratio       string    `json:"ratio"` // e.g., "4:1"
}

// CapitalGain represents a capital gain distribution.
type CapitalGain struct {
	Date   time.Time `json:"date"`
	Amount float64   `json:"amount"`
}

// Actions represents dividend and split actions.
type Actions struct {
	Dividends    []Dividend    `json:"dividends,omitempty"`
	Splits       []Split       `json:"splits,omitempty"`
	CapitalGains []CapitalGain `json:"capitalGains,omitempty"`
}

// ValidPeriods returns all valid period values.
func ValidPeriods() []string {
	return []string{"1d", "5d", "1mo", "3mo", "6mo", "1y", "2y", "5y", "10y", "ytd", "max"}
}

// ValidIntervals returns all valid interval values.
func ValidIntervals() []string {
	return []string{"1m", "2m", "5m", "15m", "30m", "60m", "90m", "1h", "1d", "5d", "1wk", "1mo", "3mo"}
}

// IsValidPeriod checks if a period string is valid.
func IsValidPeriod(period string) bool {
	for _, p := range ValidPeriods() {
		if p == period {
			return true
		}
	}
	return false
}

// IsValidInterval checks if an interval string is valid.
func IsValidInterval(interval string) bool {
	for _, i := range ValidIntervals() {
		if i == interval {
			return true
		}
	}
	return false
}
