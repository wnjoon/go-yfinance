# go-yfinance

Go implementation of Yahoo Finance API client, inspired by Python's [yfinance](https://github.com/ranaroussi/yfinance).

## Status

**Work in Progress** - See [DESIGN.md](./DESIGN.md) for implementation roadmap.

## Features (Planned)

- [x] Project Setup
- [ ] Phase 0: Foundation (HTTP Client, Auth, TLS Fingerprint)
- [ ] Phase 1: Core Data (Quote, History, Info)
- [ ] Phase 2: Options
- [ ] Phase 3: Financials
- [ ] Phase 4: Analysis
- [ ] Phase 5: Holdings & Actions
- [ ] Phase 6: Search & Screener
- [ ] Phase 7: Multi-ticker & Batch
- [ ] Phase 8: Real-time WebSocket
- [ ] Phase 9: Advanced Features

## Installation

```bash
go get github.com/your-username/go-yfinance
```

## Quick Start (Planned API)

```go
package main

import (
    "fmt"
    yf "github.com/your-username/go-yfinance/pkg/ticker"
)

func main() {
    // Create a ticker
    ticker := yf.NewTicker("AAPL")

    // Get current quote
    quote, err := ticker.Quote()
    if err != nil {
        panic(err)
    }
    fmt.Printf("AAPL: $%.2f\n", quote.Price)

    // Get historical data
    history, err := ticker.History(yf.HistoryParams{
        Period:   "1mo",
        Interval: "1d",
    })
    if err != nil {
        panic(err)
    }
    for date, bar := range history {
        fmt.Printf("%s: O=%.2f H=%.2f L=%.2f C=%.2f V=%d\n",
            date, bar.Open, bar.High, bar.Low, bar.Close, bar.Volume)
    }

    // Get company info
    info, err := ticker.Info()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Company: %s\n", info.LongName)
    fmt.Printf("Market Cap: %d\n", info.MarketCap)
}
```

## Documentation

See [DESIGN.md](./DESIGN.md) for detailed architecture and implementation plan.

## License

MIT License
