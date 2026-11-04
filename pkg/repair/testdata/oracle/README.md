# Upstream differential corpus for price-shift repair

These JSON files are goldens produced by running the **upstream Python
yfinance 1.8.0** price-repair code on a seeded corpus of inputs.
`pkg/repair/oracle_corpus_test.go` replays every case through the Go port and
requires identical results.

| File | Upstream entry point | Cases |
| --- | --- | --- |
| `split.json` | `_fix_bad_stock_splits` | random walks: forward/reverse splits (2, 3, 10, 1/4, 1/15, ...), whole/partial/old/double adjustment errors, flat volume, genuine crashes, two splits, near-ratio moves, 1d/1wk/1mo/3mo, NaN rows, zero prices, zero volume, wide rows, >1 year of history (start_min) |
| `unit_switch.json` | `_fix_unit_switch` | 100x / 0.01x switches (newest, oldest, blocks, two blocks), per-column errors (`correct_columns_individually`), suspended tails, splits near a switch (the 100x abort), volume moving inversely, KWF (1000x), `currencyRepaired`, 1d/1wk/1mo/3mo/1h |
| `level_shift.json` | `_fix_unexplained_level_shifts` (1h fetch mocked, clock fixed) | shifts of 2-25x and reciprocals, genuine crashes, flat volume, noisy 1h data, splits nearby, non-integer multiples, edge positions, data older than 729 days |
| `detect.json` | `_detect_price_shifts` | frames passed directly (newest first) with and without `change`, per column or denoised, incl. the per-column divergence example (newer O10 H12 L5 C11 vs older O20 H21 L19 C40) |
| `fixtures.json` | as above | every upstream `tests/data` split, 100x and level-shift fixture (4063.T, ALPHA.PA, AV.L 1wk, CNE.L, DEX.AX, LA.V, MOB.ST, NFE, NRDY, SPM.MI, AET.L, SSW.JO, ASAI.L 1h, SOXS, KALA) |
| `mined.json` | as above | adversarial cases kept because a single-detail mutation of upstream's code (see `MUTATIONS` in the generator) changes their result, incl. cases bisected to sit on a threshold |
| `numpy.json` | NumPy | `np.sum`/`mean`/`std`/`percentile`/`median`/`round` results that `numpy.go` must reproduce bit for bit |

Each case stores the input frame (`bars`, column-oriented, Unix-second
timestamps, `null` = NaN), the expected output (`want`: only the columns that
changed, plus the indices of rows flagged `Repaired?`), and a trace of every
`_fix_prices_sudden_change` call (`calls`): detector masks and threshold,
volume threshold, masks after the volume-spike and local-volatility filters,
and the ranges before and after the volume reconcile. Masks are row indices
(newest-first positions), or `[row, column]` pairs for per-column detection;
ranges are `[start, end, 1 if 'split' else 0]`.

## Regenerating

You need a python-yfinance **1.8.0** checkout and an environment with its
dependencies (numpy, pandas, python-dateutil, curl_cffi, ...). The script
refuses to run against another version.

```bash
YFINANCE_UPSTREAM=/path/to/python-yfinance \
  python -I pkg/repair/testdata/oracle/gen_oracle.py
```

The output is deterministic (fixed seeds); it was generated with Python 3.14,
NumPy 2.5 and pandas 3.0. The script writes no bytecode into the upstream
checkout. Regenerating takes about two minutes, most of it mining mutations.

Upstream fixtures are read from `$YFINANCE_UPSTREAM/tests/data` and are
Apache-2.0 licensed (see `../UPSTREAM.md`).
