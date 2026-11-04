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

`SOXS-1d-bad-level-shift.csv`, `SOXS-1d-bad-level-shift-fixed.csv`,
`SOXS-1h-bad-level-shift.csv`, `KALA-1d-no-bad-level-shift.csv` and
`KALA-1h-no-bad-level-shift.csv` are copied byte-for-byte from Python
yfinance 1.8.0's `tests/data` directory. They were introduced by upstream
commits `905d82f` and `a3241e6` (ranaroussi/yfinance#2990) and remain
subject to that project's Apache-2.0 license.

SOXS daily prices before 2026-05-26 are 15x too high and volume 15x too low,
with no split near that date; the 1h data is unaffected and measures the 15x.
KALA genuinely fell 92% on 2025-09-29, which the 1h data shows too, so it must
not be repaired. They are exercised by `TestRepairLevelShiftSOXSGolden`,
`TestRepairLevelShiftKALAGenuineCrash` and related tests (with the clock set
one week after the last row, instead of upstream's moving of the fixtures to
recent dates), and by the `pkg/ticker` History wiring tests.

`oracle/fixtures.json` embeds the price data of every upstream 1.8.0
`tests/data` split, 100x-error and level-shift fixture (4063-T, ALPHA-PA,
AV-L, CNE-L, DEX-AX, LA-V, MOB-ST, NFE, NRDY, SPM-MI, AET-L, SSW-JO, ASAI-L,
SOXS, KALA), with the output of the upstream repair functions on it, also
subject to the Apache-2.0 license. See `oracle/README.md`.
