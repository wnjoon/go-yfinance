package ticker

import (
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestMajorHoldersModel(t *testing.T) {
	holders := models.MajorHolders{
		InsidersPercentHeld:          0.05,
		InstitutionsPercentHeld:      0.65,
		InstitutionsFloatPercentHeld: 0.70,
		InstitutionsCount:            500,
	}

	if holders.InsidersPercentHeld != 0.05 {
		t.Errorf("Expected InsidersPercentHeld 0.05, got %f", holders.InsidersPercentHeld)
	}

	if holders.InstitutionsCount != 500 {
		t.Errorf("Expected InstitutionsCount 500, got %d", holders.InstitutionsCount)
	}
}

func TestHolderModel(t *testing.T) {
	now := time.Now()
	holder := models.Holder{
		DateReported: now,
		Holder:       "Vanguard Group Inc",
		Shares:       100000000,
		Value:        25000000000,
		PctHeld:      0.08,
		PctChange:    0.05,
	}

	if holder.Holder != "Vanguard Group Inc" {
		t.Errorf("Expected holder 'Vanguard Group Inc', got '%s'", holder.Holder)
	}

	if holder.Shares != 100000000 {
		t.Errorf("Expected shares 100000000, got %d", holder.Shares)
	}

	if holder.PctHeld != 0.08 {
		t.Errorf("Expected PctHeld 0.08, got %f", holder.PctHeld)
	}
}

func TestInsiderTransactionModel(t *testing.T) {
	tx := models.InsiderTransaction{
		StartDate:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		Insider:     "Tim Cook",
		Position:    "CEO",
		Transaction: "Sale",
		Shares:      50000,
		Value:       12500000,
		Ownership:   "D",
	}

	if tx.Insider != "Tim Cook" {
		t.Errorf("Expected insider 'Tim Cook', got '%s'", tx.Insider)
	}

	if tx.Transaction != "Sale" {
		t.Errorf("Expected transaction 'Sale', got '%s'", tx.Transaction)
	}

	if tx.Ownership != "D" {
		t.Errorf("Expected ownership 'D', got '%s'", tx.Ownership)
	}
}

func TestInsiderHolderModel(t *testing.T) {
	now := time.Now()
	holder := models.InsiderHolder{
		Name:                  "Tim Cook",
		Position:              "CEO",
		MostRecentTransaction: "Sale",
		LatestTransDate:       &now,
		PositionDirectDate:    &now,
		SharesOwnedDirectly:   100000,
		SharesOwnedIndirectly: 50000,
	}

	if holder.Name != "Tim Cook" {
		t.Errorf("Expected name 'Tim Cook', got '%s'", holder.Name)
	}

	total := holder.TotalShares()
	if total != 150000 {
		t.Errorf("Expected total shares 150000, got %d", total)
	}
}

func TestInsiderHolderTotalSharesZero(t *testing.T) {
	holder := models.InsiderHolder{
		Name:     "New Board Member",
		Position: "Director",
	}

	if holder.TotalShares() != 0 {
		t.Errorf("Expected total shares 0, got %d", holder.TotalShares())
	}
}

func TestInsiderPurchasesModel(t *testing.T) {
	purchases := models.InsiderPurchases{
		Period: "6m",
		Purchases: models.TransactionStats{
			Shares:       100000,
			Transactions: 5,
		},
		Sales: models.TransactionStats{
			Shares:       200000,
			Transactions: 10,
		},
		Net: models.TransactionStats{
			Shares:       -100000,
			Transactions: 15,
		},
		TotalInsiderShares:       5000000,
		NetPercentInsiderShares:  -0.02,
		BuyPercentInsiderShares:  0.02,
		SellPercentInsiderShares: 0.04,
	}

	if purchases.Period != "6m" {
		t.Errorf("Expected period '6m', got '%s'", purchases.Period)
	}

	if purchases.Net.Shares != -100000 {
		t.Errorf("Expected net shares -100000, got %d", purchases.Net.Shares)
	}

	if purchases.Purchases.Transactions != 5 {
		t.Errorf("Expected 5 purchase transactions, got %d", purchases.Purchases.Transactions)
	}
}

func TestHoldersDataModel(t *testing.T) {
	data := models.HoldersData{
		Major: &models.MajorHolders{
			InsidersPercentHeld:     0.05,
			InstitutionsPercentHeld: 0.65,
		},
		Institutional: []models.Holder{
			{Holder: "Vanguard"},
			{Holder: "BlackRock"},
		},
		MutualFund: []models.Holder{
			{Holder: "Fidelity Growth Fund"},
		},
	}

	if data.Major == nil {
		t.Fatal("Expected Major to be non-nil")
	}

	if len(data.Institutional) != 2 {
		t.Errorf("Expected 2 institutional holders, got %d", len(data.Institutional))
	}

	if len(data.MutualFund) != 1 {
		t.Errorf("Expected 1 mutual fund holder, got %d", len(data.MutualFund))
	}
}

func TestHoldersCacheInitialization(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	defer tkr.Close()

	if tkr.holdersCache != nil {
		t.Error("Expected holdersCache to be nil initially")
	}
}

func TestClearCacheIncludesHolders(t *testing.T) {
	tkr, err := New("AAPL")
	if err != nil {
		t.Fatalf("Failed to create ticker: %v", err)
	}
	defer tkr.Close()

	// Manually set holdersCache
	tkr.holdersCache = &holdersCache{
		major: &models.MajorHolders{
			InsidersPercentHeld: 0.05,
		},
	}

	// Clear cache
	tkr.ClearCache()

	if tkr.holdersCache != nil {
		t.Error("Expected holdersCache to be nil after ClearCache")
	}
}

// Integration test - commented out for CI, run manually
// func TestHoldersLive(t *testing.T) {
// 	tkr, err := New("AAPL")
// 	if err != nil {
// 		t.Fatalf("Failed to create ticker: %v", err)
// 	}
// 	defer tkr.Close()
//
// 	// Test Major Holders
// 	major, err := tkr.MajorHolders()
// 	if err != nil {
// 		t.Fatalf("Failed to get major holders: %v", err)
// 	}
//
// 	t.Logf("Major Holders:")
// 	t.Logf("  Insiders: %.2f%%", major.InsidersPercentHeld*100)
// 	t.Logf("  Institutions: %.2f%%", major.InstitutionsPercentHeld*100)
// 	t.Logf("  Institution Count: %d", major.InstitutionsCount)
//
// 	// Test Institutional Holders
// 	inst, err := tkr.InstitutionalHolders()
// 	if err != nil {
// 		t.Fatalf("Failed to get institutional holders: %v", err)
// 	}
//
// 	t.Logf("Top 5 Institutional Holders:")
// 	for i, h := range inst {
// 		if i >= 5 {
// 			break
// 		}
// 		t.Logf("  %s: %d shares (%.2f%%)", h.Holder, h.Shares, h.PctHeld*100)
// 	}
//
// 	// Test Mutual Fund Holders
// 	mf, err := tkr.MutualFundHolders()
// 	if err != nil {
// 		t.Fatalf("Failed to get mutual fund holders: %v", err)
// 	}
//
// 	t.Logf("Top 5 Mutual Fund Holders:")
// 	for i, h := range mf {
// 		if i >= 5 {
// 			break
// 		}
// 		t.Logf("  %s: %d shares", h.Holder, h.Shares)
// 	}
//
// 	// Test Insider Transactions
// 	transactions, err := tkr.InsiderTransactions()
// 	if err != nil {
// 		t.Fatalf("Failed to get insider transactions: %v", err)
// 	}
//
// 	t.Logf("Recent Insider Transactions: %d", len(transactions))
// 	for i, tx := range transactions {
// 		if i >= 3 {
// 			break
// 		}
// 		t.Logf("  %s: %s %d shares on %s",
// 			tx.Insider, tx.Transaction, tx.Shares, tx.StartDate.Format("2006-01-02"))
// 	}
//
// 	// Test Insider Roster
// 	roster, err := tkr.InsiderRoster()
// 	if err != nil {
// 		t.Fatalf("Failed to get insider roster: %v", err)
// 	}
//
// 	t.Logf("Insider Roster: %d insiders", len(roster))
// 	for i, insider := range roster {
// 		if i >= 3 {
// 			break
// 		}
// 		t.Logf("  %s (%s): %d shares", insider.Name, insider.Position, insider.TotalShares())
// 	}
//
// 	// Test Insider Purchases
// 	purchases, err := tkr.InsiderPurchases()
// 	if err != nil {
// 		t.Fatalf("Failed to get insider purchases: %v", err)
// 	}
//
// 	if purchases != nil {
// 		t.Logf("Insider Purchases (%s):", purchases.Period)
// 		t.Logf("  Buys: %d shares (%d transactions)", purchases.Purchases.Shares, purchases.Purchases.Transactions)
// 		t.Logf("  Sells: %d shares (%d transactions)", purchases.Sales.Shares, purchases.Sales.Transactions)
// 		t.Logf("  Net: %d shares", purchases.Net.Shares)
// 	}
// }
