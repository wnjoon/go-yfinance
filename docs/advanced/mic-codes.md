# MIC Code Mapping

The `utils` package provides Market Identifier Code (MIC) to Yahoo Finance ticker suffix mapping, enabling seamless support for international exchanges.

## Overview

ISO 10383 Market Identifier Codes (MIC) are standardized exchange identifiers. Yahoo Finance uses suffix-based ticker formats (e.g., `AAPL.L` for London). This package bridges the two standards.

## Basic Usage

### Format Ticker for Yahoo Finance

```go
import "github.com/wnjoon/go-yfinance/pkg/utils"

// Convert MIC to Yahoo format
ticker := utils.FormatYahooTicker("AAPL", "XLON")
// Result: "AAPL.L"

ticker = utils.FormatYahooTicker("7203", "XTKS")
// Result: "7203.T" (Toyota on Tokyo Stock Exchange)

// US exchanges have no suffix
ticker = utils.FormatYahooTicker("AAPL", "XNYS")
// Result: "AAPL"
```

### Parse Yahoo Ticker

```go
// Extract base ticker and suffix
base, suffix := utils.ParseYahooTicker("AAPL.L")
// Result: base="AAPL", suffix="L"

base, suffix = utils.ParseYahooTicker("AAPL")
// Result: base="AAPL", suffix=""
```

### Lookup Functions

```go
// MIC to Yahoo suffix
suffix := utils.GetYahooSuffix("XLON")
// Result: "L"

// Yahoo suffix to MIC
mic := utils.GetMIC("L")
// Result: "XLON"

// Check if US exchange
isUS := utils.IsUSExchange("XNYS")
// Result: true

isUS = utils.IsUSExchange("XLON")
// Result: false
```

### List All Supported Exchanges

```go
// Get all supported MIC codes
mics := utils.AllMICs()
// Result: ["XNYS", "XNAS", "XLON", "XTKS", ...]

// Get all Yahoo suffixes (excluding US which has no suffix)
suffixes := utils.AllYahooSuffixes()
// Result: ["L", "T", "HK", "AX", ...]
```

## Supported Exchanges

### Americas

| MIC | Yahoo | Exchange |
|-----|-------|----------|
| XNYS | (none) | New York Stock Exchange |
| XNAS | (none) | NASDAQ |
| XCBT | CBT | Chicago Board of Trade |
| XCME | CME | Chicago Mercantile Exchange |
| XNYM | NYM | New York Mercantile Exchange |
| XTSE | TO | Toronto Stock Exchange |
| XTSX | V | TSX Venture Exchange |
| CNSX | CN | Canadian Securities Exchange |
| NEOE | NE | Aequitas NEO Exchange |
| BVMF | SA | B3 (Brasil Bolsa Balcao) |
| XBUE | BA | Buenos Aires Stock Exchange |
| XSGO | SN | Santiago Stock Exchange |
| XMEX | MX | Mexican Stock Exchange |
| XBOG | CL | Colombia Stock Exchange |
| XCAR | CR | Caracas Stock Exchange |

### Europe

| MIC | Yahoo | Exchange |
|-----|-------|----------|
| XLON | L | London Stock Exchange |
| XETR | DE | XETRA (Germany) |
| XFRA | F | Frankfurt Stock Exchange |
| XPAR | PA | Euronext Paris |
| XAMS | AS | Euronext Amsterdam |
| XBRU | BR | Euronext Brussels |
| XLIS | LS | Euronext Lisbon |
| XDUB | IR | Euronext Dublin |
| XSWX | SW | SIX Swiss Exchange |
| XSTO | ST | Nasdaq Stockholm |
| XCSE | CO | Nasdaq Copenhagen |
| XHEL | HE | Nasdaq Helsinki |
| XOSL | OL | Oslo Stock Exchange |
| BMEX | MC | BME Spanish Exchanges |
| MTAA | MI | Borsa Italiana |
| XVIE | VI | Vienna Stock Exchange |
| XWAR | WA | Warsaw Stock Exchange |
| XATH | AT | Athens Stock Exchange |
| XPRA | PR | Prague Stock Exchange |
| XBUD | BD | Budapest Stock Exchange |
| XICE | IC | Nasdaq Iceland |
| XTAL | TL | Nasdaq Tallinn |
| XRIS | RG | Nasdaq Riga |
| XVIL | VS | Nasdaq Vilnius |
| XBSE | RO | Bucharest Stock Exchange |

### Asia-Pacific

| MIC | Yahoo | Exchange |
|-----|-------|----------|
| XTKS | T | Tokyo Stock Exchange |
| XHKG | HK | Hong Kong Stock Exchange |
| XSHG | SS | Shanghai Stock Exchange |
| XSHE | SZ | Shenzhen Stock Exchange |
| XTAI | TW | Taiwan Stock Exchange |
| ROCO | TWO | Taipei Exchange (TPEX) |
| XKRX | KS | Korea Exchange |
| KQKS | KQ | KOSDAQ |
| XASX | AX | Australian Securities Exchange |
| XAUS | XA | National Stock Exchange of Australia |
| XNZE | NZ | New Zealand Stock Exchange |
| XSES | SI | Singapore Exchange |
| XKLS | KL | Bursa Malaysia |
| XIDX | JK | Indonesia Stock Exchange |
| XBKK | BK | Stock Exchange of Thailand |
| XPHS | PS | Philippine Stock Exchange |
| XSTC | VN | Ho Chi Minh Stock Exchange |
| XBOM | BO | Bombay Stock Exchange |
| XNSE | NS | National Stock Exchange of India |

### Middle East & Africa

| MIC | Yahoo | Exchange |
|-----|-------|----------|
| XSAU | SR | Saudi Stock Exchange (Tadawul) |
| XDFM | AE | Dubai Financial Market |
| XQAT | QA | Qatar Stock Exchange |
| XKFE | KW | Kuwait Stock Exchange |
| XTAE | TA | Tel Aviv Stock Exchange |
| XCAI | CA | Egyptian Exchange |
| XJSE | JO | Johannesburg Stock Exchange |

## Use Cases

### Fetching International Data

```go
import (
    "github.com/wnjoon/go-yfinance/pkg/ticker"
    "github.com/wnjoon/go-yfinance/pkg/utils"
)

// When you have the base ticker and exchange MIC
baseTicker := "VOD"
exchangeMIC := "XLON"

// Format for Yahoo Finance
yahooTicker := utils.FormatYahooTicker(baseTicker, exchangeMIC)
// Result: "VOD.L"

// Fetch data
t, _ := ticker.New(yahooTicker)
defer t.Close()
bars, _ := t.History(models.HistoryParams{Period: "1y"})
```

### Building Multi-Exchange Portfolio

```go
// International portfolio
holdings := []struct {
    Ticker   string
    Exchange string
}{
    {"AAPL", "XNAS"},  // Apple on NASDAQ
    {"TSM", "XTAI"},   // TSMC on Taiwan
    {"7203", "XTKS"},  // Toyota on Tokyo
    {"NESN", "XSWX"},  // Nestle on Swiss
}

for _, h := range holdings {
    yahooTicker := utils.FormatYahooTicker(h.Ticker, h.Exchange)
    // Fetch each ticker...
}
```

### Identifying Exchange from Yahoo Ticker

```go
// Parse unknown ticker
ticker := "SAP.DE"
base, suffix := utils.ParseYahooTicker(ticker)

if suffix != "" {
    mic := utils.GetMIC(suffix)
    if mic != "" {
        fmt.Printf("%s is listed on %s\n", base, mic)
        // Output: SAP is listed on XETR
    }
}
```

## Direct Map Access

For bulk operations, access the maps directly:

```go
// MIC to suffix map
for mic, suffix := range utils.MICToYahooSuffix {
    fmt.Printf("%s -> %s\n", mic, suffix)
}

// Suffix to MIC map (auto-generated reverse mapping)
for suffix, mic := range utils.YahooSuffixToMIC {
    fmt.Printf("%s -> %s\n", suffix, mic)
}
```

## Notes

- **US Exchanges**: NYSE, NASDAQ, and other US exchanges have no suffix (empty string)
- **Reverse Mapping**: Some suffixes may map to multiple MICs; the primary exchange is used
- **Class Shares**: Tickers like `BRK.A` (Berkshire Class A) use dot notation - `ParseYahooTicker` handles this correctly by using the last dot as separator

## See Also

- [Price Repair](price-repair.md) - Data quality repair for international exchanges
- [API Reference](../API.md) - Full API documentation
