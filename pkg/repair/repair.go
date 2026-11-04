package repair

import (
	"strings"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// QuoteType represents the type of financial instrument.
type QuoteType string

const (
	QuoteTypeEquity     QuoteType = "EQUITY"
	QuoteTypeETF        QuoteType = "ETF"
	QuoteTypeMutualFund QuoteType = "MUTUALFUND"
	QuoteTypeIndex      QuoteType = "INDEX"
	QuoteTypeCurrency   QuoteType = "CURRENCY"
	QuoteTypeCrypto     QuoteType = "CRYPTOCURRENCY"
)

// Options configures the repair behavior.
//
// Repair expects bars prepared as described under "Input Bars" in the
// package documentation: missing prices as NaN, AdjClose 0 meaning
// unadjusted, and the dividend/split/capital-gain fields filled.
type Options struct {
	// Data context
	Ticker    string    // Ticker symbol
	Interval  string    // Data interval (1d, 1wk, 1mo, etc.)
	Timezone  string    // Exchange timezone
	Currency  string    // Price currency
	QuoteType QuoteType // Type of instrument (EQUITY, ETF, MUTUALFUND, etc.)
	PrePost   bool      // Whether pre/post market data is included

	// Feature flags - which repairs to apply
	FixUnitMixups   bool // Fix 100x currency errors ($/cents, £/pence)
	FixZeroes       bool // Fix missing/zero values
	FixSplits       bool // Fix bad stock split adjustments
	FixDividends    bool // Fix bad dividend adjustments
	FixCapitalGains bool // Fix capital gains double-counting (ETF/MutualFund only)

	// FixLevelShifts fixes split-like level shifts in daily data that have
	// no matching split, measuring the multiple with 1h data fetched via
	// FetchHourly. It only applies to interval "1d", tickers without '=',
	// and needs FetchHourly; otherwise it is skipped.
	FixLevelShifts bool

	// FetchHourly fetches the 1h data used by FixLevelShifts. It is called
	// at most once per Repair, and only when a candidate shift is found.
	// nil disables the level-shift repair, so Repair never touches the
	// network on its own. [github.com/wnjoon/go-yfinance/pkg/ticker.Ticker.History]
	// sets it when Repair is requested.
	FetchHourly HourlyFetcher

	// Now returns the current time, used to limit level-shift checks to the
	// last 729 days for which Yahoo serves 1h data. nil uses time.Now.
	Now func() time.Time
}

// DefaultOptions returns options with all repairs enabled.
func DefaultOptions() Options {
	return Options{
		Interval:        "1d",
		FixUnitMixups:   true,
		FixZeroes:       true,
		FixSplits:       true,
		FixDividends:    true,
		FixCapitalGains: true,
		FixLevelShifts:  true,
	}
}

// Repairer handles price data repair operations.
type Repairer struct {
	opts Options

	// currencyRepaired is upstream's history_metadata['currencyRepaired']:
	// prices were standardised from a sub-unit currency during Repair.
	currencyRepaired bool
	// traceShift, when set by tests, receives each fixPricesSuddenChange
	// trace.
	traceShift func(*priceShiftTrace)
}

// New creates a new Repairer with the given options.
func New(opts Options) *Repairer {
	return &Repairer{
		opts: opts,
	}
}

// Repair applies all enabled repair operations to the bar data, which must be
// sorted oldest first (see "Input Bars" in the package documentation).
// The order of operations matters:
//  1. Fix unexplained level shifts (daily data, needs Options.FetchHourly)
//  2. Fix dividend adjustments (must come before price-level errors)
//  3. Fix 100x unit errors
//  4. Fix stock split errors
//  5. Fix zero/missing values
//  6. Fix capital gains double-counting (last, needs clean adjustment data)
//
// Returns the repaired bars and any error encountered. The only error is a
// rate-limit error from Options.FetchHourly (see [HourlyFetcher]), returned
// unchanged with nil bars; other fetch errors skip the level-shift step.
func (r *Repairer) Repair(bars []models.Bar) ([]models.Bar, error) {
	if len(bars) == 0 {
		return bars, nil
	}

	// Make a copy to avoid modifying the original
	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Standardise sub-unit currencies (GBp/ZAc/ILA) so repair math runs in the
	// main currency; reverted before returning so user-visible prices and
	// dividends stay in the original sub-unit (upstream #2907).
	originalCurrency := r.opts.Currency
	var pricesScaled bool
	result, r.opts.Currency, pricesScaled = standardiseCurrency(result, originalCurrency)
	r.currencyRepaired = pricesScaled
	defer func() {
		r.opts.Currency = originalCurrency
		r.currencyRepaired = false
	}()

	// Unit, split and level-shift repairs depend on volume, which is
	// meaningless for FX tickers (upstream #2908: skipped when '=' is in the
	// ticker).
	isFX := strings.Contains(r.opts.Ticker, "=")

	// Apply repairs in order (order matters!)
	// 1. Unexplained level shifts, before dividend repair (upstream 1.8.0)
	if r.opts.FixLevelShifts && !isFX {
		var err error
		result, err = r.repairLevelShifts(result)
		if err != nil {
			return nil, err
		}
	}

	// 2. Dividend adjustments
	if r.opts.FixDividends {
		result = r.repairDividends(result)
	}

	// 3. 100x unit errors
	if r.opts.FixUnitMixups && !isFX {
		result = r.repairUnitMixups(result)
	}

	// 4. Stock split errors
	if r.opts.FixSplits && !isFX {
		result = r.repairStockSplits(result)
	}

	// 5. Zero/missing values
	if r.opts.FixZeroes {
		result = r.repairZeroes(result)
	}

	// 6. Capital gains double-counting (only for ETF/MutualFund)
	if r.opts.FixCapitalGains && r.isCapitalGainsApplicable() {
		result = r.repairCapitalGains(result)
	}

	if pricesScaled {
		result = revertCurrency(result, originalCurrency)
	}

	return result, nil
}

// isCapitalGainsApplicable returns true if capital gains repair should be applied.
// Only applicable for ETFs and Mutual Funds.
func (r *Repairer) isCapitalGainsApplicable() bool {
	return r.opts.QuoteType == QuoteTypeETF || r.opts.QuoteType == QuoteTypeMutualFund
}

// repairLevelShifts is implemented in level_shift.go

// repairDividends is implemented in dividend.go

// repairUnitMixups is implemented in unit_mixup.go

// repairStockSplits is implemented in split.go

// repairZeroes is implemented in zeroes.go

// repairCapitalGains is implemented in capital_gains.go

// HasCapitalGains checks if any bar has capital gains data.
func HasCapitalGains(bars []models.Bar) bool {
	for _, bar := range bars {
		if bar.CapitalGains > 0 {
			return true
		}
	}
	return false
}

// HasDividends checks if any bar has dividend data.
func HasDividends(bars []models.Bar) bool {
	for _, bar := range bars {
		if bar.Dividends > 0 {
			return true
		}
	}
	return false
}

// HasSplits checks if any bar has split data.
func HasSplits(bars []models.Bar) bool {
	for _, bar := range bars {
		if bar.Splits != 0 && bar.Splits != 1 {
			return true
		}
	}
	return false
}

// CountRepaired counts how many bars have been repaired.
func CountRepaired(bars []models.Bar) int {
	count := 0
	for _, bar := range bars {
		if bar.Repaired {
			count++
		}
	}
	return count
}
