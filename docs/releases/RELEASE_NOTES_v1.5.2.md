# go-yfinance v1.5.2

**Python yfinance v1.5.2 Parity Release**

This release follows Python yfinance v1.5.2. Upstream v1.5.2 is a single-patch
release whose only functional change fixes yfinance breaking with
`curl_cffi >= 0.16`. That fix does not apply to Go, so this release carries no
behavior change; it advances the parity baseline and its documentation.

## Upstream Scope

Python yfinance v1.5.2 rewrote the caching-session detection in
`yfinance/data.py`. Older code treated a session as a caching session whenever
the object exposed a `.cache` attribute. `curl_cffi >= 0.16` sessions always
expose `.cache` (set to `None` when caching is disabled), so non-caching
`curl_cffi` sessions were wrongly detected and had `expire_after` injected into
every request, which broke them. The patch now tests for an *active* cache
(`getattr(session, "cache", None) is not None`), raises an exception for
caching sessions, and removes the `_session_is_caching` flag and all
`expire_after` branches.

## Why This Does Not Change go-yfinance

The upstream bug only exists at Python's boundary where a user hands yfinance an
external `requests`/`curl_cffi` session and yfinance must detect whether that
session caches responses. go-yfinance has no such boundary:

- The HTTP layer is the project-owned CycleTLS-backed client (`pkg/client`,
  built on `cycletls.CycleTLS`), not a user-supplied session. There is no
  session-injection point, so the `curl_cffi >= 0.16` `.cache` attribute
  problem cannot occur.
- The crumb, cookie, and CSRF-consent flows in `pkg/client/auth.go` already
  issue plain requests with no cache-expiry branching, so the removed Python
  code paths (`_session_is_caching`, per-request `expire_after`) have no
  analogue to change.
- go-yfinance caching is a first-party, opt-in config knob
  (`config.CacheEnabled` / `config.CacheTTL`, disabled by default) and is not a
  transport wrapper the auth path has to detect.

## Changes

- No functional code change. Parity baseline advanced from v1.5.1 to v1.5.2.
- Documentation updated to record the upstream patch and its no-op assessment
  for Go.

## Installation

```bash
go get github.com/wnjoon/go-yfinance@v1.5.2
```

## Documentation

- [API Reference](https://wnjoon.github.io/go-yfinance/API/)
- [v1.5.2 Progress](https://wnjoon.github.io/go-yfinance/development/v1.5.2-progress/)
