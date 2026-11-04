# go-yfinance v1.8.0

**Python yfinance v1.8.0 Parity Release**

This release ports the Python yfinance `1.7.0...1.8.0` behavior range onto
go-yfinance v1.7.0, and fixes several Go-side issues found while auditing and
verifying it. Every upstream commit has a recorded disposition in the
[v1.8.0 progress document](https://wnjoon.github.io/go-yfinance/development/v1.8.0-progress/).

## Changes

### News via Yahoo's GraphQL API

- Yahoo retired `/xhr/ncp`, so `Ticker.News()` on v1.7.0 no longer returns
  articles. News now uses the same GraphQL request as finance.yahoo.com, with
  cursor paging and the upstream query and ranking variables.
- `PublishTime` is now filled. New optional `NewsArticle` fields: `Summary`,
  `Description`, `CanonicalURL`, `PreviewURL`, `PublisherURL`, `DisplayTime`,
  `IsHosted`, `IsPremium`; `NewsThumbnail` gains original image fields.
- **Behavior change:** an unknown tab returns an error wrapping
  `models.ErrInvalidNewsTab` instead of silently falling back to latest news.
  Tabs are case-insensitive.
- The news cache is keyed by count, tab and locale.
- New `client.Client.PostJSONWithHeaders`.

### Screener

- The screener works against live Yahoo again: predefined screens use
  `/predefined/saved?scrIds=<name>`, and all screener requests use cookie +
  crumb authentication with Python's query parameters.
- When `Screen` switches a predefined screener to the POST endpoint because
  `Offset > 0`, the requested count is sent as `size`, which that endpoint
  honours.
- New `ScreenerParams.Size`. Custom queries (`ScreenWithQuery`) need `Size`
  to control the number of rows; Yahoo ignores `count` there.
- HTTP errors wrap typed client errors (`client.IsNotFoundError`,
  `IsAuthError`, `IsRateLimitError`).
- `Screen` no longer modifies the caller's `*ScreenerParams`.

### EPS methodology

- New `EarningsEstimateWithMethodology`, `EPSTrendWithMethodology`,
  `EPSRevisionsWithMethodology` and `EarningsHistoryWithMethodology` with
  `models.EPSMethodologyGAAP` / `EPSMethodologyNormalized` (case-insensitive;
  an unknown value returns `models.ErrInvalidEPSMethodology`).
- The existing methods are unchanged and use Yahoo's default methodology.
- EPS rows carry a `Methodology` field: the requested value, or Yahoo's
  default for the ticker on the default path.
- `EPSRevision.DownLast7Days` is now populated (Yahoo spells the key
  `downLast7Days`; it was always 0 before).

### Tickers Info and Calendar

- New `Tickers.Info()` and `Tickers.Calendar()` fetch every symbol in parallel
  (default `min(4, n)` workers, `multi.WithWorkers(n)` to change).
- After a rate limit, symbols not yet started are skipped with
  `multi.ErrSkippedRateLimited`; a later call refetches only missing symbols.

### Price repair

- **Split repair** drops ranges whose trading volume does not confirm the
  split, as upstream 1.8.0 does (NFE regression).
- **Level-shift repair:** a split-like price jump with no matching split is
  confirmed against 1h data and repaired (`RepairOptions.FixLevelShifts`, on
  by default). `History` makes at most one extra 1h request, and only when a
  candidate shift persists. A rate-limit error from that request is returned.
- **One shared detector:** split, unit-switch and level-shift repair now use a
  single port of upstream's price-shift detection, including per-column ratios,
  multi-day interval handling, start_min, suspended-stock handling and the 100x
  abort. A Python-generated differential corpus of 472 cases matches 472/472.
- **History pipeline** now matches upstream:
  - repair runs on raw prices and AutoAdjust is applied afterwards;
  - repair always sees splits, dividends and capital gains, even with
    `Actions: false` (the fields are still cleared from the output);
  - null prices are treated as missing during repair (still returned as 0).
  History without `Repair` returns exactly the same bars as v1.7.0.
- Dividend repair no longer double-adjusts low-yield dividends and uses
  upstream's cluster vote for "100x too small" dividends.
- Bars passed directly to `repair.Repairer` without `AdjClose` (0) are treated
  as unadjusted instead of disabling detection.

## Behavior changes to review when upgrading

- `Ticker.News` with an unknown tab returns an error.
- `History(... Repair: true ...)` results can differ from v1.7.0, especially
  with `AutoAdjust`, `Actions: false`, missing prices, or dividends. These
  changes move results to upstream 1.8.0's output.
- Explicit `models.RepairOptions{...}` values leave `FixLevelShifts` false
  unless set; `DefaultRepairOptions()` enables it.
- Unit-switch repair now repairs every switched range, not only the first.

## Adapted or not applicable

- Go switches the predefined screener to the POST endpoint only for
  `Offset > 0` (Python also does so for an explicit offset of 0).
- `Tickers.Info` / `Calendar` report failures in `Errors` instead of empty
  dicts or NaN rows; Python's progress bar and logging internals have no Go
  counterpart.
- `News` keeps Go's default of 10 articles for `count <= 0`.
- pandas-only changes (index `freq`, `inplace`, rounding, merge helpers), live
  row merging, 60m reconstruction, earnings dates, dividend FX conversion and
  Python packaging/CI changes are not applicable; see the progress document.

## Known remaining gaps

Dividend currency conversion, reconstruction from finer intervals, partial
zero-cell repair, null-row drop order without `KeepNA`, and three of
upstream's dividend cluster votes are not implemented. Details are in the
progress document.

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.8.0
```

## Documentation

- [API Reference](https://wnjoon.github.io/go-yfinance/API/)
- [Price Repair](https://wnjoon.github.io/go-yfinance/advanced/price-repair/)
- [v1.8.0 Progress](https://wnjoon.github.io/go-yfinance/development/v1.8.0-progress/)
