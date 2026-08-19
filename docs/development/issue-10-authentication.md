# Issue #10 Authentication Hardening Specification

> Last Updated: 2026-08-19
>
> Tracking: [GitHub issue #10](https://github.com/wnjoon/go-yfinance/issues/10)

## Overview

Issue #10 reports that quote requests succeed locally but fail on Render with:

```text
failed to get crumb: authentication failed: failed to extract CSRF tokens
```

The current error does not identify the first authentication failure. The
authentication manager tries the Basic cookie/crumb strategy, falls back to
the CSRF consent strategy, and returns only the error from the final strategy.
The CSRF path also parses the consent response before classifying HTTP errors,
so a rate-limit or denial response can be reported as a token extraction
failure.

This work fixes the library behavior that can be verified independently of
Render. It does not claim that Yahoo will accept requests from any particular
hosting provider or outbound IP range.

## Branch Structure

```text
main
 └── fix/issue-10-authentication
```

## Goals

- Preserve the failures from both authentication strategies when fallback
  also fails.
- Classify HTTP failures before parsing authentication response bodies.
- Make authentication-stage HTTP 429 responses detectable with
  `client.IsRateLimitError`.
- Use the upstream-compatible crumb endpoint for each strategy: `query1` for
  Basic and `query2` for CSRF.
- Include enough stage and status context to diagnose failures without
  exposing cookies, crumb values, CSRF tokens, or session IDs.
- Add deterministic regression tests that do not depend on live Yahoo or
  Render services.

## Non-Goals

- Guarantee successful Yahoo requests from Render or another cloud provider.
- Circumvent Yahoo rate limits, bot detection, or network policy.
- Add automatic proxy rotation, retries for HTTP 429, or provider-specific
  workarounds.
- Determine the reporter's Render region, outbound IP, request volume, or
  historical Yahoo response without data from that environment.
- Close issue #10 before the original environment is retested or the remaining
  provider-specific behavior is otherwise resolved.

## Current Behavior

### Fallback loses the first error

`AuthManager.refreshAuth` stores both attempts in the same `err` variable. If
Basic fails and CSRF also fails, callers see only the CSRF failure. This can
turn an initial `429 Too Many Requests` into the misleading message
`failed to extract CSRF tokens`.

### Consent HTTP errors are parsed as HTML

`fetchCSRF` extracts `csrfToken` and `sessionId` immediately after the consent
GET. It does not first reject 429, 401, 403, or other error responses.

The collect-consent and copy-consent steps similarly check transport errors
but do not reject HTTP error status codes.

### Rate-limit errors are not typed consistently

The package exposes `ErrRateLimit`, `WrapRateLimitError`, and
`IsRateLimitError`, but crumb acquisition currently returns a plain
`fmt.Errorf("rate limited")`. Consumers therefore cannot reliably classify
authentication-stage rate limits.

### Both strategies use `query2`

`CrumbURL` and `CrumbCSRFURL` both resolve to
`https://query2.finance.yahoo.com/v1/test/getcrumb`. The upstream strategy
uses `query1` for Basic crumb acquisition and `query2` for CSRF crumb
acquisition.

## Required Behavior

### Strategy fallback and error preservation

The existing fallback order remains unchanged:

- `StrategyBasic`: Basic, then CSRF.
- `StrategyCSRF`: CSRF, then Basic.

If the first attempt fails and the fallback succeeds, return the crumb without
an error. If both attempts fail, return one authentication error whose cause
preserves both labeled strategy errors. `errors.Is` and `errors.As` must still
reach typed causes inside the combined error.

The returned message must identify which failure belongs to `basic` and which
belongs to `csrf`. It must not include response bodies or authentication
secrets.

### HTTP response classification

Authentication responses must be classified before their bodies are parsed or
accepted:

CycleTLS can encode transport failures as synthetic responses with a zero or
HTTP-like status and the underlying error in the response body. Authentication
must classify those responses as sanitized network errors before applying the
HTTP or body rules below.

| Stage | Required handling |
|-------|-------------------|
| Basic cookie GET | Preserve current behavior where `fc.yahoo.com` can return a non-2xx status while setting a usable cookie; fail on transport failure, but do not reject the response based on status alone. |
| Basic crumb GET | 429 or a known rate-limit body becomes `ErrRateLimit`; other statuses >= 400 become an appropriate typed HTTP error; an empty or HTML body becomes `ErrInvalidResponse`. |
| CSRF consent GET | 429 becomes `ErrRateLimit`; other statuses >= 400 are rejected before token extraction; missing inputs on a successful response become `ErrInvalidResponse`. |
| Collect-consent POST | Any status >= 400 is rejected with stage and status context. |
| Copy-consent GET | Any status >= 400 is rejected with stage and status context. |
| CSRF crumb GET | Apply the same classification rules as the Basic crumb GET. |

The `fc.yahoo.com` exception is intentional: Yahoo commonly returns a 404 page
while still supplying the authentication cookie.

### Endpoint selection

Endpoint constants must resolve to:

```text
Basic crumb: https://query1.finance.yahoo.com/v1/test/getcrumb
CSRF crumb:  https://query2.finance.yahoo.com/v1/test/getcrumb
```

All other authentication endpoint URLs remain unchanged.

### Error contracts

- An authentication-stage 429 must satisfy
  `client.IsRateLimitError(err) == true`, including when it is one cause of a
  combined fallback failure.
- A final two-strategy failure must satisfy
  `client.IsAuthError(err) == true`.
- Invalid successful responses must remain distinguishable from HTTP status
  failures.
- Error text may contain the strategy, stage, endpoint label, and HTTP status.
- Error text must not contain cookie values, crumb values, CSRF tokens,
  session IDs, full response bodies, or proxy credentials.

## Implementation Approach

1. Change `endpoints.CrumbURL` to use `Query1URL`; keep
   `endpoints.CrumbCSRFURL` on `BaseURL` (`query2`).
2. Introduce small shared response validators for crumb and authentication
   HTTP responses so Basic and CSRF classification cannot drift.
3. Return existing typed client errors from authentication stages instead of
   plain rate-limit strings.
4. Label each strategy error and combine both failures with Go's multi-error
   unwrapping support, then wrap the result as `ErrAuth`.
5. Add an unexported authentication-client interface or equivalent test seam
   implemented by `*client.Client`. Keep `NewAuthManager(*Client)` and all
   other public APIs source-compatible.
6. Use a scripted fake client in unit tests to assert request order, selected
   endpoints, response classification, fallback behavior, and secret-free
   errors.

## Test Plan

- [x] Basic success uses `query1` and never invokes CSRF.
- [x] Basic failure followed by CSRF success returns the CSRF crumb.
- [x] CSRF failure followed by Basic success returns the Basic crumb.
- [x] Two failures retain labeled Basic and CSRF causes.
- [x] Basic crumb 429 is recognized by `IsRateLimitError`.
- [x] CycleTLS status-zero and synthetic HTTP-like transport responses become
  sanitized network errors before parsing.
- [x] CSRF consent 429 is recognized by `IsRateLimitError` before parsing.
- [x] CSRF consent 403 reports the HTTP failure instead of missing tokens.
- [x] Successful consent HTML without required inputs is an invalid-response
  error.
- [x] Collect-consent and copy-consent HTTP errors identify their stage.
- [x] HTTP 404 authentication errors retain stage and status context.
- [x] CSRF crumb acquisition uses `query2`.
- [x] Empty, full-document HTML, and HTML-fragment crumb bodies are invalid
  responses.
- [x] Combined fallback failure is recognized by both `IsAuthError` and, when
  applicable, `IsRateLimitError`.
- [x] Error strings do not contain fixture cookie, crumb, CSRF token, session
  ID, response-body, or proxy-secret values.
- [x] Existing login-cookie, subscription, and strategy-switch tests continue
  to pass.
- [x] `go test ./pkg/client/...`
- [x] `go test ./...`
- [x] `go test -race ./pkg/client/...`
- [x] `go vet ./...`

Live Yahoo checks may be run as a limited smoke test, but they are not part of
the deterministic acceptance criteria.

## Render Follow-Up

After the library-side changes are available, keep issue #10 open and ask the
reporter to retest from the original Render deployment. The follow-up should
collect only non-sensitive diagnostics:

- Render region and whether the instance is newly created.
- Authentication stage and HTTP status.
- Final hostname or known endpoint label.
- Sanitized response category, such as rate-limit text, consent HTML, or
  challenge HTML.

Do not request or log `Set-Cookie`, crumb, `csrfToken`, `sessionId`, proxy
credentials, or complete response bodies.

Possible follow-up outcomes:

| Result | Next action |
|--------|-------------|
| HTTP 429 from Render only | Treat as provider egress/IP rate limiting; document proxy or alternate-egress guidance without adding an automatic bypass. |
| HTTP 403 or challenge page | Review request fingerprint and headers using a separately scoped change. |
| HTTP 2xx consent page with changed markup | Update the parser with a captured, sanitized fixture. |
| `query1` succeeds and `query2` fails for Basic | Confirm the endpoint split addresses the original Basic-path failure. |
| Transport, DNS, or TLS failure | Investigate CycleTLS, proxy, and Render network behavior separately. |

## Acceptance Criteria

- All goals above are implemented without changing the public constructor or
  ticker APIs.
- Both fallback failures remain discoverable through Go error unwrapping.
- Authentication-stage 429 and final authentication errors are programmatically
  classifiable.
- Basic and CSRF use their specified crumb endpoints.
- Regression tests cover every deterministic failure path listed above.
- No secret or raw authentication response is added to errors or logs.
- The implementation PR references issue #10 but does not automatically close
  it or claim verified Render compatibility.

## Status Legend

- `[x]` Done
- `[ ]` Pending
- N/A - verified not applicable, with the reason recorded inline
