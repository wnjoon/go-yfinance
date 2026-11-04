"""Generate the price-shift differential corpus from upstream Python yfinance.

Runs the upstream 1.8.0 price-repair functions on a seeded corpus of
synthetic and fixture inputs and writes their outputs as JSON goldens that
pkg/repair/oracle_corpus_test.go replays against the Go port.

usage:
    YFINANCE_UPSTREAM=/path/to/python-yfinance python -I gen_oracle.py [outdir]

See README.md in this directory.
"""
import datetime as _dt
import json
import math
import os
import sys
from unittest import mock

UPSTREAM = os.environ.get('YFINANCE_UPSTREAM')
if not UPSTREAM:
    sys.exit('set YFINANCE_UPSTREAM to a python-yfinance 1.8.0 checkout')
sys.dont_write_bytecode = True
sys.path.insert(0, UPSTREAM)

import numpy as np  # noqa: E402
import pandas as pd  # noqa: E402
from yfinance.scrapers import history as yfh  # noqa: E402
from yfinance.version import version as yf_version  # noqa: E402

if yf_version != '1.8.0':
    sys.exit(f'expected yfinance 1.8.0, found {yf_version}')

OUT = sys.argv[1] if len(sys.argv) > 1 else os.path.dirname(os.path.abspath(__file__))
FIXTURES = os.path.join(UPSTREAM, 'tests', 'data')
OHLC = ['Open', 'High', 'Low', 'Close']
COLS = {'o': 'Open', 'h': 'High', 'l': 'Low', 'c': 'Close', 'a': 'Adj Close',
        'v': 'Volume', 'd': 'Dividends', 's': 'Stock Splits'}
NOW = pd.Timestamp('2026-06-30 20:00', tz='UTC')


# ---------------------------------------------------------------- tracing ---

class Tracer:
    """Records every _fix_prices_sudden_change call and its stages."""

    def __init__(self, ph):
        self.calls = []
        self.ph = ph
        self._wrap('_fix_prices_sudden_change', self._fix)
        self._wrap('_detect_price_shifts', self._detect)
        self._wrap('_estimate_volume_shift_threshold', self._estimate)
        self._wrap('_filter_price_shifts_on_volume_spikes', self._vol)
        self._wrap('_filter_price_shifts_that_match_local_stdev', self._local)
        self._wrap('_reconcile_price_shifts_with_volume', self._reconcile)
        self.threshold = None

    def _wrap(self, name, fn):
        orig = getattr(self.ph, name)
        setattr(self.ph, name, lambda *a, **k: fn(orig, *a, **k))

    def _fix(self, orig, df, interval, tz, change, unit_switch=False, correct_volume=False, correct_dividend=False):
        self.calls.append({'change': float(change)})
        return orig(df, interval, tz, change, unit_switch=unit_switch,
                    correct_volume=correct_volume, correct_dividend=correct_dividend)

    def _detect(self, orig, df, interval, change=None, correct_columns_individually=False, *, min_change=2.0):
        code = yfh.PriceHistory._detect_price_shifts.__code__
        box = {}

        def prof(frame, event, arg):
            if event == 'return' and frame.f_code is code:
                box['threshold'] = frame.f_locals.get('threshold')

        old = sys.getprofile()
        sys.setprofile(prof)
        try:
            det = orig(df, interval, change, correct_columns_individually, min_change=min_change)
        finally:
            sys.setprofile(old)
        thr = box.get('threshold')
        self.threshold = None if thr is None else float(thr)
        if self.calls:
            c = self.calls[-1]
            c['rows'] = len(df)
            c['perColumn'] = bool(correct_columns_individually)
            c['threshold'] = self.threshold
            c['detected'] = masks(det)
        return det

    def _estimate(self, orig, df, interval, change, detection):
        thr, ratios = orig(df, interval, change, detection)
        if self.calls:
            self.calls[-1]['volThreshold'] = None if thr is None else float(thr)
        return thr, ratios

    def _vol(self, orig, df, interval, detection):
        det = orig(df, interval, detection)
        if self.calls:
            self.calls[-1]['afterVolume'] = masks(det)
        return det

    def _local(self, orig, df, interval, change, detection):
        det = orig(df, interval, change, detection)
        if self.calls:
            self.calls[-1]['afterLocal'] = masks(det)
        return det

    def _reconcile(self, orig, vol, ranges, f_up, f_down, thr, unit_switch=False):
        out = orig(vol, ranges, f_up, f_down, thr, unit_switch=unit_switch)
        if self.calls:
            self.calls[-1]['rangesIn'] = enc_ranges(ranges)
            self.calls[-1]['rangesOut'] = enc_ranges(out)
        return out


def masks(det):
    if det is None:
        return None
    return {'up': mask_idx(det.up), 'down': mask_idx(det.down)}


def mask_idx(m):
    m = np.asarray(m)
    if m.ndim == 1:
        return [int(i) for i in np.flatnonzero(m)]
    return [[int(i), int(j)] for i, j in zip(*np.nonzero(m))]


def enc_ranges(ranges):
    return [[int(r[0]), int(r[1]), 1 if r[2] == 'split' else 0] for r in ranges]


def new_ph(tz):
    ph = yfh.PriceHistory(None, 'TEST', tz, session=object())
    ph._history_metadata = {}
    return ph


# ------------------------------------------------------------- encoding ---

def fnum(x):
    x = float(x)
    if math.isnan(x):
        return None
    return x


def enc_frame(df):
    out = {'t': [int(ts.timestamp()) for ts in df.index]}
    for k, c in COLS.items():
        if c not in df.columns:
            continue
        if k == 'v':
            out[k] = [int(v) for v in df[c].to_numpy()]
        else:
            out[k] = [fnum(v) for v in df[c].to_numpy()]
    if 'Repaired?' in df.columns:
        rep = [i for i, v in enumerate(df['Repaired?'].to_numpy()) if v is True or v == True]  # noqa: E712
        if rep:
            out['rep'] = rep
    return out


def same(a, b):
    if a is None or b is None:
        return a is None and b is None
    return a == b


def enc_want(df_in, df_out):
    """Columns that changed (full column), plus Repaired? indices."""
    assert len(df_in) == len(df_out), 'row count changed'
    assert (df_in.index == df_out.index).all(), 'row order changed'
    fin, fout = enc_frame(df_in), enc_frame(df_out)
    want = {}
    for k in COLS:
        if k in fout and not all(same(x, y) for x, y in zip(fin[k], fout[k])):
            want[k] = fout[k]
    want['rep'] = fout.get('rep', [])
    return want


def frame(t, o, h, l, c, a, v, d=None, s=None, tz='America/New_York'):
    n = len(t)
    df = pd.DataFrame({
        'Open': np.asarray(o, dtype=float), 'High': np.asarray(h, dtype=float),
        'Low': np.asarray(l, dtype=float), 'Close': np.asarray(c, dtype=float),
        'Adj Close': np.asarray(a, dtype=float),
        'Volume': np.asarray(v, dtype='int64'),
        'Dividends': np.zeros(n) if d is None else np.asarray(d, dtype=float),
        'Stock Splits': np.zeros(n) if s is None else np.asarray(s, dtype=float),
    }, index=pd.to_datetime(np.asarray(t, dtype='int64'), unit='s', utc=True).tz_convert(tz))
    df.index.name = 'Date'
    return df


# ------------------------------------------------------------ synthetic ---

def sig(x, digits=6):
    x = np.asarray(x, dtype=float)
    out = np.empty_like(x)
    for i, v in enumerate(x):
        out[i] = float(f'{v:.{digits}g}') if np.isfinite(v) else v
    return out


def dates(n, interval, tz, end='2024-06-28'):
    if interval == '1d':
        idx = pd.bdate_range(end=end, periods=n, tz=tz)
    elif interval == '1wk':
        idx = pd.date_range(end=end, periods=n, freq='W-MON', tz=tz)
    elif interval == '1mo':
        idx = pd.date_range(end=end, periods=n, freq='MS', tz=tz)
    elif interval == '3mo':
        idx = pd.date_range(end=end, periods=n, freq='QS', tz=tz)
    elif interval == '1h':
        days = pd.bdate_range(end=end, periods=n // 7 + 2, tz=tz)
        idx = pd.DatetimeIndex([d + pd.Timedelta(hours=h) for d in days for h in range(10, 17)])[-n:]
    else:
        raise ValueError(interval)
    return [int(ts.timestamp()) for ts in idx]


def walk(rng, n, interval='1d', sd=None):
    if sd is None:
        sd = {'1d': 0.02, '1h': 0.006, '1wk': 0.045, '1mo': 0.09, '3mo': 0.15}[interval]
        sd *= float(rng.choice([0.3, 1.0, 1.0, 2.0]))
    p0 = float(np.exp(rng.uniform(np.log(0.5), np.log(800))))
    close = p0 * np.exp(np.cumsum(rng.normal(0, sd, n)))
    prev = np.concatenate([[close[0]], close[:-1]])
    open_ = prev * np.exp(rng.normal(0, sd / 3, n))
    high = np.maximum(open_, close) * np.exp(np.abs(rng.normal(0, sd / 2, n)))
    low = np.minimum(open_, close) * np.exp(-np.abs(rng.normal(0, sd / 2, n)))
    v0 = float(np.exp(rng.uniform(np.log(1e4), np.log(5e7))))
    vol = np.round(v0 * np.exp(rng.normal(0, float(rng.choice([0.2, 0.4, 0.8])), n)))
    o, h, l, c = (sig(x) for x in (open_, high, low, close))
    return {'o': o, 'h': h, 'l': l, 'c': c, 'a': c.copy(), 'v': vol.astype('int64'),
            'd': np.zeros(n), 's': np.zeros(n)}


def add_dividends(rng, w, k=None):
    n = len(w['c'])
    k = int(rng.integers(1, 4)) if k is None else k
    a = w['c'].copy()
    for i in sorted(rng.choice(np.arange(1, n), size=min(k, n - 1), replace=False)):
        pct = float(rng.choice([0.005, 0.01, 0.03, 0.3, 0.55]))
        w['d'][i] = float(sig([w['c'][i - 1] * pct], 4)[0])
        a[:i] *= 1 - w['d'][i] / w['c'][i - 1]
    w['a'] = sig(a, 10)


def scale_rows(w, rows, m, vol=True, cols='ohlca'):
    for k in cols:
        w[k][rows] = w[k][rows] * m
    if vol:
        w['v'][rows] = np.round(w['v'][rows] / m).astype('int64')


def overlay(rng, w, n, protect=()):
    """Randomly add NaN rows, zero prices, zero volumes and wide rows."""
    tags = []
    free = [i for i in range(1, n - 1) if i not in protect]
    if rng.random() < 0.25 and free:
        for i in rng.choice(free, size=min(len(free), int(rng.integers(1, 3))), replace=False):
            if rng.random() < 0.5:
                for k in 'ohlca':
                    w[k][i] = np.nan
            else:
                w[str(rng.choice(list('ohlc')))][i] = np.nan
        tags.append('nan')
    if rng.random() < 0.15 and free:
        i = int(rng.choice(free))
        for k in rng.choice(list('ohlc'), size=int(rng.integers(1, 5)), replace=False):
            w[k][i] = 0.0
        tags.append('zero')
    if rng.random() < 0.15 and free:
        for i in rng.choice(free, size=min(len(free), int(rng.integers(1, 4))), replace=False):
            w['v'][i] = 0
        tags.append('zerovol')
    if rng.random() < 0.15 and free:
        i = int(rng.choice(free))
        w['h'][i] *= float(rng.choice([3.0, 8.0]))
        w['l'][i] /= float(rng.choice([2.0, 4.0]))
        tags.append('wide')
    return tags


SPLITS = [2.0, 3.0, 10.0, 0.25, 1 / 15, 5.0, 0.1, 4.0, 1.5, 20.0, 0.5, 1 / 3]


def gen_split_cases(rng, count):
    cases = []
    for k in range(count):
        interval = str(rng.choice(['1d'] * 6 + ['1wk', '1wk', '1mo', '3mo']))
        n = int(rng.integers(20, 80)) if interval in ('1d', '1wk') else int(rng.integers(16, 50))
        if rng.random() < 0.08:
            n = int(rng.integers(260, 330)) if interval == '1d' else n  # start_min territory
        tz = str(rng.choice(['America/New_York', 'Europe/London', 'Asia/Tokyo']))
        w = walk(rng, n, interval)
        if rng.random() < 0.4:
            add_dividends(rng, w)
        split = float(rng.choice(SPLITS))
        si = int(rng.integers(3, n - 1))
        w['s'][si] = split
        variant = str(rng.choice(['bad_all', 'bad_all', 'bad_block', 'bad_block', 'good', 'double',
                                  'flatvol', 'crash', 'bad_old', 'two_splits', 'near']))
        tags = [variant, interval]
        if variant == 'bad_all':
            scale_rows(w, slice(0, si), split)
        elif variant == 'bad_block':
            a = int(rng.integers(0, si))
            b = int(rng.integers(a + 1, si + 1))
            scale_rows(w, slice(a, b), split)
        elif variant == 'double':
            scale_rows(w, slice(0, si), 1 / split)
        elif variant == 'flatvol':
            scale_rows(w, slice(0, si), split, vol=False)
        elif variant == 'crash':
            j = int(rng.integers(1, n))
            scale_rows(w, slice(0, j), split, vol=False)
            w['v'][j:min(n, j + 3)] *= 8
        elif variant == 'bad_old':
            b = max(1, si - int(rng.integers(n // 2, n)))
            scale_rows(w, slice(0, b), split)
        elif variant == 'two_splits':
            s2 = float(rng.choice(SPLITS))
            j = int(rng.integers(2, si)) if si > 2 else 1
            w['s'][j] = s2
            scale_rows(w, slice(0, j), s2)
            scale_rows(w, slice(0, si), split)
        elif variant == 'near':
            m = split * float(rng.uniform(0.75, 1.3)) if split > 1 else split / float(rng.uniform(0.75, 1.3))
            scale_rows(w, slice(0, si), m)
        tags += overlay(rng, w, n, protect=(si,))
        t = dates(n, interval, tz)
        cases.append(('split', f'split-{k:03d}-' + '-'.join(tags), interval, tz, t, w, {}))
    return cases


def gen_unit_cases(rng, count):
    cases = []
    for k in range(count):
        interval = str(rng.choice(['1d'] * 6 + ['1wk', '1mo', '3mo', '1h', '1h']))
        n = int(rng.integers(20, 80)) if interval in ('1d', '1wk', '1h') else int(rng.integers(16, 50))
        tz = str(rng.choice(['Europe/London', 'Africa/Johannesburg', 'Asia/Kuwait', 'America/New_York']))
        currency = {'Europe/London': 'GBP', 'Africa/Johannesburg': 'ZAR',
                    'Asia/Kuwait': str(rng.choice(['KWF', 'KWD'])), 'America/New_York': 'USD'}[tz]
        mult = 1000.0 if currency == 'KWF' else 100.0
        currency_repaired = bool(rng.random() < 0.3)
        w = walk(rng, n, interval)
        if rng.random() < 0.3:
            add_dividends(rng, w, k=1)
        variant = str(rng.choice(['newest_x100', 'newest_x001', 'oldest_x100', 'block', 'block',
                                  'percol', 'percol', 'suspended', 'split_near', 'volinverse',
                                  'none', 'twoblocks']))
        tags = [variant, interval, currency]
        a = int(rng.integers(1, n - 1))
        if variant == 'newest_x100':
            scale_rows(w, slice(a, n), mult, vol=False)
        elif variant == 'newest_x001':
            scale_rows(w, slice(a, n), 1 / mult, vol=False)
        elif variant == 'oldest_x100':
            scale_rows(w, slice(0, a), mult, vol=False)
        elif variant == 'block':
            b = int(rng.integers(a + 1, n + 1))
            scale_rows(w, slice(a, b), float(rng.choice([mult, 1 / mult])), vol=False)
        elif variant == 'twoblocks':
            b = int(rng.integers(a + 1, n + 1))
            scale_rows(w, slice(a, b), mult, vol=False)
            c = int(rng.integers(0, a))
            scale_rows(w, slice(0, c + 1), 1 / mult, vol=False)
        elif variant == 'percol':
            rows = rng.choice(np.arange(a, n), size=min(n - a, int(rng.integers(1, 6))), replace=False)
            cols = ''.join(rng.choice(list('ohlc'), size=int(rng.integers(1, 4)), replace=False))
            if 'c' in cols:
                cols += 'a'
            scale_rows(w, rows, float(rng.choice([mult, 1 / mult])), vol=False, cols=cols)
            if rng.random() < 0.5:
                scale_rows(w, slice(a, n), mult, vol=False)
        elif variant == 'suspended':
            m = int(rng.integers(2, 8))
            for kk in 'ohlca':
                w[kk][n - m:] = w['c'][n - m - 1]
            w['v'][n - m:] = 0
            if rng.random() < 0.7:
                scale_rows(w, slice(n - m, n), mult, vol=False)
            if rng.random() < 0.5:
                scale_rows(w, slice(0, a), mult, vol=False)
        elif variant == 'split_near':
            j = min(n - 1, a + int(rng.integers(1, 12)))
            w['s'][a] = float(rng.choice([2.0, 10.0, 0.1]))
            scale_rows(w, slice(j, n), mult, vol=False)
        elif variant == 'volinverse':
            scale_rows(w, slice(0, a), mult, vol=True)
        tags += overlay(rng, w, n)
        t = dates(n, interval, tz)
        cases.append(('unit_switch', f'unit-{k:03d}-' + '-'.join(tags), interval, tz, t, w,
                      {'currency': currency, 'currencyRepaired': currency_repaired}))
    return cases


def hourly_for(rng, t_daily, w, rows, tz, noise):
    """Two 1h bars per daily row whose aggregate is the TRUE (unshifted) price."""
    th, o, h, l, c, v = [], [], [], [], [], []
    for i in rows:
        if not np.isfinite(w['true_c'][i]):
            continue
        day = pd.Timestamp(t_daily[i], unit='s', tz='UTC').tz_convert(tz)
        fx = np.exp(rng.normal(0, noise, 4))
        to, thh, tl, tc = (w['true_' + k][i] * f for k, f in zip('ohlc', fx))
        mid = (to + tc) / 2
        th += [int((day + pd.Timedelta(hours=10)).timestamp()), int((day + pd.Timedelta(hours=14)).timestamp())]
        o += [to, mid]
        h += [max(to, mid, thh), max(mid, tc, thh)]
        l += [min(to, mid, tl), min(mid, tc, tl)]
        c += [mid, tc]
        v += [1000, 1000]
    return {'t': th, 'o': list(sig(o)), 'h': list(sig(h)), 'l': list(sig(l)), 'c': list(sig(c)),
            'a': list(sig(c)), 'v': v}


def gen_level_cases(rng, count):
    cases = []
    for k in range(count):
        tz = str(rng.choice(['America/New_York', 'Europe/London']))
        n = int(rng.integers(20, 70))
        w = walk(rng, n, '1d')
        if rng.random() < 0.3:
            add_dividends(rng, w, k=1)
        for kk in 'ohlc':
            w['true_' + kk] = w[kk].copy()
        variant = str(rng.choice(['shift', 'shift', 'shift_rcp', 'crash', 'flatvol', 'noisy',
                                  'near_split', 'nearly_int', 'edge', 'old']))
        tags = [variant]
        si = int(rng.integers(2, n - 2))
        mult = float(rng.choice([2, 3, 4, 5, 8, 10, 15, 20, 25]))
        if variant == 'shift_rcp':
            mult = 1 / mult
        elif variant == 'nearly_int':
            mult *= float(rng.uniform(0.95, 1.05))
        elif variant == 'edge':
            si = int(rng.choice([1, 2, 3, n - 3, n - 2]))
        if variant == 'crash':
            scale_rows(w, slice(0, si), mult, vol=False)
            for kk in 'ohlc':
                w['true_' + kk] = w[kk].copy()
        else:
            scale_rows(w, slice(0, si), mult, vol=variant != 'flatvol')
        if variant == 'near_split':
            j = min(n - 1, max(1, si + int(rng.integers(-6, 7))))
            w['s'][j] = float(rng.choice([2.0, 0.1]))
        t_end = '2026-06-26' if variant != 'old' else '2023-06-30'
        t = dates(n, '1d', tz, end=t_end)
        tags += overlay(rng, w, n, protect=(si,))
        noise = 0.04 if variant == 'noisy' else 0.002
        cover = [i for i in range(n) if rng.random() < 0.9]
        hourly = hourly_for(rng, t, w, cover, tz, noise)
        for kk in 'ohlc':
            del w['true_' + kk]
        cases.append(('level_shift', f'level-{k:03d}-' + '-'.join(tags), '1d', tz, t, w,
                      {'now': int(NOW.timestamp()), 'hourly': hourly}))
    return cases


def gen_detect_cases(rng, count):
    cases = []
    # Per-column divergence: ratio of medians 1.95 vs median of ratios 2.82.
    base = walk(np.random.default_rng(7), 30, '1d', sd=0.005)
    for k in 'ohlca':
        base[k] = base[k] / base['c'] * 20.0
    w = {k: v.copy() for k, v in base.items()}
    w['o'][15], w['h'][15], w['l'][15], w['c'][15], w['a'][15] = 20, 21, 19, 40, 40
    w['o'][16:], w['h'][16:], w['l'][16:], w['c'][16:], w['a'][16:] = 10, 12, 5, 11, 11
    for change in (2.0, 3.0, None):
        cases.append(('detect', f'detect-divergent-{change}', '1d', 'America/New_York',
                      dates(30, '1d', 'America/New_York'), {k: v.copy() for k, v in w.items()},
                      {'change': change, 'perColumn': False, 'minChange': 2.0}))
    for k in range(count):
        interval = str(rng.choice(['1d', '1d', '1wk', '1mo', '3mo', '1h']))
        n = int(rng.integers(12, 80))
        w = walk(rng, n, interval)
        if rng.random() < 0.4:
            add_dividends(rng, w)
        change = rng.choice([None, 2.0, 3.0, 10.0, 100.0, 0.25, 1000.0, 1 / 15])
        change = None if change is None else float(change)
        per_col = bool(rng.random() < 0.3)
        mc = float(rng.choice([2.0, 1.5, 3.0]))
        variant = str(rng.choice(['jump', 'jump_near', 'percol', 'divergent', 'zero', 'nanadj']))
        tags = [variant, interval]
        a = int(rng.integers(1, n))
        m = change if change is not None else mc * float(rng.uniform(0.7, 1.4))
        if variant == 'jump':
            scale_rows(w, slice(0, a), m)
        elif variant == 'jump_near':
            scale_rows(w, slice(0, a), m * float(rng.uniform(0.6, 1.0)) if m > 1 else m / float(rng.uniform(0.6, 1.0)))
        elif variant == 'percol':
            cols = ''.join(rng.choice(list('ohlc'), size=int(rng.integers(1, 4)), replace=False))
            scale_rows(w, slice(0, a), m, cols=cols)
        elif variant == 'divergent':
            scale_rows(w, slice(0, a), m, cols='oc')
            scale_rows(w, slice(0, a), float(rng.uniform(0.8, 1.2)) * m, cols='h')
        elif variant == 'zero':
            scale_rows(w, slice(0, a), m)
            for i in rng.choice(np.arange(n), size=int(rng.integers(1, 4)), replace=False):
                w[str(rng.choice(list('ohlc')))][i] = 0.0
        elif variant == 'nanadj':
            scale_rows(w, slice(0, a), m)
            w['a'][int(rng.integers(0, n))] = np.nan
        tags += ['pc' if per_col else 'nopc']
        cases.append(('detect', f'detect-{k:03d}-' + '-'.join(tags), interval, 'America/New_York',
                      dates(n, interval, 'America/New_York'), w,
                      {'change': change, 'perColumn': per_col, 'minChange': mc}))
    return cases


# -------------------------------------------------------------- running ---

def run_case(kind, name, interval, tz, t, w, extra):
    df = frame(t, w['o'], w['h'], w['l'], w['c'], w['a'], w['v'], w['d'], w['s'], tz=tz)
    case = {'name': name, 'kind': kind, 'interval': interval, 'tz': tz}
    case.update({k: v for k, v in extra.items() if k != 'hourly'})
    ph = new_ph(tz)
    tracer = Tracer(ph)
    if kind == 'detect':
        df = df.sort_index(ascending=False)
        case['bars'] = enc_frame(df)
        det = ph._detect_price_shifts(df, interval, extra['change'],
                                      correct_columns_individually=extra['perColumn'],
                                      min_change=extra['minChange'])
        case['want'] = {'detected': masks(det), 'threshold': tracer.threshold}
        return case
    df['Repaired?'] = False
    case['bars'] = enc_frame(df)
    if kind == 'split':
        out = ph._fix_bad_stock_splits(df.copy(), interval, tz)
    elif kind == 'unit_switch':
        ph._history_metadata = {'currency': extra['currency']}
        if extra['currencyRepaired']:
            ph._history_metadata['currencyRepaired'] = True
        out = ph._fix_unit_switch(df.copy(), interval, tz)
    elif kind == 'level_shift':
        hourly = extra['hourly']
        case['hourly'] = hourly
        df_1h = frame(hourly['t'], hourly['o'], hourly['h'], hourly['l'], hourly['c'],
                      hourly['a'], hourly['v'], tz=tz)
        requested = []

        def fake_history(self, *args, **kwargs):
            requested.append(kwargs['interval'])
            start = pd.Timestamp(kwargs['start']).tz_localize(tz)
            end = pd.Timestamp(kwargs['end']).tz_localize(tz)
            return df_1h[(df_1h.index >= start) & (df_1h.index < end)]

        now = pd.Timestamp(extra['now'], unit='s', tz='UTC')
        with mock.patch.object(yfh.PriceHistory, 'history', fake_history), \
                mock.patch.object(pd.Timestamp, 'now', staticmethod(lambda tz=None: now if tz is None else now.tz_convert(tz))):
            out = ph._fix_unexplained_level_shifts(df.copy(), interval, tz)
        case['fetches'] = len(requested)
    else:
        raise ValueError(kind)
    out = out.sort_index()
    if 'Repaired?' in out.columns:
        out['Repaired?'] = out['Repaired?'].fillna(False).astype(bool)
    case['want'] = enc_want(df, out)
    case['calls'] = tracer.calls
    return case


def load_fixture(name, tz):
    df = pd.read_csv(os.path.join(FIXTURES, name), index_col=0)
    df.index = pd.to_datetime(df.index, utc=True).tz_convert(tz)
    df = df.sort_index()
    if 'Dividends' not in df.columns:
        df['Dividends'] = 0.0
    if 'Stock Splits' not in df.columns:
        df['Stock Splits'] = 0.0
    df['Volume'] = df['Volume'].fillna(0).round().astype('int64')
    t = [int(ts.timestamp()) for ts in df.index]
    w = {k: df[c].to_numpy(dtype=float if k != 'v' else 'int64') for k, c in COLS.items()}
    return t, w


def fixture_cases():
    out = []
    splits = [('4063-T', 'Asia/Tokyo'), ('ALPHA-PA', 'Europe/Paris'), ('CNE-L', 'Europe/London'),
              ('DEX-AX', 'Australia/Sydney'), ('LA-V', 'America/Toronto'), ('MOB-ST', 'Europe/Stockholm'),
              ('NFE', 'America/New_York'), ('NRDY', 'America/New_York'), ('SPM-MI', 'Europe/Rome')]
    for tkr, tz in splits:
        t, w = load_fixture(f'{tkr}-1d-bad-stock-split.csv', tz)
        out.append(('split', f'fixture-{tkr}-1d', '1d', tz, t, w, {}))
        if tkr == 'NRDY':
            out.append(('split', 'fixture-NRDY-1d-last27', '1d', tz, t[-27:],
                        {k: v[-27:] for k, v in w.items()}, {}))
    t, w = load_fixture('AV-L-1wk-bad-stock-split.csv', 'Europe/London')
    out.append(('split', 'fixture-AV-L-1wk-as-1d', '1d', 'Europe/London', t, w, {}))
    out.append(('split', 'fixture-AV-L-1wk', '1wk', 'Europe/London', t, w, {}))
    for tkr, tz, cur in [('AET-L', 'Europe/London', 'GBP'), ('SSW-JO', 'Africa/Johannesburg', 'ZAR')]:
        t, w = load_fixture(f'{tkr}-1d-100x-error.csv', tz)
        for cr in (False, True):
            out.append(('unit_switch', f'fixture-{tkr}-1d-cr{int(cr)}', '1d', tz, t, w,
                        {'currency': cur, 'currencyRepaired': cr}))
    t, w = load_fixture('ASAI-L-1h-bad-unit.csv', 'Europe/London')
    for cr in (False, True):
        out.append(('unit_switch', f'fixture-ASAI-L-1h-cr{int(cr)}', '1h', 'Europe/London', t, w,
                    {'currency': 'GBP', 'currencyRepaired': cr}))
    for tkr, kind in [('SOXS', 'bad'), ('KALA', 'no-bad')]:
        tz = 'America/New_York'
        t, w = load_fixture(f'{tkr}-1d-{kind}-level-shift.csv', tz)
        th, wh = load_fixture(f'{tkr}-1h-{kind}-level-shift.csv', tz)
        hourly = {k: [fnum(x) if k != 'v' else int(x) for x in wh[k]] for k in 'ohlcav'}
        hourly['t'] = th
        out.append(('level_shift', f'fixture-{tkr}', '1d', tz, t, w,
                    {'now': t[-1] + 7 * 86400, 'hourly': hourly}))
    # Split fixtures through the level-shift path (no 1h data near: none fetched).
    return out


# --------------------------------------------------------------- mining ---
#
# Random walks rarely land on the inputs where a detail of the upstream
# algorithm matters. Each mutation below removes or alters one such detail in
# a copy of upstream's history.py; candidate cases whose repaired output
# changes under the mutation are kept, so the corpus pins every detail.

MUTATIONS = [
    ('ratio_of_medians', '_1d_change_denoised = np.median(_1d_change_x, axis=1)',
     '_1d_change_denoised = np.concatenate([[1.0], np.median(price_data, axis=1)[1:] / np.median(price_data, axis=1)[:-1]])'),
    ('ohlc_for_multiday', "price_data_cols = ['Open','Close']", 'price_data_cols = OHLC'),
    ('median_for_multiday', '_1d_change_denoised = np.average(_1d_change_x, axis=1)',
     '_1d_change_denoised = np.median(_1d_change_x, axis=1)'),
    ('no_zero_close_rows', 'if f_zero_num_denom.any():', 'if False:'),
    ('nan_adj_as_one', 'df_dtype = price_data.dtype', 'adj = np.where(np.isnan(adj), 1.0, adj); df_dtype = price_data.dtype'),
    ('no_early_exit', 'if np.max(_1d_change_denoised) <', 'if False and np.max(_1d_change_denoised) <'),
    ('std_ddof1', 'sd = np.std(_1d_change_denoised[f])', 'sd = np.std(_1d_change_denoised[f], ddof=1)'),
    ('local_idx_end', 'idx_end = min(len(df)-1, idx+2)', 'idx_end = min(len(df), idx+2)'),
    ('no_round3', "df_workings['1D %'] = df_workings['1D %'].round(3)", 'pass'),
    ('volspike_whole_block', 'block = df.iloc[flat_indices[start:end]]', 'block = df.iloc[flat_indices]'),
    ('volspike_excl_down', 'nf_up_shifts = ~f_up_shifts',
     'nf_up_shifts = ~(f_up_shifts | (f_down if f_down.ndim == 1 else f_down.any(axis=1)))'),
    ('no_start_min', 'start_min = (df.index[f].min() - _dateutil.relativedelta.relativedelta(years=1)).date()',
     'start_min = None'),
    ('no_per_column', 'if unit_switch:\n                correct_columns_individually = True',
     'if unit_switch:\n                correct_columns_individually = False'),
    ('no_suspended', 'appears_suspended = f_no_activity.any() and np.where(f_no_activity)[0][0]==0',
     'appears_suspended = False'),
    ('no_100x_abort', 'if change in [100.0, 0.01] and f_splits.any():', 'if False:'),
    ('no_currency_revert', "if 'currencyRepaired' in self._history_metadata and self._history_metadata['currencyRepaired']:",
     'if False:'),
]


def mutated_class(old, new):
    src = open(yfh.__file__).read()
    assert src.count(old) == 1, old
    ns = dict(vars(yfh))
    exec(compile(src.replace(old, new), yfh.__file__ + ':mutated', 'exec'), ns)
    return ns['PriceHistory']


def run_quiet(case, cls):
    orig = yfh.PriceHistory
    yfh.PriceHistory = cls
    try:
        return run_case(*case)
    finally:
        yfh.PriceHistory = orig


def mine(rng, per_mutation=6, attempts=500):
    gens = [lambda r: gen_split_cases(r, 1), lambda r: gen_unit_cases(r, 1), lambda r: gen_level_cases(r, 1)]
    out = []
    for name, old, new in MUTATIONS:
        cls = mutated_class(old, new)
        found = 0
        for k in range(attempts):
            case = gens[k % len(gens)](rng)[0]
            want = run_case(*case)['want']
            if run_quiet(case, cls)['want'] != want:
                kind, cname, *rest = case
                out.append((kind, f'mined-{name}-{found}-' + cname.split('-', 2)[2], *rest))
                found += 1
                if found >= per_mutation:
                    break
        print(f'mutation {name}: {found} cases in {k + 1} attempts')
    return out


def family_global(seed, f):
    """Split case whose bad range is off the split ratio by f: crosses the
    global detection threshold as f varies."""
    rng = np.random.default_rng(seed)
    interval = str(rng.choice(['1d', '1d', '1wk', '1mo', '3mo']))
    n = int(rng.integers(20, 70))
    tz = 'America/New_York'
    w = walk(rng, n, interval)
    split = float(rng.choice(SPLITS))
    si = int(rng.integers(n // 2, n - 1))
    w['s'][si] = split
    a = int(rng.integers(0, si // 2 + 1))
    m = split * f if split > 1 else split / f
    scale_rows(w, slice(a, si), m)
    if rng.random() < 0.5:
        # Wide wicks that only High/Low see.
        for i in rng.choice(np.arange(n), size=int(rng.integers(1, 5)), replace=False):
            w['h'][i] *= float(rng.uniform(1.2, 3))
            w['l'][i] /= float(rng.uniform(1.2, 3))
    if rng.random() < 0.4:
        w['a'][int(rng.choice([si, si - 1, a, max(a - 1, 0)]))] = np.nan
    if rng.random() < 0.3:
        w['c'][int(rng.choice([si, si - 1, a]))] = 0.0
    return ('split', f'family-global-{seed}-{f:.6f}', interval, tz, dates(n, interval, tz), w, {})


def family_local(seed, f):
    """Split case with a volatile stretch around a jump at the oldest rows:
    crosses the local-volatility threshold as f varies."""
    rng = np.random.default_rng(seed)
    interval = str(rng.choice(['1d', '1d', '1wk']))
    n = int(rng.integers(16, 40))
    tz = 'America/New_York'
    w = walk(rng, n, interval, sd=0.02)
    k = int(rng.integers(6, 14))
    noisy = walk(rng, k, interval, sd=float(rng.uniform(0.1, 0.35)))
    for c in 'ohlca':
        w[c][:k] = noisy[c] / noisy['c'][-1] * w['c'][k]
    split = float(rng.choice([2.0, 3.0, 0.5, 1.5, 4.0]))
    si = n - 2
    w['s'][si] = split
    b = int(rng.integers(1, 5))
    scale_rows(w, slice(0, b), split * f if split > 1 else split / f)
    return ('split', f'family-local-{seed}-{f:.6f}', interval, tz, dates(n, interval, tz), w, {})


def family_volspike(seed, f):
    """Split case with up and down signals and a volume spike of size f on
    the drop: crosses the volume-spike z-score as f varies."""
    rng = np.random.default_rng(seed)
    n = int(rng.integers(30, 80))
    tz = 'America/New_York'
    w = walk(rng, n, '1d', sd=0.01)
    w['v'][:] = np.round(w['v'].mean() * np.exp(rng.normal(0, 0.15, n))).astype('int64')
    split = float(rng.choice([2.0, 3.0, 10.0]))
    si = n - int(rng.integers(2, 6))
    w['s'][si] = split
    a = int(rng.integers(1, si // 2))
    b = int(rng.integers(a + 2, si))
    scale_rows(w, slice(a, b), split, vol=bool(rng.random() < 0.5))
    if rng.random() < 0.5:
        c = int(rng.integers(b + 1, si + 1))
        scale_rows(w, slice(c, min(n, c + int(rng.integers(1, 4)))), 1 / split, vol=False)
    j = int(rng.choice([a, b]))
    w['v'][j:j + int(rng.integers(1, 3))] = np.round(w['v'][j] * f * 3).astype('int64')
    if rng.random() < 0.5:
        w['v'][int(rng.integers(0, n))] = 0
    return ('split', f'family-volspike-{seed}-{f:.6f}', '1d', tz, dates(n, '1d', tz), w, {})


def signature(case_out):
    want = case_out['want']
    return (tuple(want['rep']), tuple(sorted(k for k in want if k != 'rep')))


BOUNDARY_FAMILIES = [
    (family_global, ['ohlc_for_multiday', 'median_for_multiday', 'nan_adj_as_one', 'std_ddof1',
                     'ratio_of_medians', 'no_zero_close_rows']),
    (family_local, ['local_idx_end', 'no_round3']),
    (family_volspike, ['volspike_excl_down', 'volspike_whole_block']),
]


def mine_boundaries(rng, per_mutation=5, attempts=120):
    """Bisect each family's parameter to where upstream's outcome flips and
    keep the near-boundary cases that a mutation decides differently."""
    muts = {name: mutated_class(old, new) for name, old, new in MUTATIONS}
    out = []
    for family, names in BOUNDARY_FAMILIES:
        found = {name: 0 for name in names}
        for _ in range(attempts):
            if all(v >= per_mutation for v in found.values()):
                break
            seed = int(rng.integers(1 << 30))
            lo, hi = 0.4, 2.5
            s_lo, s_hi = signature(run_case(*family(seed, lo))), signature(run_case(*family(seed, hi)))
            points = [lo, hi]
            if s_lo != s_hi:
                for _ in range(40):
                    mid = (lo + hi) / 2
                    s_mid = signature(run_case(*family(seed, mid)))
                    if s_mid == s_lo:
                        lo = mid
                    else:
                        hi, s_hi = mid, s_mid
                points = [lo, hi, lo * (1 - 1e-4), hi * (1 + 1e-4), lo * (1 - 1e-3), hi * (1 + 1e-3)]
            for f in points:
                case = family(seed, f)
                want = run_case(*case)['want']
                for name in names:
                    if found[name] >= per_mutation:
                        continue
                    if run_quiet(case, muts[name])['want'] != want:
                        kind, cname, *rest = case
                        out.append((kind, f'mined-{name}-b{found[name]}-{cname}', *rest))
                        found[name] += 1
        print(f'boundary {family.__name__}: {found}')
    return out


def numpy_cases(rng, count=60):
    """Exact NumPy reductions, to pin the bit-level port in numpy.go."""
    out = []
    for k in range(count):
        n = int(rng.integers(1, 300)) if k > 3 else [1, 2, 7, 8][k]
        x = rng.random(n) * float(rng.choice([1e-3, 1.0, 37.0, 1e6])) + float(rng.choice([0.0, 1.0, 1e3]))
        out.append({
            'x': [float(v) for v in x],
            'sum': float(np.sum(x)), 'mean': float(np.mean(x)), 'std0': float(np.std(x)),
            'std1': float(np.std(x, ddof=1)) if n > 1 else None,
            'p25': float(np.percentile(x, 25)), 'p75': float(np.percentile(x, 75)),
            'median': float(np.median(x)), 'round3': [float(v) for v in np.round(x, 3)],
        })
    return out


def main():
    np.seterr(all='ignore')
    import warnings
    warnings.simplefilter('ignore')
    rng = np.random.default_rng(20261104)
    groups = {
        'split': gen_split_cases(rng, 160),
        'unit_switch': gen_unit_cases(rng, 140),
        'level_shift': gen_level_cases(rng, 70),
        'detect': gen_detect_cases(rng, 90),
        'fixtures': fixture_cases(),
        'mined': mine(np.random.default_rng(1104)) + mine_boundaries(np.random.default_rng(2026)),
    }
    for group, cases in groups.items():
        results = [run_case(*c) for c in cases]
        doc = {'upstream': 'yfinance ' + yf_version, 'generator': 'gen_oracle.py', 'cases': results}
        fp = os.path.join(OUT, f'{group}.json')
        with open(fp, 'w') as f:
            json.dump(doc, f, separators=(',', ':'), allow_nan=False)
            f.write('\n')
        print(f'{fp}: {len(results)} cases, {os.path.getsize(fp)} bytes')
    fp = os.path.join(OUT, 'numpy.json')
    with open(fp, 'w') as f:
        json.dump({'numpy': np.__version__, 'cases': numpy_cases(np.random.default_rng(7))}, f,
                  separators=(',', ':'), allow_nan=False)
        f.write('\n')
    print(f'{fp}: {os.path.getsize(fp)} bytes')


if __name__ == '__main__':
    main()
