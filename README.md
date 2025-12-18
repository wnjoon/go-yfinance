# go-yfinance

Go implementation of Yahoo Finance API client, inspired by Python's [yfinance](https://github.com/ranaroussi/yfinance).

## Features

- **TLS Fingerprint Spoofing**: Uses CycleTLS to bypass Yahoo's bot detection with Chrome JA3 fingerprint
- **Automatic Authentication**: Cookie/Crumb management with CSRF fallback for EU users
- **Thread-Safe**: Concurrent-safe client and configuration
- **Comprehensive Error Handling**: Typed errors with proper Go error wrapping

## Installation

```bash
go get github.com/wnjoon/go-yfinance
```

## Implementation Status

- [x] **Phase 0: Foundation**
  - [x] CycleTLS-based HTTP client with Chrome TLS fingerprint
  - [x] Cookie/Crumb authentication system
  - [x] CSRF consent flow fallback (for EU users)
  - [x] Comprehensive error handling
  - [x] Configuration management
- [x] **Phase 1: Core Data**
  - [x] Ticker interface with symbol management
  - [x] Quote: real-time quote data
  - [x] History: OHLCV with period/interval support
  - [x] Info: company profile and financial data
  - [x] FastInfo: quick access to common fields
  - [x] Actions: dividends and splits history
- [ ] **Phase 2: Options**
- [ ] **Phase 3: Financials**
- [ ] **Phase 4: Analysis**
- [ ] **Phase 5: Holdings & Actions**
- [ ] **Phase 6: Search & Screener**
- [ ] **Phase 7: Multi-ticker & Batch**
- [ ] **Phase 8: Real-time WebSocket**
- [ ] **Phase 9: Advanced Features**

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/wnjoon/go-yfinance/pkg/models"
    "github.com/wnjoon/go-yfinance/pkg/ticker"
)

func main() {
    // Create a ticker
    t, err := ticker.New("AAPL")
    if err != nil {
        log.Fatal(err)
    }
    defer t.Close()

    // Get current quote
    quote, err := t.Quote()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("AAPL: $%.2f (%+.2f%%)\n",
        quote.RegularMarketPrice,
        quote.RegularMarketChangePercent)

    // Get historical data
    bars, err := t.History(models.HistoryParams{
        Period:   "1mo",
        Interval: "1d",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\nLast 5 days:\n")
    start := len(bars) - 5
    if start < 0 {
        start = 0
    }
    for _, bar := range bars[start:] {
        fmt.Printf("%s: O=%.2f H=%.2f L=%.2f C=%.2f V=%d\n",
            bar.Date.Format("2006-01-02"),
            bar.Open, bar.High, bar.Low, bar.Close, bar.Volume)
    }

    // Get company info
    info, err := t.Info()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\nCompany: %s\n", info.LongName)
    fmt.Printf("Sector: %s\n", info.Sector)
    fmt.Printf("Industry: %s\n", info.Industry)
    fmt.Printf("Market Cap: $%d\n", info.MarketCap)
}
```

## Configuration

```go
import "github.com/wnjoon/go-yfinance/pkg/config"

// Global configuration
cfg := config.Get()
cfg.SetTimeout(60 * time.Second).
    SetMaxRetries(5).
    SetDebug(true)

// Or create custom config
customCfg := config.NewDefault().
    SetProxy("http://proxy:8080").
    EnableCache(10 * time.Minute)
```

## Project Structure

```
go-yfinance/
├── cmd/example/              # Usage examples
├── internal/
│   └── endpoints/            # Yahoo Finance API endpoints
├── pkg/
│   ├── client/               # HTTP client with TLS fingerprint
│   │   ├── client.go         # CycleTLS-based client
│   │   ├── auth.go           # Cookie/Crumb authentication
│   │   └── errors.go         # Error types
│   ├── config/               # Configuration management
│   ├── ticker/               # Ticker data (Phase 1+)
│   └── models/               # Data models (Phase 1+)
├── DESIGN.md                 # Detailed design document
└── README.md
```

## Documentation

See [DESIGN.md](./DESIGN.md) for detailed architecture and implementation roadmap.

## License

MIT License

## Acknowledgments

- [yfinance](https://github.com/ranaroussi/yfinance) - The original Python library
- [CycleTLS](https://github.com/Danny-Dasilva/CycleTLS) - TLS fingerprint spoofing
