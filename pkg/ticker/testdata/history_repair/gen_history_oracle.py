"""Generate the History repair-pipeline fixture and its upstream goldens.

Builds one raw Yahoo chart payload (chart.json) containing a missing 2:1
split adjustment, a correctly adjusted dividend, a fully null OHLC row, a
row with a null Open and a 100x unit-switch block, then runs Python yfinance
1.8.0's PriceHistory.history() on it (network mocked) for every combination
of repair, actions, auto_adjust and keepna. Results go to expected.json.

A variant whose 4% dividend is missing from Adj Close
(chart_missing_div_adj.json) is run with actions=True, keepna=False and
repair / auto_adjust on and off (expected_missing_div_adj.json).

Needs python-yfinance 1.8.0's dependencies plus scipy and scikit-learn
(imported by its 100x and reconstruction repairs). Output is deterministic.

usage: YFINANCE_UPSTREAM=/path/to/python-yfinance python -I gen_history_oracle.py
"""
import datetime as dt
import json
import math
import os
import sys
from unittest import mock

UPSTREAM = os.environ.get('YFINANCE_UPSTREAM', '')
if not UPSTREAM:
    sys.exit('set YFINANCE_UPSTREAM to a python-yfinance 1.8.0 checkout')
sys.dont_write_bytecode = True
sys.path.insert(0, UPSTREAM)

import numpy as np  # noqa: E402
import pandas as pd  # noqa: E402
from yfinance.scrapers import history as yfh  # noqa: E402
from yfinance.version import version as yf_version  # noqa: E402
from yfinance.config import YfConfig  # noqa: E402

if yf_version != '1.8.0':
    sys.exit(f'need python-yfinance 1.8.0, found {yf_version}')

HERE = os.path.dirname(os.path.abspath(__file__))
TZ = 'America/New_York'

# Row layout (oldest first).
N = 120
NULL_ROW = 15      # every OHLC / Adj Close / Volume null
DIV_ROW = 25       # dividend 1.0, Adj Close correctly adjusted
SPLIT_ROW = 45     # 2:1 split, older rows not split-adjusted
OPEN_NULL_ROW = 60  # only Open null
UNIT_ROWS = range(85, 91)  # 100x too high (cents), Adj Close too


def build_payload(div_adjusted=True, dividend=1.0):
    rng = np.random.default_rng(17)
    days = pd.bdate_range('2021-02-01', periods=N)
    ts = [int(pd.Timestamp(d.date()).tz_localize(TZ).replace(hour=9, minute=30).timestamp()) for d in days]
    close = 100 * np.exp(np.cumsum(rng.normal(0, 0.01, N)))
    opn = close * (1 + rng.normal(0, 0.004, N))
    high = np.maximum(opn, close) * (1 + np.abs(rng.normal(0, 0.006, N)))
    low = np.minimum(opn, close) * (1 - np.abs(rng.normal(0, 0.006, N)))
    vol = rng.integers(900_000, 1_100_000, N).astype(float)
    # Missing split adjustment: prices before the split are 2x, volume half.
    pre = np.arange(N) < SPLIT_ROW
    for a in (opn, high, low, close):
        a[pre] *= 2
    vol[pre] = np.round(vol[pre] / 2)
    # Price drop on the ex-dividend day.
    for a in (opn, high, low, close):
        a[DIV_ROW:] -= dividend
    # Dividend reflected in Adj Close (or missing from it).
    adj = close.copy()
    if div_adjusted:
        adj[:DIV_ROW] *= 1 - dividend / close[DIV_ROW - 1]
    # 100x unit switch.
    for i in UNIT_ROWS:
        for a in (opn, high, low, close, adj):
            a[i] *= 100

    def r(x):
        return [None if x is None else round(float(v), 6) for v in x]

    o, h, l, c, a = r(opn), r(high), r(low), r(close), r(adj)
    v = [int(x) for x in vol]
    for col in (o, h, l, c, a):
        col[NULL_ROW] = None
    v[NULL_ROW] = None
    o[OPEN_NULL_ROW] = None

    meta = {
        'currency': 'USD', 'symbol': 'MTRX', 'exchangeName': 'NMS',
        'instrumentType': 'EQUITY', 'firstTradeDate': ts[0] - 86400 * 365,
        'regularMarketTime': ts[-1] + 6 * 3600 + 1800, 'gmtoffset': -18000,
        'timezone': 'EST', 'exchangeTimezoneName': TZ, 'regularMarketPrice': c[-1],
        'priceHint': 2, 'dataGranularity': '1d', 'range': '1y',
        'validRanges': ['1d', '5d', '1mo', '3mo', '6mo', '1y', '2y', '5y', '10y', 'ytd', 'max'],
    }
    events = {
        'dividends': {str(ts[DIV_ROW]): {'amount': dividend, 'date': ts[DIV_ROW]}},
        'splits': {str(ts[SPLIT_ROW]): {'date': ts[SPLIT_ROW], 'numerator': 2.0, 'denominator': 1.0, 'splitRatio': '2:1'}},
    }
    return {'chart': {'result': [{
        'meta': meta, 'timestamp': ts, 'events': events,
        'indicators': {'quote': [{'open': o, 'high': h, 'low': l, 'close': c, 'volume': v}],
                       'adjclose': [{'adjclose': a}]},
    }], 'error': None}}


class FakeResponse:
    def __init__(self, payload):
        self._payload = payload
        self.text = json.dumps(payload)

    def json(self):
        return json.loads(self.text)


class FakeData:
    def __init__(self, payload):
        self.payload = payload

    def get(self, url, params=None, timeout=None):
        if params.get('interval') != '1d':
            raise AssertionError(f'unexpected request {params}')
        return FakeResponse(self.payload)

    cache_get = get


def enc(x):
    if x is None:
        return None
    x = float(x)
    return None if math.isnan(x) else x


def run(payload, repair, actions, auto_adjust, keepna):
    ph = yfh.PriceHistory(FakeData(payload), 'MTRX', TZ, session=object())
    df = ph.history(period='1y', interval='1d', repair=repair, actions=actions,
                    auto_adjust=auto_adjust, keepna=keepna)
    cols = {}
    names = {'Open': 'o', 'High': 'h', 'Low': 'l', 'Close': 'c', 'Adj Close': 'a', 'Volume': 'v',
             'Dividends': 'd', 'Stock Splits': 's', 'Capital Gains': 'cg', 'Repaired?': 'rep'}
    for col, key in names.items():
        if col not in df.columns:
            continue
        if col == 'Volume':
            cols[key] = [int(x) for x in df[col]]
        elif col == 'Repaired?':
            cols[key] = [i for i, x in enumerate(df[col]) if x]
        else:
            cols[key] = [enc(x) for x in df[col]]
    # Daily index -> the chart timestamp of the same exchange date.
    by_date = {pd.Timestamp(t, unit='s', tz='UTC').tz_convert(TZ).date(): t
               for t in payload['chart']['result'][0]['timestamp']}
    cols['t'] = [by_date[d.date()] for d in df.index]
    cols['columns'] = list(df.columns)
    return cols


def main():
    YfConfig.debug.hide_exceptions = False
    payload = build_payload()
    with open(os.path.join(HERE, 'chart.json'), 'w') as f:
        json.dump(payload, f, separators=(',', ':'))
        f.write('\n')
    cases = []
    for repair in (False, True):
        for actions in (False, True):
            for auto_adjust in (False, True):
                for keepna in (False, True):
                    cases.append({'repair': repair, 'actions': actions, 'autoAdjust': auto_adjust,
                                  'keepna': keepna, 'want': run(payload, repair, actions, auto_adjust, keepna)})
    with open(os.path.join(HERE, 'expected.json'), 'w') as f:
        json.dump({'upstream': yf_version, 'cases': cases}, f, separators=(',', ':'))
        f.write('\n')

    # Variant: a 4% dividend missing from Adj Close, which repair adds.
    payload = build_payload(div_adjusted=False, dividend=4.0)
    with open(os.path.join(HERE, 'chart_missing_div_adj.json'), 'w') as f:
        json.dump(payload, f, separators=(',', ':'))
        f.write('\n')
    cases = []
    for repair in (False, True):
        for auto_adjust in (False, True):
            cases.append({'repair': repair, 'actions': True, 'autoAdjust': auto_adjust, 'keepna': False,
                          'want': run(payload, repair, True, auto_adjust, False)})
    with open(os.path.join(HERE, 'expected_missing_div_adj.json'), 'w') as f:
        json.dump({'upstream': yf_version, 'cases': cases}, f, separators=(',', ':'))
        f.write('\n')


if __name__ == '__main__':
    main()
