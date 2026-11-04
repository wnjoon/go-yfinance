# Upstream test fixtures

`NRDY-1d-bad-stock-split.csv` and
`NRDY-1d-bad-stock-split-fixed.csv` are copied from Python yfinance 1.7.0's
`tests/data` directory. They were introduced by ranaroussi/yfinance#2958 and
its follow-up fix, and remain subject to that project's Apache-2.0 license.

The fixtures intentionally contain alternating ranges of correct and missing
1:15 reverse-split adjustment. Both the complete table and its final 27 rows
are exercised by `TestRepairStockSplitsNRDYGolden`.

`NFE-1d-bad-stock-split.csv` and `NFE-1d-bad-stock-split-fixed.csv` are
copied from Python yfinance 1.8.0's `tests/data` directory. They were
introduced by upstream commit `aec8332` (follow-up `5b63983`) and remain
subject to that project's Apache-2.0 license.

The fixtures miss a 1:50 reverse-split adjustment on every row before the
split. The split-day row has no prices (NaN), and volume genuinely spiked for
a few days after the split, so the range is confirmed only by the
>15-standard-deviation volume fallback. They are exercised by
`TestRepairStockSplitsNFEGolden`, which treats NaN as equal to NaN like
upstream's `equal_nan=True` comparison.
