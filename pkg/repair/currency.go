package repair

import (
	"github.com/wnjoon/go-yfinance/pkg/models"
)

// currencyConversion describes a sub-unit currency and its main-unit equivalent.
type currencyConversion struct {
	factor   float64 // multiplier from sub-unit to main unit (GBp -> GBP is 0.01)
	standard string  // main-unit currency code
}

// currencyConversions mirrors python yfinance's _CURRENCY_CONVERSIONS: currencies
// Yahoo quotes in sub-units, which repair math must standardise to the main unit.
var currencyConversions = map[string]currencyConversion{
	"GBp": {factor: 0.01, standard: "GBP"}, // British pence
	"ZAc": {factor: 0.01, standard: "ZAR"}, // South African cents
	"ILA": {factor: 0.01, standard: "ILS"}, // Israeli agorot
}

// standardiseCurrency converts sub-unit prices (GBp/ZAc/ILA) to the main
// currency so repair math runs in the unit Yahoo's reference data uses.
// Returns the converted bars, the currency the repair should assume, and
// whether scaling was applied.
//
// Prices are standardised unconditionally: upstream skips this when a fresh
// quote shows Yahoo already returned main-unit prices, but go-yfinance's
// repair layer has no access to that live quote (documented gap in the
// v1.6.0 progress notes) — the common case (prices genuinely in sub-units)
// is unaffected.
func standardiseCurrency(bars []models.Bar, currency string) ([]models.Bar, string, bool) {
	conv, ok := currencyConversions[currency]
	if !ok {
		return bars, currency, false
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)
	for i := range result {
		scalePrice(&result[i], conv.factor)
	}

	// Dividends are scaled only when they still look like they're expressed
	// in the sub-unit: some LSE-style tickers report dividends already in
	// the main currency even though prices are in pence (upstream #2907).
	// Heuristic (matches upstream): compare each dividend to the PRIOR
	// bar's now-standardised Close; if the average ratio exceeds 1 (an
	// implausible >100% yield unless the dividend is itself ~100x too
	// large), the dividend needs the same scaling as prices.
	if avg, ok := averageDividendToPrevCloseRatio(result); ok && avg > 1 {
		for i := range result {
			result[i].Dividends *= conv.factor
		}
	}

	return result, conv.standard, true
}

// averageDividendToPrevCloseRatio returns the mean of Dividends/prevClose
// across bars with a nonzero dividend, using the prior bar's Close (or the
// bar's own Close for the first bar, matching upstream's fill_value).
func averageDividendToPrevCloseRatio(bars []models.Bar) (float64, bool) {
	var sum float64
	var count int
	for i, bar := range bars {
		if bar.Dividends == 0 {
			continue
		}
		prevClose := bar.Close
		if i > 0 {
			prevClose = bars[i-1].Close
		}
		if prevClose == 0 {
			continue
		}
		sum += bar.Dividends / prevClose
		count++
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

// revertCurrency undoes standardiseCurrency so user-visible prices stay in the
// original sub-unit. Prices and dividends are both reverted unconditionally:
// after dividend repair they are always in the same unit (upstream #2907
// regression fix — repair=True must not permanently convert GBp/ZAc/ILA to
// main currency), regardless of whether standardiseCurrency scaled the
// dividend on entry.
func revertCurrency(bars []models.Bar, originalCurrency string) []models.Bar {
	conv, ok := currencyConversions[originalCurrency]
	if !ok {
		return bars
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)
	for i := range result {
		scalePrice(&result[i], 1/conv.factor)
		result[i].Dividends /= conv.factor
	}
	return result
}

func scalePrice(bar *models.Bar, m float64) {
	bar.Open *= m
	bar.High *= m
	bar.Low *= m
	bar.Close *= m
	bar.AdjClose *= m
}

// currencySubUnitDivisor returns the sub-unit factor used by 100x unit-mixup
// and dividend repairs. The Kuwaiti dinar divides into 1000 fils, not 100.
func currencySubUnitDivisor(currency string) float64 {
	if currency == "KWF" {
		return 1000.0
	}
	return 100.0
}
