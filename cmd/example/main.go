// Example usage of go-yfinance
package main

import (
	"fmt"
	"log"

	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/ticker"
)

func main() {
	symbol := "AAPL"

	fmt.Printf("=== go-yfinance Example: %s ===\n\n", symbol)

	// Create a ticker
	t, err := ticker.New(symbol)
	if err != nil {
		log.Fatalf("Failed to create ticker: %v", err)
	}
	defer t.Close()

	// 1. Get current quote
	fmt.Println("1. Current Quote")
	fmt.Println("----------------")
	quote, err := t.Quote()
	if err != nil {
		log.Fatalf("Failed to get quote: %v", err)
	}
	fmt.Printf("Symbol: %s (%s)\n", quote.Symbol, quote.ShortName)
	fmt.Printf("Price: $%.2f\n", quote.RegularMarketPrice)
	fmt.Printf("Change: %+.2f (%+.2f%%)\n", quote.RegularMarketChange, quote.RegularMarketChangePercent)
	fmt.Printf("Day Range: $%.2f - $%.2f\n", quote.RegularMarketDayLow, quote.RegularMarketDayHigh)
	fmt.Printf("52 Week Range: $%.2f - $%.2f\n", quote.FiftyTwoWeekLow, quote.FiftyTwoWeekHigh)
	fmt.Printf("Volume: %d\n", quote.RegularMarketVolume)
	fmt.Printf("Market Cap: $%d\n", quote.MarketCap)
	fmt.Printf("PE Ratio: %.2f\n", quote.TrailingPE)
	fmt.Printf("Market State: %s\n\n", quote.MarketState)

	// 2. Get historical data
	fmt.Println("2. Historical Data (Last 10 Days)")
	fmt.Println("----------------------------------")
	bars, err := t.History(models.HistoryParams{
		Period:     "1mo",
		Interval:   "1d",
		AutoAdjust: true,
	})
	if err != nil {
		log.Fatalf("Failed to get history: %v", err)
	}

	// Show last 10 bars
	start := len(bars) - 10
	if start < 0 {
		start = 0
	}
	fmt.Printf("%-12s %10s %10s %10s %10s %12s\n", "Date", "Open", "High", "Low", "Close", "Volume")
	for _, bar := range bars[start:] {
		fmt.Printf("%-12s %10.2f %10.2f %10.2f %10.2f %12d\n",
			bar.Date.Format("2006-01-02"),
			bar.Open, bar.High, bar.Low, bar.Close, bar.Volume)
	}
	fmt.Println()

	// 3. Get company info
	fmt.Println("3. Company Info")
	fmt.Println("---------------")
	info, err := t.Info()
	if err != nil {
		log.Fatalf("Failed to get info: %v", err)
	}
	fmt.Printf("Name: %s\n", info.LongName)
	fmt.Printf("Sector: %s\n", info.Sector)
	fmt.Printf("Industry: %s\n", info.Industry)
	fmt.Printf("Country: %s\n", info.Country)
	fmt.Printf("Employees: %d\n", info.FullTimeEmployees)
	fmt.Printf("Website: %s\n", info.Website)
	fmt.Printf("\nKey Statistics:\n")
	fmt.Printf("  Market Cap: $%d\n", info.MarketCap)
	fmt.Printf("  Enterprise Value: $%d\n", info.EnterpriseValue)
	fmt.Printf("  Trailing PE: %.2f\n", info.TrailingPE)
	fmt.Printf("  Forward PE: %.2f\n", info.ForwardPE)
	fmt.Printf("  PEG Ratio: %.2f\n", info.PegRatio)
	fmt.Printf("  Price to Book: %.2f\n", info.PriceToBook)
	fmt.Printf("  Revenue: $%d\n", info.TotalRevenue)
	fmt.Printf("  Profit Margins: %.2f%%\n", info.ProfitMargins*100)
	fmt.Printf("  Recommendation: %s (%.2f)\n\n", info.RecommendationKey, info.RecommendationMean)

	// 4. Get dividends
	fmt.Println("4. Recent Dividends")
	fmt.Println("-------------------")
	dividends, err := t.Dividends()
	if err != nil {
		log.Printf("Failed to get dividends: %v", err)
	} else if len(dividends) > 0 {
		// Show last 5 dividends
		start := len(dividends) - 5
		if start < 0 {
			start = 0
		}
		for _, div := range dividends[start:] {
			fmt.Printf("%s: $%.4f\n", div.Date.Format("2006-01-02"), div.Amount)
		}
	} else {
		fmt.Println("No dividend history")
	}
	fmt.Println()

	// 5. Get splits
	fmt.Println("5. Stock Splits")
	fmt.Println("---------------")
	splits, err := t.Splits()
	if err != nil {
		log.Printf("Failed to get splits: %v", err)
	} else if len(splits) > 0 {
		for _, split := range splits {
			fmt.Printf("%s: %s\n", split.Date.Format("2006-01-02"), split.Ratio)
		}
	} else {
		fmt.Println("No split history")
	}
	fmt.Println()

	fmt.Println("=== Example Complete ===")
}
