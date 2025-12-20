# go-yfinance Development Plan

Phase 0-4가 완료되었으며, Phase 5-9의 상세 개발 계획을 정리합니다.

---

## Phase 5: Holdings & Actions

**목표**: 기관/펀드 보유자, 내부자 거래, 캘린더 이벤트 기능 구현

### 5.1 Holdings (보유자 정보)

#### API 정보
- **Endpoint**: `query2.finance.yahoo.com/v10/finance/quoteSummary/{symbol}`
- **Modules**:
  - `institutionOwnership`: 기관 투자자
  - `fundOwnership`: 뮤추얼 펀드
  - `majorDirectHolders`: 주요 직접 보유자
  - `majorHoldersBreakdown`: 보유자 비율 분석
  - `insiderTransactions`: 내부자 거래
  - `insiderHolders`: 내부자 보유 현황
  - `netSharePurchaseActivity`: 순매수/매도 활동

#### 구현 항목

| 메소드 | 설명 | 반환 타입 |
|--------|------|-----------|
| `MajorHolders()` | 보유자 비율 분석 | `*models.MajorHolders` |
| `InstitutionalHolders()` | 기관 투자자 목록 | `[]models.Holder` |
| `MutualFundHolders()` | 뮤추얼 펀드 목록 | `[]models.Holder` |
| `InsiderTransactions()` | 내부자 거래 내역 | `[]models.InsiderTransaction` |
| `InsiderPurchases()` | 내부자 매수/매도 요약 | `*models.InsiderPurchases` |
| `InsiderRoster()` | 내부자 명단 | `[]models.InsiderHolder` |

#### 모델 설계

```go
// pkg/models/holders.go

type MajorHolders struct {
    InsidersPercentHeld        float64 `json:"insidersPercentHeld"`
    InstitutionsPercentHeld    float64 `json:"institutionsPercentHeld"`
    InstitutionsFloatPercentHeld float64 `json:"institutionsFloatPercentHeld"`
    InstitutionsCount          int     `json:"institutionsCount"`
}

type Holder struct {
    DateReported time.Time `json:"dateReported"`
    Holder       string    `json:"holder"`
    Shares       int64     `json:"shares"`
    Value        float64   `json:"value"`
    PctHeld      float64   `json:"pctHeld"`
    PctChange    float64   `json:"pctChange"`
}

type InsiderTransaction struct {
    StartDate   time.Time `json:"startDate"`
    Insider     string    `json:"insider"`
    Position    string    `json:"position"`
    URL         string    `json:"url"`
    Transaction string    `json:"transaction"`
    Text        string    `json:"text"`
    Shares      int64     `json:"shares"`
    Value       float64   `json:"value"`
    Ownership   string    `json:"ownership"`
}

type InsiderHolder struct {
    Name                  string    `json:"name"`
    Position              string    `json:"position"`
    URL                   string    `json:"url"`
    MostRecentTransaction string    `json:"mostRecentTransaction"`
    LatestTransDate       time.Time `json:"latestTransDate"`
    PositionDirectDate    time.Time `json:"positionDirectDate"`
    SharesOwnedDirectly   int64     `json:"sharesOwnedDirectly"`
    SharesOwnedIndirectly int64     `json:"sharesOwnedIndirectly"`
}

type InsiderPurchases struct {
    Period                    string  `json:"period"`
    BuyShares                 int64   `json:"buyShares"`
    BuyTransactions           int     `json:"buyTransactions"`
    SellShares                int64   `json:"sellShares"`
    SellTransactions          int     `json:"sellTransactions"`
    NetShares                 int64   `json:"netShares"`
    NetTransactions           int     `json:"netTransactions"`
    TotalInsiderShares        int64   `json:"totalInsiderShares"`
    NetPercentInsiderShares   float64 `json:"netPercentInsiderShares"`
}
```

### 5.2 Calendar Events (캘린더 이벤트)

#### API 정보
- **Module**: `calendarEvents`

#### 구현 항목

| 메소드 | 설명 | 반환 타입 |
|--------|------|-----------|
| `Calendar()` | 예정된 이벤트 (배당, 실적발표 등) | `*models.Calendar` |

#### 모델 설계

```go
// pkg/models/calendar.go

type Calendar struct {
    DividendDate   *time.Time `json:"dividendDate,omitempty"`
    ExDividendDate *time.Time `json:"exDividendDate,omitempty"`
    EarningsDate   []time.Time `json:"earningsDate,omitempty"`
    EarningsHigh   *float64   `json:"earningsHigh,omitempty"`
    EarningsLow    *float64   `json:"earningsLow,omitempty"`
    EarningsAverage *float64  `json:"earningsAverage,omitempty"`
    RevenueHigh    *float64   `json:"revenueHigh,omitempty"`
    RevenueLow     *float64   `json:"revenueLow,omitempty"`
    RevenueAverage *float64   `json:"revenueAverage,omitempty"`
}
```

### 5.3 SEC Filings (SEC 보고서) - 선택사항

#### API 정보
- **Module**: `secFilings`

#### 구현 항목

| 메소드 | 설명 | 반환 타입 |
|--------|------|-----------|
| `SECFilings()` | SEC 제출 보고서 목록 | `[]models.SECFiling` |

---

## Phase 6: Search & Lookup

**목표**: 심볼 검색 및 조회 기능 구현

### 6.1 Search (검색)

#### API 정보
- **Endpoint**: `query2.finance.yahoo.com/v1/finance/search`
- **Parameters**:
  - `q`: 검색어
  - `quotesCount`: 결과 수
  - `newsCount`: 뉴스 수
  - `listsCount`: 리스트 수
  - `enableFuzzyQuery`: 오타 허용
  - `quotesQueryId`: `tss_match_phrase_query`
  - `newsQueryId`: `news_cie_vespa`

#### 구현 항목

```go
// pkg/search/search.go

type Search struct {
    query   string
    data    *client.Client
    options SearchOptions
}

type SearchOptions struct {
    MaxResults       int  // default 8
    NewsCount        int  // default 8
    ListsCount       int  // default 8
    EnableFuzzyQuery bool // default false
    IncludeResearch  bool // default false
}

type SearchResult struct {
    Quotes   []QuoteResult   `json:"quotes"`
    News     []NewsResult    `json:"news"`
    Lists    []ListResult    `json:"lists"`
    Research []ResearchResult `json:"research,omitempty"`
}

func New(query string, opts ...SearchOptions) *Search
func (s *Search) Search() (*SearchResult, error)
```

#### 사용 예시

```go
search := search.New("Apple", search.SearchOptions{MaxResults: 10})
results, err := search.Search()

for _, q := range results.Quotes {
    fmt.Printf("%s: %s (%s)\n", q.Symbol, q.ShortName, q.QuoteType)
}
```

### 6.2 Lookup (심볼 조회)

#### API 정보
- **Endpoint**: `query1.finance.yahoo.com/v1/finance/lookup`
- **Parameters**:
  - `query`: 검색어
  - `type`: all, equity, mutualfund, etf, index, future, currency, cryptocurrency
  - `count`: 결과 수

#### 구현 항목

```go
// pkg/lookup/lookup.go

type Lookup struct {
    query string
    data  *client.Client
}

type LookupType string

const (
    LookupAll          LookupType = "all"
    LookupEquity       LookupType = "equity"
    LookupMutualFund   LookupType = "mutualfund"
    LookupETF          LookupType = "etf"
    LookupIndex        LookupType = "index"
    LookupFuture       LookupType = "future"
    LookupCurrency     LookupType = "currency"
    LookupCrypto       LookupType = "cryptocurrency"
)

func New(query string) *Lookup
func (l *Lookup) All(count int) ([]LookupResult, error)
func (l *Lookup) Stock(count int) ([]LookupResult, error)
func (l *Lookup) ETF(count int) ([]LookupResult, error)
func (l *Lookup) Index(count int) ([]LookupResult, error)
func (l *Lookup) Crypto(count int) ([]LookupResult, error)
// ... 기타 타입별 메소드
```

---

## Phase 7: Multi-ticker & Batch

**목표**: 다중 종목 처리 및 배치 다운로드 기능 구현

### 7.1 Tickers (다중 종목)

#### 구현 항목

```go
// pkg/tickers/tickers.go

type Tickers struct {
    symbols []string
    tickers map[string]*ticker.Ticker
    data    *client.Client
}

func New(symbols ...string) (*Tickers, error)
func (t *Tickers) History(params models.HistoryParams) (map[string]*models.History, error)
func (t *Tickers) Download(params DownloadParams) (*DataFrame, error)
```

### 7.2 Download (배치 다운로드)

#### 구현 항목

```go
// pkg/download/download.go

type DownloadParams struct {
    Symbols     []string
    Period      string
    Interval    string
    Start       *time.Time
    End         *time.Time
    PrePost     bool
    Actions     bool
    AutoAdjust  bool
    Repair      bool
    Threads     int  // goroutine 수, 0=자동
    Progress    bool
    Timeout     time.Duration
}

type DownloadResult struct {
    Data   map[string]*models.History
    Errors map[string]error
}

func Download(params DownloadParams) (*DownloadResult, error)
```

### 7.3 Parallel Processing (병렬 처리)

#### 구현 전략

```go
// 내부 구현 - worker pool 패턴

type worker struct {
    id      int
    jobs    <-chan string
    results chan<- *tickerResult
}

type tickerResult struct {
    Symbol string
    Data   *models.History
    Err    error
}

func (d *Downloader) downloadParallel(symbols []string, params HistoryParams) (*DownloadResult, error) {
    numWorkers := d.params.Threads
    if numWorkers == 0 {
        numWorkers = min(len(symbols), runtime.NumCPU()*2)
    }

    jobs := make(chan string, len(symbols))
    results := make(chan *tickerResult, len(symbols))

    // Start workers
    for w := 0; w < numWorkers; w++ {
        go worker(w, jobs, results)
    }

    // Send jobs
    for _, symbol := range symbols {
        jobs <- symbol
    }
    close(jobs)

    // Collect results
    result := &DownloadResult{
        Data:   make(map[string]*models.History),
        Errors: make(map[string]error),
    }
    for i := 0; i < len(symbols); i++ {
        r := <-results
        if r.Err != nil {
            result.Errors[r.Symbol] = r.Err
        } else {
            result.Data[r.Symbol] = r.Data
        }
    }

    return result, nil
}
```

### 7.4 Rate Limiting (속도 제한)

```go
// pkg/client/ratelimit.go

type RateLimiter struct {
    ticker   *time.Ticker
    requests chan struct{}
}

func NewRateLimiter(rps float64) *RateLimiter
func (r *RateLimiter) Wait()
func (r *RateLimiter) Stop()
```

---

## Phase 8: Real-time WebSocket

**목표**: 실시간 가격 데이터 스트리밍 구현

### 8.1 WebSocket Client

#### API 정보
- **Endpoint**: `wss://streamer.finance.yahoo.com/?version=2`
- **Protocol**: WebSocket + Protobuf
- **Message Format**: Base64 encoded Protobuf

#### 핵심 구현

```go
// pkg/live/websocket.go

type WebSocket struct {
    url            string
    conn           *websocket.Conn
    subscriptions  map[string]bool
    messageHandler MessageHandler
    mu             sync.RWMutex
    done           chan struct{}
}

type MessageHandler func(data *PricingData)

type PricingData struct {
    ID              string  `json:"id"`
    Price           float32 `json:"price"`
    Time            int64   `json:"time"`
    Currency        string  `json:"currency"`
    Exchange        string  `json:"exchange"`
    QuoteType       int32   `json:"quoteType"`
    MarketHours     int32   `json:"marketHours"`
    ChangePercent   float32 `json:"changePercent"`
    DayVolume       int64   `json:"dayVolume"`
    DayHigh         float32 `json:"dayHigh"`
    DayLow          float32 `json:"dayLow"`
    Change          float32 `json:"change"`
    ShortName       string  `json:"shortName"`
    OpenPrice       float32 `json:"openPrice"`
    PreviousClose   float32 `json:"previousClose"`
    Bid             float32 `json:"bid"`
    BidSize         int64   `json:"bidSize"`
    Ask             float32 `json:"ask"`
    AskSize         int64   `json:"askSize"`
    MarketCap       float64 `json:"marketCap"`
    // ... 옵션 관련 필드
}

func NewWebSocket() *WebSocket
func (ws *WebSocket) Subscribe(symbols ...string) error
func (ws *WebSocket) Unsubscribe(symbols ...string) error
func (ws *WebSocket) Listen(handler MessageHandler) error
func (ws *WebSocket) Close() error
```

### 8.2 Protobuf Decoder

#### Protobuf 스키마 (pricing.proto)

```protobuf
syntax = "proto3";

message PricingData {
    string id = 1;
    float price = 2;
    sint64 time = 3;
    string currency = 4;
    string exchange = 5;
    int32 quote_type = 6;
    int32 market_hours = 7;
    float change_percent = 8;
    sint64 day_volume = 9;
    float day_high = 10;
    float day_low = 11;
    float change = 12;
    string short_name = 13;
    sint64 expire_date = 14;
    float open_price = 15;
    float previous_close = 16;
    float strike_price = 17;
    string underlying_symbol = 18;
    sint64 open_interest = 19;
    sint64 options_type = 20;
    sint64 mini_option = 21;
    sint64 last_size = 22;
    float bid = 23;
    sint64 bid_size = 24;
    float ask = 25;
    sint64 ask_size = 26;
    sint64 price_hint = 27;
    sint64 vol_24hr = 28;
    sint64 vol_all_currencies = 29;
    string from_currency = 30;
    string last_market = 31;
    double circulating_supply = 32;
    double market_cap = 33;
}
```

### 8.3 AsyncWebSocket (비동기)

```go
// pkg/live/async.go

type AsyncWebSocket struct {
    *WebSocket
    ctx    context.Context
    cancel context.CancelFunc
}

func NewAsyncWebSocket(ctx context.Context) *AsyncWebSocket
func (aws *AsyncWebSocket) Start() error
func (aws *AsyncWebSocket) Stop()
```

### 8.4 사용 예시

```go
// 동기 방식
ws := live.NewWebSocket()
defer ws.Close()

ws.Subscribe("AAPL", "GOOGL", "MSFT")
ws.Listen(func(data *live.PricingData) {
    fmt.Printf("%s: $%.2f (%.2f%%)\n",
        data.ID, data.Price, data.ChangePercent)
})

// 비동기 방식
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

aws := live.NewAsyncWebSocket(ctx)
aws.Subscribe("AAPL")
go aws.Start()

// 다른 작업 수행...
```

---

## Phase 9: Advanced Features

**목표**: 캐싱, 데이터 정합성, 타임존, 마켓 캘린더, 뉴스 피드 구현

### 9.1 Caching Layer (캐싱)

#### 구현 항목

```go
// pkg/cache/cache.go

type Cache interface {
    Get(key string) ([]byte, bool)
    Set(key string, value []byte, ttl time.Duration)
    Delete(key string)
    Clear()
}

// 메모리 캐시
type MemoryCache struct {
    data map[string]*cacheEntry
    mu   sync.RWMutex
}

// 파일 캐시 (SQLite)
type FileCache struct {
    db   *sql.DB
    path string
}

// 캐시 키 패턴
// ticker:{symbol}:quote
// ticker:{symbol}:history:{period}:{interval}
// ticker:{symbol}:info
// tz:{symbol}
```

### 9.2 Price Repair (가격 데이터 보정)

Python yfinance의 `repair` 기능 포팅:
- 100x 환율 오류 감지 및 수정
- 스플릿 미반영 데이터 보정
- 이상치 감지

```go
// pkg/repair/repair.go

type Repairer struct {
    history *models.History
}

func New(history *models.History) *Repairer
func (r *Repairer) DetectCurrencyMixups() []RepairSuggestion
func (r *Repairer) DetectSplitErrors() []RepairSuggestion
func (r *Repairer) Repair() (*models.History, error)
```

### 9.3 Timezone Handling (타임존)

```go
// pkg/timezone/timezone.go

type TzCache struct {
    db *sql.DB
}

func GetTickerTimezone(symbol string) (string, error)
func CacheTimezone(symbol, tz string) error
func ValidateTimezone(tz string) bool

// 자동 타임존 감지
func (t *Ticker) GetTimezone() string {
    if t.tz != "" {
        return t.tz
    }

    // 1. 캐시 확인
    if tz, ok := tzCache.Get(t.Symbol); ok {
        t.tz = tz
        return tz
    }

    // 2. API에서 가져오기
    md := t.getHistoryMetadata()
    t.tz = md.ExchangeTimezoneName
    tzCache.Set(t.Symbol, t.tz)

    return t.tz
}
```

### 9.4 Market Calendar (마켓 캘린더)

```go
// pkg/market/calendar.go

type MarketCalendar struct {
    Exchange string
}

type TradingDay struct {
    Date        time.Time
    Open        time.Time
    Close       time.Time
    IsHalfDay   bool
    IsHoliday   bool
}

func NewCalendar(exchange string) *MarketCalendar
func (c *MarketCalendar) IsOpen(t time.Time) bool
func (c *MarketCalendar) NextOpen(t time.Time) time.Time
func (c *MarketCalendar) TradingDays(start, end time.Time) []TradingDay
func (c *MarketCalendar) Holidays(year int) []time.Time
```

### 9.5 News Feed (뉴스 피드)

#### API 정보
- **Endpoint**: `finance.yahoo.com/xhr/ncp`
- **Parameters**:
  - `queryRef`: `latestNews`, `newsAll`, `pressRelease`
  - `serviceKey`: `ncp_fin`

```go
// pkg/news/news.go

type NewsItem struct {
    UUID            string    `json:"uuid"`
    Title           string    `json:"title"`
    Publisher       string    `json:"publisher"`
    Link            string    `json:"link"`
    ProviderPublishTime time.Time `json:"providerPublishTime"`
    Type            string    `json:"type"`
    Thumbnail       *Thumbnail `json:"thumbnail,omitempty"`
    RelatedTickers  []string  `json:"relatedTickers"`
}

func (t *Ticker) News(opts NewsOptions) ([]NewsItem, error)
```

---

## Implementation Priority & Dependencies

### Phase 의존성

```
Phase 5 (Holdings) ─────────────────────────┐
                                            │
Phase 6 (Search) ──────────────────────────┼──→ 독립적 구현 가능
                                            │
Phase 7 (Multi-ticker) ─────────────────────┤
          ↓                                 │
Phase 8 (WebSocket) ────────────────────────┤
                                            │
Phase 9 (Advanced) ─────────────────────────┘
    - Caching: 모든 Phase에서 사용 가능
    - Timezone: Phase 7-8에서 중요
    - News: Phase 5와 함께 구현 가능
```

### 추천 구현 순서

1. **Phase 5**: Holdings & Calendar (기존 패턴 활용, 난이도 낮음)
2. **Phase 9.1**: Caching (이후 Phase 성능 개선)
3. **Phase 6**: Search & Lookup (독립적, 새 패키지)
4. **Phase 7**: Multi-ticker (Phase 1-4 기반 확장)
5. **Phase 8**: WebSocket (새로운 프로토콜, Protobuf)
6. **Phase 9 나머지**: 점진적 개선

---

## 예상 파일 구조

```
go-yfinance/
├── pkg/
│   ├── ticker/
│   │   ├── holders.go      # Phase 5
│   │   ├── calendar.go     # Phase 5
│   │   └── news.go         # Phase 9
│   ├── models/
│   │   ├── holders.go      # Phase 5
│   │   ├── calendar.go     # Phase 5
│   │   ├── search.go       # Phase 6
│   │   └── live.go         # Phase 8
│   ├── search/             # Phase 6
│   │   ├── search.go
│   │   └── lookup.go
│   ├── tickers/            # Phase 7
│   │   └── tickers.go
│   ├── download/           # Phase 7
│   │   └── download.go
│   ├── live/               # Phase 8
│   │   ├── websocket.go
│   │   ├── async.go
│   │   └── pricing.pb.go
│   ├── cache/              # Phase 9
│   │   ├── cache.go
│   │   ├── memory.go
│   │   └── file.go
│   └── market/             # Phase 9
│       └── calendar.go
└── internal/
    └── proto/              # Phase 8
        └── pricing.proto
```

---

## 테스트 전략

각 Phase별 테스트 파일 작성:
- Unit tests: 각 함수/메소드 단위 테스트
- Integration tests: 실제 API 호출 테스트 (선택적)
- Mock tests: HTTP 응답 모킹

```go
// Example test structure
func TestMajorHolders(t *testing.T) {
    // Mock HTTP response
    // Test parsing
    // Verify data structure
}
```

---

## 성능 목표

| 기능 | 목표 | 비고 |
|------|------|------|
| 단일 Quote | < 500ms | 현재 달성 |
| 배치 다운로드 (10종목) | < 3s | 병렬 처리 |
| 배치 다운로드 (100종목) | < 15s | Rate limiting 고려 |
| WebSocket 지연 | < 100ms | 실시간 스트리밍 |
| 캐시 히트 응답 | < 10ms | 메모리 캐시 |

---

## 다음 단계

Phase 5부터 순차적으로 구현을 시작합니다. 각 Phase 완료 후:
1. STATUS.md 업데이트
2. 테스트 작성 및 실행
3. API 문서 자동 생성 (gomarkdoc)
4. main 브랜치에 머지
