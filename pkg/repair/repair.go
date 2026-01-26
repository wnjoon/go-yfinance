package repair

import (
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
type Options struct {
	// Data context
	Ticker   string    // Ticker symbol
	Interval string    // Data interval (1d, 1wk, 1mo, etc.)
	Timezone string    // Exchange timezone
	Currency string    // Price currency
	QuoteType QuoteType // Type of instrument (EQUITY, ETF, MUTUALFUND, etc.)

	// Feature flags - which repairs to apply
	FixUnitMixups   bool // Fix 100x currency errors ($/cents, £/pence)
	FixZeroes       bool // Fix missing/zero values
	FixSplits       bool // Fix bad stock split adjustments
	FixDividends    bool // Fix bad dividend adjustments
	FixCapitalGains bool // Fix capital gains double-counting (ETF/MutualFund only)
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
	}
}

// Repairer handles price data repair operations.
type Repairer struct {
	opts Options
}

// New creates a new Repairer with the given options.
func New(opts Options) *Repairer {
	return &Repairer{
		opts: opts,
	}
}

// Repair applies all enabled repair operations to the bar data.
// The order of operations matters:
//  1. Fix dividend adjustments (must come before price-level errors)
//  2. Fix 100x unit errors
//  3. Fix stock split errors
//  4. Fix zero/missing values
//  5. Fix capital gains double-counting (last, needs clean adjustment data)
//
// Returns the repaired bars and any error encountered.
func (r *Repairer) Repair(bars []models.Bar) ([]models.Bar, error) {
	if len(bars) == 0 {
		return bars, nil
	}

	// Make a copy to avoid modifying the original
	result := make([]models.Bar, len(bars))
	copy(result, bars)

	// Apply repairs in order (order matters!)
	// 1. Dividend adjustments first
	if r.opts.FixDividends {
		result = r.repairDividends(result)
	}

	// 2. 100x unit errors
	if r.opts.FixUnitMixups {
		result = r.repairUnitMixups(result)
	}

	// 3. Stock split errors
	if r.opts.FixSplits {
		result = r.repairStockSplits(result)
	}

	// 4. Zero/missing values
	if r.opts.FixZeroes {
		result = r.repairZeroes(result)
	}

	// 5. Capital gains double-counting (only for ETF/MutualFund)
	if r.opts.FixCapitalGains && r.isCapitalGainsApplicable() {
		result = r.repairCapitalGains(result)
	}

	return result, nil
}

// isCapitalGainsApplicable returns true if capital gains repair should be applied.
// Only applicable for ETFs and Mutual Funds.
func (r *Repairer) isCapitalGainsApplicable() bool {
	return r.opts.QuoteType == QuoteTypeETF || r.opts.QuoteType == QuoteTypeMutualFund
}

// repairDividends fixes bad dividend adjustments.
// This is a placeholder - full implementation in Phase 4.
func (r *Repairer) repairDividends(bars []models.Bar) []models.Bar {
	// TODO: Implement in Phase 4
	return bars
}

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
