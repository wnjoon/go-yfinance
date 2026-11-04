package multi

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/ticker"
)

// DefaultAttrWorkers is the default maximum number of symbols fetched in
// parallel by Tickers.Info and Tickers.Calendar.
//
// Attributes such as Info cost several requests per ticker, so the default is
// lower than a typical history download to stay under Yahoo's rate limit
// (Python yfinance _ATTR_MAX_THREADS).
const DefaultAttrWorkers = 4

// ErrSkippedRateLimited is reported for every symbol that Tickers.Info or
// Tickers.Calendar did not request because Yahoo rate-limited an earlier
// request in the same call. Use errors.Is to detect it.
//
// Values that were fetched successfully stay cached on each Ticker, so calling
// the method again later only requests the skipped and failed symbols.
var ErrSkippedRateLimited = errors.New("skipped because Yahoo rate-limited an earlier request")

// AttrOption configures Tickers.Info and Tickers.Calendar.
type AttrOption func(*attrConfig)

type attrConfig struct {
	workers int
}

// WithWorkers sets how many symbols are fetched in parallel.
//
// Values <= 0 select the default, min(DefaultAttrWorkers, number of symbols).
// Use 1 to fetch sequentially in Tickers order. Fewer workers are gentler on
// Yahoo's rate limit.
func WithWorkers(n int) AttrOption {
	return func(c *attrConfig) {
		c.workers = n
	}
}

// attrFetcher fetches one attribute for one symbol. It is a seam for tests;
// production code calls the corresponding *ticker.Ticker method so values stay
// cached on the Tickers' own Ticker instances.
type attrFetcher[T any] func(symbol string, tkr *ticker.Ticker) (T, error)

func fetchTickerInfo(_ string, tkr *ticker.Ticker) (*models.Info, error) {
	return tkr.Info()
}

func fetchTickerCalendar(_ string, tkr *ticker.Ticker) (*models.Calendar, error) {
	return tkr.Calendar()
}

// Info fetches company information for every symbol in parallel
// (Python yfinance Tickers.info / get_info).
//
// Each symbol is fetched through the Tickers' own Ticker, so successful values
// stay cached and a later call only requests symbols that are still missing.
// At most min(DefaultAttrWorkers, Count()) symbols are fetched at once unless
// WithWorkers is given. Once Yahoo returns a rate-limit error, symbols that
// have not started yet are not requested and report ErrSkippedRateLimited;
// requests already in flight complete and their results are kept.
//
// Per-symbol failures are reported in the result's Errors map; like
// History, the returned error is nil even when every symbol fails.
//
// Example:
//
//	result, err := tickers.Info(multi.WithWorkers(2))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, symbol := range result.Symbols {
//	    fmt.Println(symbol, result.Data[symbol].Currency)
//	}
//	for symbol, err := range result.Errors {
//	    if errors.Is(err, multi.ErrSkippedRateLimited) {
//	        fmt.Println(symbol, "skipped; call Info again later")
//	    }
//	}
func (t *Tickers) Info(opts ...AttrOption) (*models.TickersInfoResult, error) {
	fetch := t.infoFetcher
	if fetch == nil {
		fetch = fetchTickerInfo
	}

	symbols, data, errs := fetchAttr(t, fetch, opts)
	return &models.TickersInfoResult{Data: data, Errors: errs, Symbols: symbols}, nil
}

// Calendar fetches upcoming calendar events for every symbol in parallel
// (Python yfinance Tickers.calendar / get_calendar).
//
// Concurrency, caching, rate-limit and error semantics are the same as Info.
// Where Python returns a NaN row for a failed ticker, Go reports the symbol
// in the result's Errors map instead.
//
// Example:
//
//	result, _ := tickers.Calendar()
//	for _, symbol := range result.Symbols {
//	    cal := result.Data[symbol]
//	    if cal.HasEarnings() {
//	        fmt.Println(symbol, cal.EarningsDate[0].Format("2006-01-02"))
//	    }
//	}
func (t *Tickers) Calendar(opts ...AttrOption) (*models.TickersCalendarResult, error) {
	fetch := t.calendarFetcher
	if fetch == nil {
		fetch = fetchTickerCalendar
	}

	symbols, data, errs := fetchAttr(t, fetch, opts)
	return &models.TickersCalendarResult{Data: data, Errors: errs, Symbols: symbols}, nil
}

// fetchAttr fetches one attribute for every unique symbol of t using a bounded
// worker pool. It returns the successful symbols in Tickers order, the values
// keyed by symbol and the errors keyed by symbol.
func fetchAttr[T any](t *Tickers, fetch attrFetcher[T], opts []AttrOption) ([]string, map[string]T, map[string]error) {
	cfg := attrConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	// Snapshot symbols and their Ticker instances; fetch outside the lock.
	t.mu.RLock()
	symbols := uniqueSymbols(t.symbols)
	tickers := make([]*ticker.Ticker, len(symbols))
	for i, sym := range symbols {
		tickers[i] = t.tickers[sym]
	}
	t.mu.RUnlock()

	values, errs := runAttrPool(symbols, cfg.workers, func(i int) (T, error) {
		if tickers[i] == nil {
			var zero T
			return zero, fmt.Errorf("ticker not found: %s", symbols[i])
		}
		return fetch(symbols[i], tickers[i])
	})

	succeeded := make([]string, 0, len(symbols))
	data := make(map[string]T, len(symbols))
	errMap := make(map[string]error)
	for i, sym := range symbols {
		if errs[i] != nil {
			errMap[sym] = errs[i]
			continue
		}
		data[sym] = values[i]
		succeeded = append(succeeded, sym)
	}
	return succeeded, data, errMap
}

// runAttrPool calls fetch for every index of symbols with at most workers
// concurrent calls (default min(DefaultAttrWorkers, len(symbols))). After a
// rate-limit error, indexes that have not started yet are skipped with
// ErrSkippedRateLimited. Results are stored by index, so they do not depend on
// completion order. All workers have exited when runAttrPool returns.
func runAttrPool[T any](symbols []string, workers int, fetch func(i int) (T, error)) ([]T, []error) {
	n := len(symbols)
	values := make([]T, n)
	errs := make([]error, n)
	if n == 0 {
		return values, errs
	}

	if workers <= 0 {
		workers = DefaultAttrWorkers
	}
	if workers > n {
		workers = n
	}

	jobs := make(chan int, n)
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)

	var rateLimited atomic.Bool
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := range jobs {
				if rateLimited.Load() {
					// More requests would only extend Yahoo's block.
					errs[i] = fmt.Errorf("%s: %w", symbols[i], ErrSkippedRateLimited)
					continue
				}
				v, err := fetch(i)
				if err != nil {
					if client.IsRateLimitError(err) {
						rateLimited.Store(true)
					}
					errs[i] = err
					continue
				}
				values[i] = v
			}
		}()
	}
	wg.Wait()

	return values, errs
}

// uniqueSymbols returns symbols without duplicates, keeping first occurrence
// order. Tickers keeps one Ticker per symbol, so duplicates are fetched once.
func uniqueSymbols(symbols []string) []string {
	seen := make(map[string]struct{}, len(symbols))
	out := make([]string, 0, len(symbols))
	for _, sym := range symbols {
		if _, ok := seen[sym]; ok {
			continue
		}
		seen[sym] = struct{}{}
		out = append(out, sym)
	}
	return out
}
