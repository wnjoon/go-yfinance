// Package models provides data structures for Yahoo Finance API responses.
package models

import "time"

// MajorHolders represents the breakdown of major shareholders.
//
// This includes percentages held by insiders, institutions, and the total
// count of institutional holders.
//
// Example:
//
//	holders, err := ticker.MajorHolders()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Insiders: %.2f%%\n", holders.InsidersPercentHeld*100)
//	fmt.Printf("Institutions: %.2f%%\n", holders.InstitutionsPercentHeld*100)
type MajorHolders struct {
	// InsidersPercentHeld is the percentage of shares held by insiders (0.0-1.0).
	InsidersPercentHeld float64 `json:"insidersPercentHeld"`

	// InstitutionsPercentHeld is the percentage of shares held by institutions (0.0-1.0).
	InstitutionsPercentHeld float64 `json:"institutionsPercentHeld"`

	// InstitutionsFloatPercentHeld is the percentage of float held by institutions (0.0-1.0).
	InstitutionsFloatPercentHeld float64 `json:"institutionsFloatPercentHeld"`

	// InstitutionsCount is the number of institutional holders.
	InstitutionsCount int `json:"institutionsCount"`
}

// Holder represents an institutional or mutual fund holder.
//
// This structure is used for both institutional holders and mutual fund holders.
type Holder struct {
	// DateReported is when this holding was reported.
	DateReported time.Time `json:"dateReported"`

	// Holder is the name of the holding institution or fund.
	Holder string `json:"holder"`

	// Shares is the number of shares held.
	Shares int64 `json:"shares"`

	// Value is the total value of the holding.
	Value float64 `json:"value"`

	// PctHeld is the percentage of outstanding shares held (0.0-1.0).
	PctHeld float64 `json:"pctHeld"`

	// PctChange is the percentage change in position since last report (0.0-1.0).
	PctChange float64 `json:"pctChange"`
}

// InsiderTransaction represents a single insider transaction.
//
// This includes purchases, sales, and other transactions by company insiders.
type InsiderTransaction struct {
	// StartDate is the transaction date.
	StartDate time.Time `json:"startDate"`

	// Insider is the name of the insider.
	Insider string `json:"insider"`

	// Position is the insider's position/title in the company.
	Position string `json:"position"`

	// URL is a link to more information about the insider.
	URL string `json:"url,omitempty"`

	// Transaction is the type of transaction (e.g., "Sale", "Purchase").
	Transaction string `json:"transaction"`

	// Text is a description of the transaction.
	Text string `json:"text,omitempty"`

	// Shares is the number of shares involved.
	Shares int64 `json:"shares"`

	// Value is the total value of the transaction.
	Value float64 `json:"value"`

	// Ownership indicates ownership type (e.g., "D" for direct, "I" for indirect).
	Ownership string `json:"ownership"`
}

// InsiderHolder represents an insider on the company's roster.
//
// This provides information about company insiders and their holdings.
type InsiderHolder struct {
	// Name is the insider's name.
	Name string `json:"name"`

	// Position is the insider's position/title.
	Position string `json:"position"`

	// URL is a link to more information.
	URL string `json:"url,omitempty"`

	// MostRecentTransaction describes the most recent transaction.
	MostRecentTransaction string `json:"mostRecentTransaction,omitempty"`

	// LatestTransDate is the date of the most recent transaction.
	LatestTransDate *time.Time `json:"latestTransDate,omitempty"`

	// PositionDirectDate is when direct position was last reported.
	PositionDirectDate *time.Time `json:"positionDirectDate,omitempty"`

	// SharesOwnedDirectly is the number of shares owned directly.
	SharesOwnedDirectly int64 `json:"sharesOwnedDirectly"`

	// SharesOwnedIndirectly is the number of shares owned indirectly.
	SharesOwnedIndirectly int64 `json:"sharesOwnedIndirectly"`
}

// TotalShares returns the total shares owned (direct + indirect).
func (h *InsiderHolder) TotalShares() int64 {
	return h.SharesOwnedDirectly + h.SharesOwnedIndirectly
}

// InsiderPurchases represents net share purchase activity by insiders.
//
// This summarizes insider buying and selling activity over a period.
type InsiderPurchases struct {
	// Period is the time period covered (e.g., "6m" for 6 months).
	Period string `json:"period"`

	// Purchases contains purchase statistics.
	Purchases TransactionStats `json:"purchases"`

	// Sales contains sale statistics.
	Sales TransactionStats `json:"sales"`

	// Net contains net purchase/sale statistics.
	Net TransactionStats `json:"net"`

	// TotalInsiderShares is total shares held by insiders.
	TotalInsiderShares int64 `json:"totalInsiderShares"`

	// NetPercentInsiderShares is net shares as percent of insider holdings.
	NetPercentInsiderShares float64 `json:"netPercentInsiderShares"`

	// BuyPercentInsiderShares is buy shares as percent of insider holdings.
	BuyPercentInsiderShares float64 `json:"buyPercentInsiderShares"`

	// SellPercentInsiderShares is sell shares as percent of insider holdings.
	SellPercentInsiderShares float64 `json:"sellPercentInsiderShares"`
}

// TransactionStats represents statistics for a type of transaction.
type TransactionStats struct {
	// Shares is the number of shares involved.
	Shares int64 `json:"shares"`

	// Transactions is the number of transactions.
	Transactions int `json:"transactions"`
}

// HoldersData contains all holder-related data for a ticker.
type HoldersData struct {
	// Major contains the major holders breakdown.
	Major *MajorHolders `json:"major,omitempty"`

	// Institutional contains the list of institutional holders.
	Institutional []Holder `json:"institutional,omitempty"`

	// MutualFund contains the list of mutual fund holders.
	MutualFund []Holder `json:"mutualFund,omitempty"`

	// InsiderTransactions contains the list of insider transactions.
	InsiderTransactions []InsiderTransaction `json:"insiderTransactions,omitempty"`

	// InsiderRoster contains the list of insiders.
	InsiderRoster []InsiderHolder `json:"insiderRoster,omitempty"`

	// InsiderPurchases contains insider purchase activity summary.
	InsiderPurchases *InsiderPurchases `json:"insiderPurchases,omitempty"`
}
