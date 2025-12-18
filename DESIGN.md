# Go yfinance - Design Document

## Overview

Python yfinance 라이브러리를 Go로 재구현하는 프로젝트. Yahoo Finance의 비공식 API를 활용하여 주식, 옵션, 재무제표 등의 금융 데이터를 조회한다.

## Background

### Python yfinance 분석 결과

| 항목 | 내용 |
|------|------|
| 총 코드량 | ~11,000줄 |
| 핵심 파일 | ~20개 |
| API 엔드포인트 | 20+ |
| 주요 의존성 | curl_cffi, pandas, beautifulsoup4, protobuf |

### 기존 Go 구현체 (yahoo-finance-api) 분석

| 항목 | 내용 |
|------|------|
| 총 코드량 | ~1,037줄 |
| 구현된 기능 | History, Quote, Options, Info (price 모듈만) |
| 완성도 | Python 대비 약 25% |
| 핵심 문제 | TLS Fingerprint 위장 미구현 |

### 새로 구현하는 이유

1. 기존 Go 구현체의 60% 이상 수정 필요
2. TLS 위장 미구현으로 Yahoo 차단 가능성 높음
3. 확장성 낮은 구조
4. 새로 설계하는 것이 더 효율적 (~17% 시간 절약 예상)

---

## Yahoo Finance Data Sources

### API Endpoints

| 데이터 종류 | 엔드포인트 | 방식 |
|------------|-----------|------|
| 가격 히스토리 (OHLCV) | `query2.finance.yahoo.com/v8/finance/chart/{ticker}` | JSON API |
| 회사 정보 (info) | `query2.finance.yahoo.com/v10/finance/quoteSummary/{ticker}` | JSON API |
| 재무제표 | `query2.finance.yahoo.com/ws/fundamentals-timeseries/v1/finance/timeseries/{ticker}` | JSON API |
| 검색 | `query2.finance.yahoo.com/v1/finance/search` | JSON API |
| 종목 조회 | `query1.finance.yahoo.com/v1/finance/lookup` | JSON API |
| 옵션 | `query2.finance.yahoo.com/v7/finance/options/{ticker}` | JSON API |
| 스크리너 | `query1.finance.yahoo.com/v1/finance/screener` | JSON API |
| 시장 요약 | `query1.finance.yahoo.com/v6/finance/quote/marketSummary` | JSON API |
| 실시간 스트리밍 | `wss://streamer.finance.yahoo.com/?version=2` | WebSocket + Protobuf |
| 어닝 캘린더 | `finance.yahoo.com/calendar/earnings?symbol={ticker}` | HTML 스크래핑 |

### Authentication Mechanism

Yahoo Finance는 Cookie + Crumb 기반 인증을 사용한다.

```
┌─────────────────────────────────────────────────────────────────┐
│                    Cookie/Crumb 인증 시스템                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [전략 1: Basic]                                                │
│   1. GET https://fc.yahoo.com → A3 쿠키 획득                    │
│   2. GET https://query1.finance.yahoo.com/v1/test/getcrumb      │
│      → crumb 토큰 획득                                          │
│   3. 모든 요청에 ?crumb={crumb} 파라미터 추가                    │
│                                                                 │
│  [전략 2: CSRF] (Basic 실패 시 폴백)                             │
│   1. GET https://guce.yahoo.com/consent → CSRF 토큰 추출        │
│   2. POST https://consent.yahoo.com/v2/collectConsent           │
│   3. GET https://guce.yahoo.com/copyConsent                     │
│   4. GET https://query2.finance.yahoo.com/v1/test/getcrumb      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### TLS Fingerprint (Critical)

Yahoo는 TLS fingerprint (JA3)를 통해 봇을 탐지한다. User-Agent와 TLS fingerprint가 불일치하면 차단된다.

**해결 방안**: CycleTLS 또는 utls 라이브러리를 사용하여 Chrome TLS fingerprint를 위장

```go
// CycleTLS 사용 예시
client := cycletls.Init()
response, err := client.Do(url, cycletls.Options{
    Ja3:       "771,4865-4866-4867-49195...",  // Chrome JA3
    UserAgent: "Mozilla/5.0 (Windows NT 10.0...",
}, "GET")
```

---

## Architecture

### Project Structure

```
go-yfinance/
├── cmd/
│   └── example/                 # 사용 예제
├── pkg/
│   ├── client/
│   │   ├── client.go           # CycleTLS 기반 HTTP 클라이언트
│   │   ├── auth.go             # Cookie/Crumb + CSRF 폴백
│   │   └── errors.go           # 커스텀 에러 타입
│   ├── config/
│   │   └── config.go           # 설정 관리
│   ├── ticker/
│   │   ├── ticker.go           # 메인 Ticker 인터페이스
│   │   ├── quote.go            # 현재가
│   │   ├── history.go          # 히스토리 OHLCV
│   │   ├── info.go             # 회사 정보
│   │   ├── options.go          # 옵션 체인
│   │   ├── financials.go       # 재무제표
│   │   ├── holders.go          # 보유 정보
│   │   ├── actions.go          # 배당/분할
│   │   └── analysis.go         # 애널리스트 데이터
│   ├── search/
│   │   └── search.go           # 검색/조회
│   ├── screener/
│   │   ├── screener.go         # 스크리너
│   │   └── query.go            # 쿼리 빌더
│   ├── multi/
│   │   ├── tickers.go          # 복수 종목 처리
│   │   └── download.go         # 배치 다운로드
│   ├── live/
│   │   ├── websocket.go        # WebSocket 클라이언트
│   │   └── decoder.go          # Protobuf 디코더
│   └── models/
│       ├── price.go            # 가격 데이터 모델
│       ├── quote.go            # Quote 모델
│       ├── option.go           # 옵션 모델
│       ├── financials.go       # 재무 모델
│       ├── holders.go          # 보유 모델
│       └── analysis.go         # 분석 모델
├── internal/
│   ├── endpoints/
│   │   └── endpoints.go        # API 엔드포인트 상수
│   └── parser/
│       └── parser.go           # JSON 파싱 유틸리티
├── go.mod
├── go.sum
├── README.md
└── DESIGN.md                   # 이 문서
```

### Core Design Principles

```go
// 1. CycleTLS 기반 클라이언트
type Client struct {
    cycleTLS *cycletls.CycleTLS
    auth     *AuthManager
    config   *Config
}

// 2. 상태 머신 기반 인증
type AuthManager struct {
    mu       sync.RWMutex
    cookie   string
    crumb    string
    strategy AuthStrategy  // Basic → CSRF 자동 전환
    expiry   time.Time
}

// 3. 통일된 에러 체계
type YFError struct {
    Code    ErrorCode
    Message string
    Cause   error
}

// 4. 인터페이스 기반 확장성
type DataFetcher interface {
    Fetch(symbol string, params map[string]string) ([]byte, error)
}
```

---

## Implementation Roadmap

### Phase 0: Foundation (기반) - 필수

모든 기능의 토대가 되는 핵심 인프라

**구현 항목**:
- HTTP 클라이언트 (CycleTLS 기반)
- 인증 시스템 (Cookie/Crumb + CSRF 폴백)
- 에러 처리 체계
- 설정 관리 (Config)
- 기본 프로젝트 구조

**예상 코드량**: ~500줄

**완료 기준**: Yahoo Finance API에 성공적으로 인증된 요청을 보낼 수 있음

---

### Phase 1: Core Data (핵심 데이터) - 필수

가장 많이 사용되는 기본 기능

**구현 항목**:
- Ticker 기본 구조
- Quote (현재가)
- History (OHLCV 히스토리)
- Info (기본 회사 정보)
- 기본 모델 구조체

**예상 코드량**: ~600줄

**완료 기준**: `ticker.Quote()`, `ticker.History()`, `ticker.Info()` 동작

---

### Phase 2: Options (옵션 데이터) - 권장

옵션 트레이더를 위한 기능

**구현 항목**:
- Option Chain (콜/풋)
- Expiration Dates
- 옵션 관련 모델

**예상 코드량**: ~300줄

**완료 기준**: `ticker.OptionChain()`, `ticker.ExpirationDates()` 동작

---

### Phase 3: Financials (재무제표) - 권장

기본적 분석을 위한 재무 데이터

**구현 항목**:
- Income Statement (손익계산서)
- Balance Sheet (재무상태표)
- Cash Flow (현금흐름표)
- Annual / Quarterly 지원

**예상 코드량**: ~500줄

**완료 기준**: `ticker.IncomeStatement()`, `ticker.BalanceSheet()`, `ticker.CashFlow()` 동작

---

### Phase 4: Analysis (분석 데이터) - 권장

애널리스트 및 예측 데이터

**구현 항목**:
- Analyst Recommendations
- Price Targets
- Earnings Estimates
- Revenue Estimates
- EPS Trend / Revisions
- Growth Estimates

**예상 코드량**: ~400줄

**완료 기준**: `ticker.Recommendations()`, `ticker.EarningsEstimate()` 등 동작

---

### Phase 5: Holdings & Actions (보유/이벤트) - 선택

주주 정보 및 기업 이벤트

**구현 항목**:
- Institutional Holders
- Mutual Fund Holders
- Insider Transactions
- Dividends
- Stock Splits
- Calendar Events

**예상 코드량**: ~500줄

**완료 기준**: `ticker.Holders()`, `ticker.Dividends()`, `ticker.Splits()` 동작

---

### Phase 6: Search & Screener (검색/스크리닝) - 선택

종목 탐색 기능

**구현 항목**:
- Symbol Search
- Symbol Lookup
- Stock Screener
- Predefined Screens
- Custom Query Builder

**예상 코드량**: ~400줄

**완료 기준**: `Search()`, `Screen()` 동작

---

### Phase 7: Multi-ticker & Batch (대량 처리) - 선택

여러 종목 동시 처리

**구현 항목**:
- Tickers (복수 종목)
- Download (배치 다운로드)
- Goroutine 병렬 처리
- Rate Limiting
- Progress Tracking

**예상 코드량**: ~300줄

**완료 기준**: `Download([]string{"AAPL", "GOOGL", "MSFT"})` 병렬 동작

---

### Phase 8: Real-time (실시간) - 고급 (Optional)

WebSocket 기반 실시간 스트리밍

**구현 항목**:
- WebSocket Client
- Protobuf Decoder
- Subscribe/Unsubscribe
- Callback Handler
- Reconnection Logic

**예상 코드량**: ~400줄

**완료 기준**: 실시간 가격 스트리밍 동작

---

### Phase 9: Advanced (고급) - 고급 (Optional)

고급 기능 및 최적화

**구현 항목**:
- Caching Layer (메모리/파일)
- Price Repair (데이터 정합성)
- Timezone Handling
- Market Calendar
- News Feed

**예상 코드량**: ~600줄

---

## Summary

### 코드량 예상

| Phase | 이름 | 우선순위 | 예상 코드량 | 누적 |
|-------|------|---------|-----------|------|
| 0 | Foundation | 필수 | ~500줄 | 500줄 |
| 1 | Core Data | 필수 | ~600줄 | 1,100줄 |
| 2 | Options | 권장 | ~300줄 | 1,400줄 |
| 3 | Financials | 권장 | ~500줄 | 1,900줄 |
| 4 | Analysis | 권장 | ~400줄 | 2,300줄 |
| 5 | Holdings | 선택 | ~500줄 | 2,800줄 |
| 6 | Search | 선택 | ~400줄 | 3,200줄 |
| 7 | Multi-ticker | 선택 | ~300줄 | 3,500줄 |
| 8 | Real-time | 고급 | ~400줄 | 3,900줄 |
| 9 | Advanced | 고급 | ~600줄 | 4,500줄 |

### 목표 수준

```
MVP (최소 실행 가능 제품)
  Phase 0 + Phase 1 = ~1,100줄
  → 기본적인 가격 조회 가능

실용적 수준
  Phase 0~4 = ~2,300줄
  → 대부분의 투자 분석 가능

Python yfinance 동등
  Phase 0~7 = ~3,500줄
  → 거의 모든 기능 사용 가능
```

---

## Dependencies

### Required

```
github.com/Danny-Dasilva/CycleTLS/cycletls  # TLS Fingerprint 위장
```

### Optional (Phase별)

```
google.golang.org/protobuf                   # Phase 8: WebSocket Protobuf
github.com/gorilla/websocket                 # Phase 8: WebSocket
```

---

## References

- [Python yfinance](https://github.com/ranaroussi/yfinance)
- [CycleTLS](https://github.com/Danny-Dasilva/CycleTLS)
- [utls](https://github.com/refraction-networking/utls)
- [Yahoo Finance API (Unofficial)](https://query2.finance.yahoo.com)
