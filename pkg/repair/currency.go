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
func standardiseCurrency(bars []models.Bar, currency string) ([]models.Bar, string, bool) {
	conv, ok := currencyConversions[currency]
	if !ok {
		return bars, currency, false
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)
	for i := range result {
		scaleBarPrices(&result[i], conv.factor)
	}
	return result, conv.standard, true
}

// revertCurrency undoes standardiseCurrency so user-visible prices stay in the
// original sub-unit. Dividends are reverted unconditionally: after dividend
// repair they are always in the same unit as prices (upstream #2907 regression
// fix — repair=True must not permanently convert GBp/ZAc/ILA to main currency).
func revertCurrency(bars []models.Bar, originalCurrency string) []models.Bar {
	conv, ok := currencyConversions[originalCurrency]
	if !ok {
		return bars
	}

	result := make([]models.Bar, len(bars))
	copy(result, bars)
	for i := range result {
		scaleBarPrices(&result[i], 1/conv.factor)
	}
	return result
}

func scaleBarPrices(bar *models.Bar, m float64) {
	bar.Open *= m
	bar.High *= m
	bar.Low *= m
	bar.Close *= m
	bar.AdjClose *= m
	bar.Dividends *= m
}

// currencySubUnitDivisor returns the sub-unit factor used by 100x unit-mixup
// and dividend repairs. The Kuwaiti dinar divides into 1000 fils, not 100.
func currencySubUnitDivisor(currency string) float64 {
	if currency == "KWF" {
		return 1000.0
	}
	return 100.0
}
