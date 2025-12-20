package ticker

import (
	"fmt"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// holdersCache stores cached holders data.
type holdersCache struct {
	major               *models.MajorHolders
	institutional       []models.Holder
	mutualFund          []models.Holder
	insiderTransactions []models.InsiderTransaction
	insiderRoster       []models.InsiderHolder
	insiderPurchases    *models.InsiderPurchases
}

// holdersModules are the quoteSummary modules for holders data.
var holdersModules = []string{
	"institutionOwnership",
	"fundOwnership",
	"majorDirectHolders",
	"majorHoldersBreakdown",
	"insiderTransactions",
	"insiderHolders",
	"netSharePurchaseActivity",
}

// MajorHolders returns the major holders breakdown.
//
// This includes percentages held by insiders, institutions, and institutional count.
//
// Example:
//
//	holders, err := ticker.MajorHolders()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Insiders: %.2f%%\n", holders.InsidersPercentHeld*100)
//	fmt.Printf("Institutions: %.2f%%\n", holders.InstitutionsPercentHeld*100)
func (t *Ticker) MajorHolders() (*models.MajorHolders, error) {
	if err := t.ensureHoldersCache(); err != nil {
		return nil, err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.holdersCache == nil || t.holdersCache.major == nil {
		return nil, fmt.Errorf("major holders data not available")
	}

	return t.holdersCache.major, nil
}

// InstitutionalHolders returns the list of institutional holders.
//
// Each holder includes the institution name, shares held, value, and percentage.
//
// Example:
//
//	holders, err := ticker.InstitutionalHolders()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, h := range holders {
//	    fmt.Printf("%s: %d shares (%.2f%%)\n", h.Holder, h.Shares, h.PctHeld*100)
//	}
func (t *Ticker) InstitutionalHolders() ([]models.Holder, error) {
	if err := t.ensureHoldersCache(); err != nil {
		return nil, err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.holdersCache == nil {
		return nil, nil
	}

	return t.holdersCache.institutional, nil
}

// MutualFundHolders returns the list of mutual fund holders.
//
// Each holder includes the fund name, shares held, value, and percentage.
//
// Example:
//
//	holders, err := ticker.MutualFundHolders()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, h := range holders {
//	    fmt.Printf("%s: %d shares\n", h.Holder, h.Shares)
//	}
func (t *Ticker) MutualFundHolders() ([]models.Holder, error) {
	if err := t.ensureHoldersCache(); err != nil {
		return nil, err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.holdersCache == nil {
		return nil, nil
	}

	return t.holdersCache.mutualFund, nil
}

// InsiderTransactions returns the list of insider transactions.
//
// This includes purchases, sales, and other transactions by company insiders.
//
// Example:
//
//	transactions, err := ticker.InsiderTransactions()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, tx := range transactions {
//	    fmt.Printf("%s: %s %d shares on %s\n",
//	        tx.Insider, tx.Transaction, tx.Shares, tx.StartDate.Format("2006-01-02"))
//	}
func (t *Ticker) InsiderTransactions() ([]models.InsiderTransaction, error) {
	if err := t.ensureHoldersCache(); err != nil {
		return nil, err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.holdersCache == nil {
		return nil, nil
	}

	return t.holdersCache.insiderTransactions, nil
}

// InsiderRoster returns the list of company insiders.
//
// This includes insider names, positions, and their holdings.
//
// Example:
//
//	roster, err := ticker.InsiderRoster()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, insider := range roster {
//	    fmt.Printf("%s (%s): %d shares\n",
//	        insider.Name, insider.Position, insider.TotalShares())
//	}
func (t *Ticker) InsiderRoster() ([]models.InsiderHolder, error) {
	if err := t.ensureHoldersCache(); err != nil {
		return nil, err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.holdersCache == nil {
		return nil, nil
	}

	return t.holdersCache.insiderRoster, nil
}

// InsiderPurchases returns insider purchase activity summary.
//
// This summarizes net share purchases/sales by insiders over a period.
//
// Example:
//
//	purchases, err := ticker.InsiderPurchases()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Net shares: %d (%s)\n", purchases.Net.Shares, purchases.Period)
func (t *Ticker) InsiderPurchases() (*models.InsiderPurchases, error) {
	if err := t.ensureHoldersCache(); err != nil {
		return nil, err
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.holdersCache == nil || t.holdersCache.insiderPurchases == nil {
		return nil, nil
	}

	return t.holdersCache.insiderPurchases, nil
}

// ensureHoldersCache fetches and caches all holders data.
func (t *Ticker) ensureHoldersCache() error {
	t.mu.RLock()
	cached := t.holdersCache != nil
	t.mu.RUnlock()

	if cached {
		return nil
	}

	data, err := t.fetchQuoteSummary(holdersModules)
	if err != nil {
		return fmt.Errorf("failed to fetch holders data: %w", err)
	}

	cache := &holdersCache{}

	// Parse majorHoldersBreakdown
	if breakdown, ok := data["majorHoldersBreakdown"].(map[string]interface{}); ok {
		cache.major = t.parseMajorHolders(breakdown)
	}

	// Parse institutionOwnership
	if instOwnership, ok := data["institutionOwnership"].(map[string]interface{}); ok {
		cache.institutional = t.parseOwnershipList(instOwnership)
	}

	// Parse fundOwnership
	if fundOwnership, ok := data["fundOwnership"].(map[string]interface{}); ok {
		cache.mutualFund = t.parseOwnershipList(fundOwnership)
	}

	// Parse insiderTransactions
	if insiderTx, ok := data["insiderTransactions"].(map[string]interface{}); ok {
		cache.insiderTransactions = t.parseInsiderTransactions(insiderTx)
	}

	// Parse insiderHolders
	if insiderHolders, ok := data["insiderHolders"].(map[string]interface{}); ok {
		cache.insiderRoster = t.parseInsiderHolders(insiderHolders)
	}

	// Parse netSharePurchaseActivity
	if netActivity, ok := data["netSharePurchaseActivity"].(map[string]interface{}); ok {
		cache.insiderPurchases = t.parseNetSharePurchaseActivity(netActivity)
	}

	t.mu.Lock()
	t.holdersCache = cache
	t.mu.Unlock()

	return nil
}

// parseMajorHolders parses majorHoldersBreakdown data.
func (t *Ticker) parseMajorHolders(data map[string]interface{}) *models.MajorHolders {
	return &models.MajorHolders{
		InsidersPercentHeld:          getNestedFloat(data, "insidersPercentHeld"),
		InstitutionsPercentHeld:      getNestedFloat(data, "institutionsPercentHeld"),
		InstitutionsFloatPercentHeld: getNestedFloat(data, "institutionsFloatPercentHeld"),
		InstitutionsCount:            getNestedInt(data, "institutionsCount"),
	}
}

// parseOwnershipList parses institutionOwnership or fundOwnership data.
func (t *Ticker) parseOwnershipList(data map[string]interface{}) []models.Holder {
	ownershipList, ok := data["ownershipList"].([]interface{})
	if !ok {
		return nil
	}

	holders := make([]models.Holder, 0, len(ownershipList))

	for _, item := range ownershipList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Parse reportDate
		var reportDate time.Time
		if dateVal := getNestedFloat(itemMap, "reportDate"); dateVal > 0 {
			reportDate = time.Unix(int64(dateVal), 0)
		}

		holder := models.Holder{
			DateReported: reportDate,
			Holder:       getString(itemMap, "organization"),
			Shares:       int64(getNestedFloat(itemMap, "position")),
			Value:        getNestedFloat(itemMap, "value"),
			PctHeld:      getNestedFloat(itemMap, "pctHeld"),
			PctChange:    getNestedFloat(itemMap, "pctChange"),
		}
		holders = append(holders, holder)
	}

	return holders
}

// parseInsiderTransactions parses insiderTransactions data.
func (t *Ticker) parseInsiderTransactions(data map[string]interface{}) []models.InsiderTransaction {
	transactions, ok := data["transactions"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]models.InsiderTransaction, 0, len(transactions))

	for _, item := range transactions {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Parse startDate
		var startDate time.Time
		if dateVal := getNestedFloat(itemMap, "startDate"); dateVal > 0 {
			startDate = time.Unix(int64(dateVal), 0)
		}

		tx := models.InsiderTransaction{
			StartDate:   startDate,
			Insider:     getString(itemMap, "filerName"),
			Position:    getString(itemMap, "filerRelation"),
			URL:         getString(itemMap, "filerUrl"),
			Transaction: getString(itemMap, "moneyText"),
			Text:        getString(itemMap, "transactionText"),
			Shares:      int64(getNestedFloat(itemMap, "shares")),
			Value:       getNestedFloat(itemMap, "value"),
			Ownership:   getString(itemMap, "ownership"),
		}
		result = append(result, tx)
	}

	return result
}

// parseInsiderHolders parses insiderHolders data.
func (t *Ticker) parseInsiderHolders(data map[string]interface{}) []models.InsiderHolder {
	holders, ok := data["holders"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]models.InsiderHolder, 0, len(holders))

	for _, item := range holders {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		// Parse dates
		var latestTransDate, positionDirectDate *time.Time
		if dateVal := getNestedFloat(itemMap, "latestTransDate"); dateVal > 0 {
			t := time.Unix(int64(dateVal), 0)
			latestTransDate = &t
		}
		if dateVal := getNestedFloat(itemMap, "positionDirectDate"); dateVal > 0 {
			t := time.Unix(int64(dateVal), 0)
			positionDirectDate = &t
		}

		holder := models.InsiderHolder{
			Name:                  getString(itemMap, "name"),
			Position:              getString(itemMap, "relation"),
			URL:                   getString(itemMap, "url"),
			MostRecentTransaction: getString(itemMap, "transactionDescription"),
			LatestTransDate:       latestTransDate,
			PositionDirectDate:    positionDirectDate,
			SharesOwnedDirectly:   int64(getNestedFloat(itemMap, "positionDirect")),
			SharesOwnedIndirectly: int64(getNestedFloat(itemMap, "positionIndirect")),
		}
		result = append(result, holder)
	}

	return result
}

// parseNetSharePurchaseActivity parses netSharePurchaseActivity data.
func (t *Ticker) parseNetSharePurchaseActivity(data map[string]interface{}) *models.InsiderPurchases {
	return &models.InsiderPurchases{
		Period: getString(data, "period"),
		Purchases: models.TransactionStats{
			Shares:       int64(getNestedFloat(data, "buyInfoShares")),
			Transactions: getNestedInt(data, "buyInfoCount"),
		},
		Sales: models.TransactionStats{
			Shares:       int64(getNestedFloat(data, "sellInfoShares")),
			Transactions: getNestedInt(data, "sellInfoCount"),
		},
		Net: models.TransactionStats{
			Shares:       int64(getNestedFloat(data, "netInfoShares")),
			Transactions: getNestedInt(data, "netInfoCount"),
		},
		TotalInsiderShares:       int64(getNestedFloat(data, "totalInsiderShares")),
		NetPercentInsiderShares:  getNestedFloat(data, "netPercentInsiderShares"),
		BuyPercentInsiderShares:  getNestedFloat(data, "buyPercentInsiderShares"),
		SellPercentInsiderShares: getNestedFloat(data, "sellPercentInsiderShares"),
	}
}
