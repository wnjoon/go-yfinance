# go-yfinance v1.6.0

**Python yfinance v1.6.0 Parity Release**

This release follows Python yfinance v1.6.0. Upstream v1.6.0 is dominated by a
price-repair overhaul; it also adds screener and balance-sheet fields and
improves error messages. The Go port carries every upstream change that maps
onto go-yfinance's in-place repair model, plus one Go-side bug fix the port
uncovered.

The repair port was independently cross-verified by two blind agents against
the upstream diff after the initial implementation; 8 findings were accepted
and are folded into the changes below (see the
[v1.6.0 progress document](../development/v1.6.0-progress.md#post-merge-cross-verification-round-parallel-verify)
for the full before/after list).

## Ported Changes

### Price repair

- **Sub-unit currencies are preserved** (upstream #2907 + regression fix):
  repair math for GBp/ZAc/ILA tickers now runs in the main currency
  (GBP/ZAR/ILS) and the result — prices *and* dividends — is unconditionally
  converted back, so `Repair` no longer permanently converts pence/cents/agorot
  quotes to the main currency. Dividends are scaled to the main currency on
  entry only when they look like they're still in the sub-unit (average
  dividend/prevClose ratio > 1), matching upstream's handling of the common
  LSE pattern where a dividend is already reported in GBP even though prices
  are in pence.
- **Volume cross-check for unit switches and splits** (upstream #2908/#2943):
  a candidate unit switch whose boundary volume mirrors the price move is a
  real corporate action and is skipped; stock-split repair now *requires* the
  mirror-image volume jump at the split date, and does not veto when the
  boundary volume simply can't be computed. Data with no volume at all is
  left untouched. The detection threshold moved to upstream's
  `1 + (change - 1 + pct) * 0.6` formula, with the correct interday noise
  multiplier (1wk/1mo/3mo, not 5d) and population (not sample) standard
  deviation on both the price and volume sides. Unit-switch repair no longer
  rescales Volume (a currency switch doesn't change share counts); both
  unit-switch and split repair now rescale Dividends along with prices,
  matching upstream's `correct_dividend=True`.
- **Per-cell 100x repair** (upstream #2908, ASAI.L fixture): bars where only
  some columns are 100x wrong are repaired column by column; the good columns
  stay untouched, and Low/High are recalculated from Open/Close afterwards.
  The upstream `ASAI-L-1h-bad-unit*.csv` fixtures are ported as the repo's
  first `testdata` golden files.
- **Contradictory OHLC values are repaired** (upstream #2908): bars with
  `Close`/`Open` outside the `Low..High` range have the offending pair refilled
  from the remaining consistent values.
- **Dividend repair** (upstream v1.6.0): the pre/post false-positive test
  measures recovery within the ex-div bar (open-to-close), and a dividend that
  is both 100x too small and missing its adjustment is now fixed in one pass,
  deriving the adjustment from the corrected dividend.
- Unit and split repairs are skipped for FX tickers (`=` in the symbol), and
  the duplicated KWF sub-unit divisor is consolidated.
- **Go-side fix:** `expectedSplitChange` had inverted signs, so genuinely
  unadjusted splits were never detected by `repairStockSplits`. Found while
  porting the volume gate; now fixed and covered end to end.

### Data tables

- Equity screener `profitability` fields gain `dividendyield` and
  `dividendpershare.lasttwelvemonths` (upstream #2888).
- Balance-sheet timeseries keys gain `FixedMaturityInvestments`,
  `EquityInvestments`, `NetLoan`, `DeferredAssets` (upstream #2879).

### Error handling

- Chart errors are now a typed `client.ChartAPIError` whose message is
  `$SYM: <yahoo description>` — Yahoo's reason is surfaced directly, with the
  error code preserved as a field (upstream #2903's final shape).
- Lookup API errors include the query string (upstream #2896).
- A regression test locks JSON-null `quoteSummary` results to the not-found
  path (upstream #2906; Go's `encoding/json` already behaved correctly).

## Not Applicable to Go

- Reconstruction-internal changes — DBSCAN ratio pruning, the Adj-Close
  post-block anchor, newest-first group iteration, and the scikit-learn
  dependency — live inside python's `_reconstruct_intervals_batch`;
  go-yfinance repairs in place and has no sub-interval re-fetch.
- Dividend cluster-threshold changes (0.25→0.5, 0.15→0.11, `.TA` 0.74):
  Go classifies each dividend event independently, without clusters.
- The 30m→15m interval-substitution message fix: go-yfinance fetches 30m
  directly and substitutes nothing.
- Read-only numpy arrays, pandas/numpy deprecation silencing, the packaging
  migration to `pyproject.toml`, and CI/ruff changes are Python-runtime only.

The full item-by-item assessment is recorded in the
[v1.6.0 progress document](../development/v1.6.0-progress.md).

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.6.0
```

## Documentation

- [API Reference](https://wnjoon.github.io/go-yfinance/API/)
- [v1.6.0 Progress](https://wnjoon.github.io/go-yfinance/development/v1.6.0-progress/)
