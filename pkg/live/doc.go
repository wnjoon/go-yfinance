// Package live provides real-time WebSocket streaming for Yahoo Finance data.
//
// # Overview
//
// The live package enables real-time price updates via Yahoo Finance's
// WebSocket streaming API. Messages are delivered as protobuf-encoded
// pricing data and decoded into [models.PricingData] structs.
//
// # Basic Usage
//
//	ws, err := live.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer ws.Close()
//
//	// Subscribe to symbols
//	ws.Subscribe([]string{"AAPL", "MSFT", "GOOGL"})
//
//	// Listen for updates
//	ws.Listen(func(data *models.PricingData) {
//	    fmt.Printf("%s: $%.2f (%.2f%%)\n",
//	        data.ID, data.Price, data.ChangePercent)
//	})
//
// # Async Listening
//
// For non-blocking operation, use ListenAsync:
//
//	ws.Subscribe([]string{"AAPL"})
//	ws.ListenAsync(func(data *models.PricingData) {
//	    fmt.Printf("%s: $%.2f\n", data.ID, data.Price)
//	})
//
//	// Do other work...
//	time.Sleep(10 * time.Second)
//	ws.Close()
//
// # Error Handling
//
// Set an error handler to receive connection errors:
//
//	ws, _ := live.New(
//	    live.WithErrorHandler(func(err error) {
//	        log.Printf("WebSocket error: %v", err)
//	    }),
//	)
//
// # Configuration Options
//
//   - [WithURL]: Set custom WebSocket URL
//   - [WithHeartbeatInterval]: Set heartbeat interval (default 15s)
//   - [WithReconnectDelay]: Set reconnection delay (default 3s)
//   - [WithErrorHandler]: Set error callback
//
// # Data Fields
//
// The [models.PricingData] struct contains real-time market data:
//
//   - ID: Ticker symbol
//   - Price: Current price
//   - Change: Price change from previous close
//   - ChangePercent: Percentage change
//   - DayHigh, DayLow: Day's high and low
//   - DayVolume: Trading volume
//   - Bid, Ask: Current bid/ask prices
//   - MarketHours: Market state (pre/regular/post/closed)
//
// # Thread Safety
//
// All WebSocket methods are safe for concurrent use from multiple goroutines.
package live
