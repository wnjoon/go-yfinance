package multi

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/wnjoon/go-yfinance/pkg/client"
	"github.com/wnjoon/go-yfinance/pkg/models"
	"github.com/wnjoon/go-yfinance/pkg/ticker"
)

// fakeAttr stands in for a Ticker attribute without network. Like the real
// Ticker methods it caches successes per *ticker.Ticker, and it records calls
// and peak concurrency.
type fakeAttr[T any] struct {
	mu     sync.Mutex
	values map[string]T
	errs   map[string]error
	delay  time.Duration
	before func(symbol string)

	cache  map[*ticker.Ticker]T
	calls  []string
	active int
	peak   int
}

func newFakeAttr[T any](values map[string]T, errs map[string]error) *fakeAttr[T] {
	if errs == nil {
		errs = map[string]error{}
	}
	return &fakeAttr[T]{values: values, errs: errs, cache: map[*ticker.Ticker]T{}}
}

func (f *fakeAttr[T]) fetch(symbol string, tkr *ticker.Ticker) (T, error) {
	f.mu.Lock()
	if v, ok := f.cache[tkr]; ok {
		f.mu.Unlock()
		return v, nil
	}
	f.calls = append(f.calls, symbol)
	f.active++
	if f.active > f.peak {
		f.peak = f.active
	}
	before := f.before
	f.mu.Unlock()

	if before != nil {
		before(symbol)
	}
	time.Sleep(f.delay)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.active--
	if err := f.errs[symbol]; err != nil {
		var zero T
		return zero, err
	}
	v := f.values[symbol]
	f.cache[tkr] = v
	return v, nil
}

func (f *fakeAttr[T]) setErr(symbol string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err == nil {
		delete(f.errs, symbol)
		return
	}
	f.errs[symbol] = err
}

func (f *fakeAttr[T]) takeCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	calls := f.calls
	f.calls = nil
	return calls
}

func (f *fakeAttr[T]) peakConcurrency() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.peak
}

func rateLimitErr() error {
	// Mirrors how Ticker.Info wraps the client's 429 error.
	return fmt.Errorf("failed to fetch info: %w", client.WrapRateLimitError())
}

func newTestTickers(t *testing.T, symbols ...string) *Tickers {
	t.Helper()
	tickers, err := NewTickers(symbols)
	if err != nil {
		t.Fatalf("NewTickers: %v", err)
	}
	// Note: Don't call Close() since CycleTLS is lazily initialized
	return tickers
}

func symbolsN(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("T%d", i)
	}
	return out
}

func infoValues(symbols []string) map[string]*models.Info {
	values := make(map[string]*models.Info, len(symbols))
	for _, s := range symbols {
		values[s] = &models.Info{Symbol: s, Currency: "USD"}
	}
	return values
}

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func TestTickersInfoAllSucceed(t *testing.T) {
	tickers := newTestTickers(t, "AAPL", "MC.PA", "MSFT")
	values := infoValues([]string{"AAPL", "MC.PA", "MSFT"})
	values["MC.PA"].Currency = "EUR"
	fake := newFakeAttr(values, nil)
	tickers.infoFetcher = fake.fetch

	result, err := tickers.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if want := []string{"AAPL", "MC.PA", "MSFT"}; !reflect.DeepEqual(result.Symbols, want) {
		t.Errorf("Symbols = %v, want %v (Tickers order)", result.Symbols, want)
	}
	if got := result.Get("MC.PA").Currency; got != "EUR" {
		t.Errorf("MC.PA currency = %q, want EUR", got)
	}
	if result.SuccessCount() != 3 || result.ErrorCount() != 0 {
		t.Errorf("counts = %d/%d, want 3/0", result.SuccessCount(), result.ErrorCount())
	}
}

func TestTickersInfoPartialFailure(t *testing.T) {
	tickers := newTestTickers(t, "AAPL", "BAD", "MSFT")
	boom := errors.New("boom")
	fake := newFakeAttr(infoValues([]string{"AAPL", "MSFT"}), map[string]error{"BAD": boom})
	tickers.infoFetcher = fake.fetch

	result, err := tickers.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if want := []string{"AAPL", "MSFT"}; !reflect.DeepEqual(result.Symbols, want) {
		t.Errorf("Symbols = %v, want %v", result.Symbols, want)
	}
	if !errors.Is(result.Errors["BAD"], boom) {
		t.Errorf("BAD error = %v, want boom", result.Errors["BAD"])
	}
	if _, ok := result.Data["BAD"]; ok {
		t.Error("failed symbol must not appear in Data")
	}
	if result.Get("BAD") != nil {
		t.Error("Get(BAD) should be nil")
	}
	// A non-rate-limit error must not stop the other fetches.
	if got := sorted(fake.takeCalls()); !reflect.DeepEqual(got, []string{"AAPL", "BAD", "MSFT"}) {
		t.Errorf("calls = %v", got)
	}
}

func TestTickersInfoAllFailReturnsNilError(t *testing.T) {
	tickers := newTestTickers(t, "A", "B")
	boom := errors.New("boom")
	fake := newFakeAttr[*models.Info](nil, map[string]error{"A": boom, "B": boom})
	tickers.infoFetcher = fake.fetch

	result, err := tickers.Info()
	if err != nil {
		t.Fatalf("Info returned top-level error %v; per-symbol errors expected (Download convention)", err)
	}
	if result.SuccessCount() != 0 || result.ErrorCount() != 2 {
		t.Errorf("counts = %d/%d, want 0/2", result.SuccessCount(), result.ErrorCount())
	}
	if result.Symbols == nil || result.Data == nil {
		t.Error("Symbols and Data should be non-nil empty values")
	}
}

func TestTickersInfoRateLimitSkipsRestThenRetryFetchesMissing(t *testing.T) {
	symbols := symbolsN(6)
	tickers := newTestTickers(t, symbols...)
	fake := newFakeAttr(infoValues(symbols), map[string]error{"T2": rateLimitErr()})
	tickers.infoFetcher = fake.fetch

	result, err := tickers.Info(WithWorkers(1))
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	// No request after Yahoo's refusal.
	if got := fake.takeCalls(); !reflect.DeepEqual(got, []string{"T0", "T1", "T2"}) {
		t.Errorf("calls = %v, want [T0 T1 T2]", got)
	}
	if want := []string{"T0", "T1"}; !reflect.DeepEqual(result.Symbols, want) {
		t.Errorf("Symbols = %v, want %v", result.Symbols, want)
	}
	if !client.IsRateLimitError(result.Errors["T2"]) {
		t.Errorf("T2 error = %v, want rate limit", result.Errors["T2"])
	}
	if errors.Is(result.Errors["T2"], ErrSkippedRateLimited) {
		t.Error("the rate-limited symbol itself was requested, not skipped")
	}
	for _, s := range []string{"T3", "T4", "T5"} {
		err := result.Errors[s]
		if !errors.Is(err, ErrSkippedRateLimited) {
			t.Errorf("%s error = %v, want ErrSkippedRateLimited", s, err)
		}
		if err != nil && err.Error() != s+": "+ErrSkippedRateLimited.Error() {
			t.Errorf("%s error message = %q", s, err.Error())
		}
	}

	// Yahoo recovers: the retry only fetches what is missing.
	fake.setErr("T2", nil)
	result, err = tickers.Info(WithWorkers(1))
	if err != nil {
		t.Fatalf("Info retry: %v", err)
	}
	if got := fake.takeCalls(); !reflect.DeepEqual(got, []string{"T2", "T3", "T4", "T5"}) {
		t.Errorf("retry calls = %v, want [T2 T3 T4 T5]", got)
	}
	if result.HasErrors() || !reflect.DeepEqual(result.Symbols, symbols) {
		t.Errorf("retry result = %v errors %v", result.Symbols, result.Errors)
	}

	// Everything cached now: no further requests.
	if _, err := tickers.Info(); err != nil {
		t.Fatalf("Info third call: %v", err)
	}
	if got := fake.takeCalls(); len(got) != 0 {
		t.Errorf("third call made requests %v", got)
	}
}

func TestTickersInfoRateLimitKeepsInFlightResults(t *testing.T) {
	symbols := symbolsN(6)
	tickers := newTestTickers(t, symbols...)
	fake := newFakeAttr(infoValues(symbols), map[string]error{"T1": rateLimitErr()})

	t1Called := make(chan struct{})
	fake.before = func(symbol string) {
		switch symbol {
		case "T1":
			close(t1Called)
		case "T0":
			// Still in flight when T1 is rate-limited; give T1's worker
			// time to record the rate limit and drain the queue.
			<-t1Called
			time.Sleep(50 * time.Millisecond)
		}
	}
	tickers.infoFetcher = fake.fetch

	result, err := tickers.Info(WithWorkers(2))
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if got := sorted(fake.takeCalls()); !reflect.DeepEqual(got, []string{"T0", "T1"}) {
		t.Errorf("calls = %v, want T0 and T1 only", got)
	}
	if result.Get("T0") == nil {
		t.Errorf("in-flight T0 result should be kept, errors: %v", result.Errors)
	}
	if !client.IsRateLimitError(result.Errors["T1"]) {
		t.Errorf("T1 error = %v, want rate limit", result.Errors["T1"])
	}
	for _, s := range symbols[2:] {
		if !errors.Is(result.Errors[s], ErrSkippedRateLimited) {
			t.Errorf("%s error = %v, want ErrSkippedRateLimited", s, result.Errors[s])
		}
	}
	if len(result.Data)+len(result.Errors) != len(symbols) {
		t.Errorf("every symbol must be in Data or Errors: %d + %d", len(result.Data), len(result.Errors))
	}
}

func TestTickersInfoUsesOwnTickerInstances(t *testing.T) {
	tickers := newTestTickers(t, "AAPL", "MSFT")
	var mu sync.Mutex
	seen := map[string]*ticker.Ticker{}
	tickers.infoFetcher = func(symbol string, tkr *ticker.Ticker) (*models.Info, error) {
		mu.Lock()
		seen[symbol] = tkr
		mu.Unlock()
		return &models.Info{Symbol: tkr.Symbol()}, nil
	}

	if _, err := tickers.Info(); err != nil {
		t.Fatalf("Info: %v", err)
	}
	for _, s := range []string{"AAPL", "MSFT"} {
		if seen[s] == nil || seen[s] != tickers.Get(s) {
			t.Errorf("%s fetched through %p, want Tickers' own %p", s, seen[s], tickers.Get(s))
		}
	}
}

func TestTickersInfoDefaultWorkersCapped(t *testing.T) {
	symbols := symbolsN(3 * DefaultAttrWorkers)
	tickers := newTestTickers(t, symbols...)
	fake := newFakeAttr(infoValues(symbols), nil)
	fake.delay = 30 * time.Millisecond
	tickers.infoFetcher = fake.fetch

	if _, err := tickers.Info(); err != nil {
		t.Fatalf("Info: %v", err)
	}
	if peak := fake.peakConcurrency(); peak != DefaultAttrWorkers {
		t.Errorf("peak concurrency = %d, want %d", peak, DefaultAttrWorkers)
	}
}

func TestTickersInfoWorkerOption(t *testing.T) {
	tests := []struct {
		name    string
		symbols int
		workers int
		want    int
	}{
		{"two workers", 8, 2, 2},
		{"one worker is serial", 4, 1, 1},
		{"zero means default", 8, 0, DefaultAttrWorkers},
		{"negative means default", 8, -3, DefaultAttrWorkers},
		{"capped by symbol count", 3, 10, 3},
		{"default capped by symbol count", 2, 0, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbols := symbolsN(tt.symbols)
			tickers := newTestTickers(t, symbols...)
			fake := newFakeAttr(infoValues(symbols), nil)
			fake.delay = 30 * time.Millisecond
			tickers.infoFetcher = fake.fetch

			result, err := tickers.Info(WithWorkers(tt.workers))
			if err != nil {
				t.Fatalf("Info: %v", err)
			}
			if result.SuccessCount() != tt.symbols {
				t.Errorf("SuccessCount = %d, want %d", result.SuccessCount(), tt.symbols)
			}
			if peak := fake.peakConcurrency(); peak != tt.want {
				t.Errorf("peak concurrency = %d, want %d", peak, tt.want)
			}
		})
	}
}

func TestTickersInfoOneWorkerKeepsOrder(t *testing.T) {
	symbols := symbolsN(5)
	tickers := newTestTickers(t, symbols...)
	fake := newFakeAttr(infoValues(symbols), nil)
	tickers.infoFetcher = fake.fetch

	if _, err := tickers.Info(WithWorkers(1), nil); err != nil {
		t.Fatalf("Info: %v", err)
	}
	if got := fake.takeCalls(); !reflect.DeepEqual(got, symbols) {
		t.Errorf("calls = %v, want %v", got, symbols)
	}
}

func TestTickersInfoDuplicateSymbolsFetchedOnce(t *testing.T) {
	tickers := newTestTickers(t, "AAPL", "aapl", "MSFT", "AAPL")
	fake := newFakeAttr(infoValues([]string{"AAPL", "MSFT"}), nil)
	tickers.infoFetcher = fake.fetch

	result, err := tickers.Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if want := []string{"AAPL", "MSFT"}; !reflect.DeepEqual(result.Symbols, want) {
		t.Errorf("Symbols = %v, want %v", result.Symbols, want)
	}
	if got := sorted(fake.takeCalls()); !reflect.DeepEqual(got, []string{"AAPL", "MSFT"}) {
		t.Errorf("calls = %v, want one per unique symbol", got)
	}
}

func TestTickersInfoEmpty(t *testing.T) {
	result, err := (&Tickers{}).Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if result.SuccessCount() != 0 || result.ErrorCount() != 0 || result.Data == nil || result.Errors == nil {
		t.Errorf("unexpected empty result: %+v", result)
	}
}

func TestTickersInfoMissingTicker(t *testing.T) {
	tickers := &Tickers{symbols: []string{"GHOST"}, tickers: map[string]*ticker.Ticker{}}
	fake := newFakeAttr(infoValues([]string{"GHOST"}), nil)
	tickers.infoFetcher = fake.fetch

	result, _ := tickers.Info()
	if result.Errors["GHOST"] == nil {
		t.Error("expected error for a symbol without a Ticker")
	}
	if calls := fake.takeCalls(); len(calls) != 0 {
		t.Errorf("fetcher should not be called, got %v", calls)
	}
}

func TestTickersCalendarMapping(t *testing.T) {
	tickers := newTestTickers(t, "AAPL", "MSFT", "SPY", "BAD")
	exDiv := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	avg := 4.72
	values := map[string]*models.Calendar{
		"AAPL": {ExDividendDate: &exDiv},
		"MSFT": {EarningsAverage: &avg, EarningsDate: []time.Time{exDiv}},
		"SPY":  {},
	}
	boom := errors.New("boom")
	fake := newFakeAttr(values, map[string]error{"BAD": boom})
	tickers.calendarFetcher = fake.fetch

	result, err := tickers.Calendar()
	if err != nil {
		t.Fatalf("Calendar: %v", err)
	}
	if want := []string{"AAPL", "MSFT", "SPY"}; !reflect.DeepEqual(result.Symbols, want) {
		t.Errorf("Symbols = %v, want %v", result.Symbols, want)
	}
	if got := result.Get("AAPL").ExDividendDate; got == nil || !got.Equal(exDiv) {
		t.Errorf("AAPL ex-dividend = %v", got)
	}
	if got := result.Get("MSFT"); got.EarningsAverage == nil || *got.EarningsAverage != 4.72 || !got.HasEarnings() {
		t.Errorf("MSFT calendar = %+v", got)
	}
	if got := result.Get("SPY"); got == nil || got.HasEarnings() {
		t.Errorf("SPY empty calendar = %+v", got)
	}
	if !errors.Is(result.Errors["BAD"], boom) || result.Get("BAD") != nil {
		t.Errorf("BAD = %v / %v", result.Get("BAD"), result.Errors["BAD"])
	}
	if result.SuccessCount() != 3 || result.ErrorCount() != 1 || !result.HasErrors() {
		t.Errorf("counts = %d/%d", result.SuccessCount(), result.ErrorCount())
	}
}

func TestTickersCalendarRateLimitRetry(t *testing.T) {
	symbols := symbolsN(4)
	tickers := newTestTickers(t, symbols...)
	values := map[string]*models.Calendar{}
	for _, s := range symbols {
		values[s] = &models.Calendar{}
	}
	fake := newFakeAttr(values, map[string]error{"T1": rateLimitErr()})
	tickers.calendarFetcher = fake.fetch

	result, _ := tickers.Calendar(WithWorkers(1))
	if got := fake.takeCalls(); !reflect.DeepEqual(got, []string{"T0", "T1"}) {
		t.Errorf("calls = %v", got)
	}
	if !errors.Is(result.Errors["T2"], ErrSkippedRateLimited) || !errors.Is(result.Errors["T3"], ErrSkippedRateLimited) {
		t.Errorf("errors = %v", result.Errors)
	}

	fake.setErr("T1", nil)
	result, _ = tickers.Calendar(WithWorkers(1))
	if got := fake.takeCalls(); !reflect.DeepEqual(got, []string{"T1", "T2", "T3"}) {
		t.Errorf("retry calls = %v", got)
	}
	if result.HasErrors() {
		t.Errorf("retry errors = %v", result.Errors)
	}
}

func TestTickersInfoAndCalendarConcurrentCalls(t *testing.T) {
	symbols := symbolsN(8)
	tickers := newTestTickers(t, symbols...)
	infoFake := newFakeAttr(infoValues(symbols), nil)
	calValues := map[string]*models.Calendar{}
	for _, s := range symbols {
		calValues[s] = &models.Calendar{}
	}
	calFake := newFakeAttr(calValues, nil)
	tickers.infoFetcher = infoFake.fetch
	tickers.calendarFetcher = calFake.fetch

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if r, _ := tickers.Info(); r.SuccessCount() != len(symbols) {
				t.Errorf("Info success = %d", r.SuccessCount())
			}
		}()
		go func() {
			defer wg.Done()
			if r, _ := tickers.Calendar(); r.SuccessCount() != len(symbols) {
				t.Errorf("Calendar success = %d", r.SuccessCount())
			}
		}()
	}
	wg.Wait()
}

func TestRunAttrPoolEmpty(t *testing.T) {
	called := false
	values, errs := runAttrPool(nil, 0, func(int) (int, error) {
		called = true
		return 0, nil
	})
	if called || len(values) != 0 || len(errs) != 0 {
		t.Errorf("empty pool: called=%v values=%v errs=%v", called, values, errs)
	}
}

func TestErrSkippedRateLimitedMessage(t *testing.T) {
	if got := ErrSkippedRateLimited.Error(); got != "skipped because Yahoo rate-limited an earlier request" {
		t.Errorf("message = %q", got)
	}
}
