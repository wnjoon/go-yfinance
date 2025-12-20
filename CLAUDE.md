# go-yfinance Development Guidelines

This file contains project-specific instructions for Claude Code when working on go-yfinance.

## Project Overview

go-yfinance is a Go port of Python yfinance library for accessing Yahoo Finance data.

**Key Files:**
- `STATUS.md` - Implementation progress and phase tracking
- `DESIGN.md` - Architecture and design details
- `CONTRIBUTING.md` - Development workflow and code style guidelines
- `docs/API.md` - Auto-generated API documentation (gomarkdoc)

**Python yfinance Reference (상위 디렉토리):**
- `../yfinance/` - Python yfinance 소스 코드 (API 참조용)
- `../YFINANCE_ANALYSIS.md` - Python yfinance 분석 문서

## Phase Development Workflow

### Before Starting a New Phase

1. Read `STATUS.md` to understand current progress
2. Verify previous phase is merged and pushed to main
3. Check that `docs/API.md` is up to date

### During Development

1. Create phase branch from main:
   ```bash
   git checkout main && git pull origin main
   git checkout -b phase{N}/{feature-name}
   ```

2. Follow existing code patterns in `pkg/ticker/` and `pkg/models/`

3. Write unit tests for all new functionality

4. Update documentation:
   - `pkg/ticker/doc.go` - Add new Ticker methods
   - `pkg/models/doc.go` - Add new model types
   - `STATUS.md` - Update completion status

### API Consistency Check (CRITICAL)

**IMPORTANT: 웹 검색 대신 로컬 Python yfinance 소스를 참조할 것!**

Python yfinance 소스 코드 위치 (`../yfinance/`):
- `ticker.py` - Ticker 클래스
- `tickers.py` - Tickers (멀티 티커) 클래스
- `multi.py` - download() 함수
- `search.py` - Search 기능
- `screener/` - Screener 기능
- `base.py` - 기본 클래스 및 공통 로직

**Before completing any phase, verify consistency with Python yfinance:**

| Check | Description |
|-------|-------------|
| Method Names | Go names should match Python (PascalCase conversion) |
| Parameters | Input parameters should match Python's interface |
| Return Types | Output structures should contain equivalent fields |

**Python yfinance method mapping:**

| Python | Go |
|--------|-----|
| `ticker.major_holders` | `MajorHolders()` |
| `ticker.institutional_holders` | `InstitutionalHolders()` |
| `ticker.mutualfund_holders` | `MutualFundHolders()` |
| `ticker.insider_transactions` | `InsiderTransactions()` |
| `ticker.insider_roster_holders` | `InsiderRosterHolders()` |
| `ticker.insider_purchases` | `InsiderPurchases()` |
| `ticker.calendar` | `Calendar()` |
| `ticker.recommendations` | `Recommendations()` |
| `ticker.analyst_price_targets` | `AnalystPriceTargets()` |
| `ticker.earnings_estimate` | `EarningsEstimate()` |
| `ticker.revenue_estimate` | `RevenueEstimate()` |
| `ticker.eps_trend` | `EPSTrend()` |
| `ticker.eps_revisions` | `EPSRevisions()` |
| `ticker.earnings_history` | `EarningsHistory()` |
| `ticker.growth_estimates` | `GrowthEstimates()` |

### Phase Completion Checklist

Before merging to main:

1. **Tests pass**: Run `go test ./... -v`
2. **Build succeeds**: Run `go build ./...`
3. **API consistency verified** with Python yfinance
4. **Documentation updated** (doc.go, STATUS.md)
5. **Generate API docs**:
   ```bash
   make docs
   ```
6. **Review generated docs**: Check `docs/API.md` for accuracy
7. **Commit all changes** including `docs/API.md`
8. **Push to remote**: `git push origin main`

## Code Style

- Use gomarkdoc-compatible documentation comments
- Include Example sections in doc comments
- Add deprecated aliases when renaming methods:
  ```go
  // NewMethod is the current implementation.
  func (t *Ticker) NewMethod() (*Result, error) { ... }

  // OldMethod is deprecated. Use NewMethod instead.
  //
  // Deprecated: Use NewMethod instead.
  func (t *Ticker) OldMethod() (*Result, error) {
      return t.NewMethod()
  }
  ```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the project |
| `make test` | Run tests |
| `make docs` | Generate API documentation |
| `make lint` | Run golangci-lint |

## Known Issues

1. **CycleTLS Close Panic**: Wrapped with `recover()` to handle gracefully
2. **Shell cd issues**: Use `GIT_DIR` and `GIT_WORK_TREE` env vars for git commands

## Directory Structure

```
go-yfinance/
├── pkg/
│   ├── client/     # HTTP client with TLS fingerprinting
│   ├── config/     # Configuration management
│   ├── models/     # Data structures
│   └── ticker/     # Main Ticker interface
├── internal/
│   └── endpoints/  # Yahoo Finance API endpoints
├── docs/
│   └── API.md      # Generated API documentation
└── examples/       # Usage examples
```
