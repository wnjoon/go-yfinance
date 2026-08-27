# Upstream test fixtures

`NRDY-1d-bad-stock-split.csv` and
`NRDY-1d-bad-stock-split-fixed.csv` are copied from Python yfinance 1.7.0's
`tests/data` directory. They were introduced by ranaroussi/yfinance#2958 and
its follow-up fix, and remain subject to that project's Apache-2.0 license.

The fixtures intentionally contain alternating ranges of correct and missing
1:15 reverse-split adjustment. Both the complete table and its final 27 rows
are exercised by `TestRepairStockSplitsNRDYGolden`.
