package live

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

func TestNew(t *testing.T) {
	ws, err := New()
	if err != nil {
		t.Fatalf("Failed to create WebSocket: %v", err)
	}

	if ws.url != DefaultURL {
		t.Errorf("Expected URL %s, got %s", DefaultURL, ws.url)
	}

	if ws.heartbeatInterval != DefaultHeartbeatInterval {
		t.Errorf("Expected heartbeat interval %v, got %v", DefaultHeartbeatInterval, ws.heartbeatInterval)
	}
}

func TestNewWithOptions(t *testing.T) {
	customURL := "wss://custom.example.com"
	customInterval := 30 * time.Second

	ws, err := New(
		WithURL(customURL),
		WithHeartbeatInterval(customInterval),
	)
	if err != nil {
		t.Fatalf("Failed to create WebSocket: %v", err)
	}

	if ws.url != customURL {
		t.Errorf("Expected URL %s, got %s", customURL, ws.url)
	}

	if ws.heartbeatInterval != customInterval {
		t.Errorf("Expected heartbeat interval %v, got %v", customInterval, ws.heartbeatInterval)
	}
}

func TestNewRejectsNonPositiveHeartbeatInterval(t *testing.T) {
	if _, err := New(WithHeartbeatInterval(0)); err == nil {
		t.Fatal("New() unexpectedly accepted zero heartbeat interval")
	}
}

func TestSubscriptions(t *testing.T) {
	ws, _ := New()

	// Initially empty
	if len(ws.Subscriptions()) != 0 {
		t.Error("Expected empty subscriptions initially")
	}

	// Add to subscriptions map directly (without connecting)
	ws.subscriptions["AAPL"] = struct{}{}
	ws.subscriptions["MSFT"] = struct{}{}

	subs := ws.Subscriptions()
	if len(subs) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subs))
	}
}

func TestIsConnected(t *testing.T) {
	ws, _ := New()

	if ws.IsConnected() {
		t.Error("Expected not connected initially")
	}
}

func TestPricingDataTimestamp(t *testing.T) {
	now := time.Now().Unix()
	pd := &models.PricingData{
		Time: now,
	}

	ts := pd.Timestamp()
	if ts.Unix() != now {
		t.Errorf("Expected timestamp %d, got %d", now, ts.Unix())
	}
}

func TestPricingDataMarketState(t *testing.T) {
	tests := []struct {
		marketHours int32
		isRegular   bool
		isPre       bool
		isPost      bool
	}{
		{0, false, true, false},
		{1, true, false, false},
		{2, false, false, true},
		{3, false, false, false},
	}

	for _, tt := range tests {
		pd := &models.PricingData{MarketHours: tt.marketHours}

		if pd.IsRegularMarket() != tt.isRegular {
			t.Errorf("MarketHours %d: IsRegularMarket expected %v", tt.marketHours, tt.isRegular)
		}
		if pd.IsPreMarket() != tt.isPre {
			t.Errorf("MarketHours %d: IsPreMarket expected %v", tt.marketHours, tt.isPre)
		}
		if pd.IsPostMarket() != tt.isPost {
			t.Errorf("MarketHours %d: IsPostMarket expected %v", tt.marketHours, tt.isPost)
		}
	}
}

func TestMarketStateString(t *testing.T) {
	tests := []struct {
		state    models.MarketState
		expected string
	}{
		{models.MarketStatePreMarket, "PRE_MARKET"},
		{models.MarketStateRegular, "REGULAR"},
		{models.MarketStatePostMarket, "POST_MARKET"},
		{models.MarketStateClosed, "CLOSED"},
		{models.MarketState(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("MarketState(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}

// TestDecodeProtobuf tests the protobuf decoder with a manually crafted message.
func TestDecodeProtobuf(t *testing.T) {
	// Manually encode a simple protobuf message:
	// field 1 (id): "AAPL"
	// field 2 (price): 150.25
	buf := make([]byte, 0, 50)

	// Field 1: id = "AAPL" (wire type 2 = length-delimited)
	buf = append(buf, (1<<3)|2)  // tag = (1 << 3) | 2
	buf = append(buf, 4)         // length = 4
	buf = append(buf, "AAPL"...) // value

	// Field 2: price = 150.25 (wire type 5 = 32-bit)
	buf = append(buf, (2<<3)|5) // tag = (2 << 3) | 5
	priceBits := math.Float32bits(150.25)
	priceBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(priceBytes, priceBits)
	buf = append(buf, priceBytes...)

	// Field 4: currency = "USD" (wire type 2)
	buf = append(buf, (4<<3)|2)
	buf = append(buf, 3)
	buf = append(buf, "USD"...)

	// Field 7: market_hours = 1 (wire type 0 = varint)
	buf = append(buf, 7<<3)
	buf = append(buf, 1)

	pd, err := decodeProtobuf(buf)
	if err != nil {
		t.Fatalf("Failed to decode protobuf: %v", err)
	}

	if pd.ID != "AAPL" {
		t.Errorf("Expected ID 'AAPL', got '%s'", pd.ID)
	}

	if pd.Price != 150.25 {
		t.Errorf("Expected price 150.25, got %f", pd.Price)
	}

	if pd.Currency != "USD" {
		t.Errorf("Expected currency 'USD', got '%s'", pd.Currency)
	}

	if pd.MarketHours != 1 {
		t.Errorf("Expected market_hours 1, got %d", pd.MarketHours)
	}
}

// TestDecodeBase64Message tests the full decode pipeline.
func TestDecodeBase64Message(t *testing.T) {
	// Build a protobuf message
	buf := make([]byte, 0, 50)

	// Field 1: id = "MSFT"
	buf = append(buf, (1<<3)|2)
	buf = append(buf, 4)
	buf = append(buf, "MSFT"...)

	// Field 2: price = 420.50
	buf = append(buf, (2<<3)|5)
	priceBits := math.Float32bits(420.50)
	priceBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(priceBytes, priceBits)
	buf = append(buf, priceBytes...)

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(buf)

	pd, err := decodeBase64Message(encoded)
	if err != nil {
		t.Fatalf("Failed to decode base64 message: %v", err)
	}

	if pd.ID != "MSFT" {
		t.Errorf("Expected ID 'MSFT', got '%s'", pd.ID)
	}

	if pd.Price != 420.50 {
		t.Errorf("Expected price 420.50, got %f", pd.Price)
	}
}

func TestDecodeBase64MessageInvalid(t *testing.T) {
	_, err := decodeBase64Message("not-valid-base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64")
	}
}

func TestDecodeProtobufRejectsKnownFieldWithWrongWireType(t *testing.T) {
	// Field 2 (price) must be fixed32/wire type 5, not length-delimited/type 2.
	_, err := decodeProtobuf([]byte{(2 << 3) | 2, 4, 0, 0, 0, 0})
	if err == nil || !strings.Contains(err.Error(), "wire type") {
		t.Fatalf("decodeProtobuf() error = %v, want wire-type error", err)
	}
}

// TestProtoReaderVarint tests varint reading.
func TestProtoReaderVarint(t *testing.T) {
	tests := []struct {
		data     []byte
		expected uint64
	}{
		{[]byte{0x00}, 0},
		{[]byte{0x01}, 1},
		{[]byte{0x7F}, 127},
		{[]byte{0x80, 0x01}, 128},
		{[]byte{0xAC, 0x02}, 300},
	}

	for _, tt := range tests {
		r := &protoReader{data: tt.data, pos: 0}
		got, err := r.readVarint()
		if err != nil {
			t.Errorf("readVarint(%v) error: %v", tt.data, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("readVarint(%v) = %d, want %d", tt.data, got, tt.expected)
		}
	}
}

func TestProtoReaderRejectsVarintOverflow(t *testing.T) {
	tests := [][]byte{
		{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02},
		{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00},
	}
	for _, data := range tests {
		r := &protoReader{data: data}
		if _, err := r.readVarint(); err == nil {
			t.Fatalf("readVarint(%x) unexpectedly succeeded", data)
		}
	}
}

func TestProtoReaderRejectsOversizedLengthWithoutPanic(t *testing.T) {
	// MaxUint64 encoded as a protobuf varint. Converting this length to int
	// before checking the remaining buffer used to permit integer overflow.
	length := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}

	t.Run("read string", func(t *testing.T) {
		r := &protoReader{data: append(append([]byte(nil), length...), 'x')}
		if _, err := r.readString(); err == nil {
			t.Fatal("readString unexpectedly accepted oversized length")
		}
	})

	t.Run("skip field", func(t *testing.T) {
		r := &protoReader{data: append(append([]byte(nil), length...), 'x')}
		if err := r.skipField(2); err == nil {
			t.Fatal("skipField unexpectedly accepted oversized length")
		}
	})
}

func TestProtoReaderAcceptsMaximumUint64Varint(t *testing.T) {
	data := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}
	r := &protoReader{data: data}
	got, err := r.readVarint()
	if err != nil {
		t.Fatalf("readVarint() error: %v", err)
	}
	if got != math.MaxUint64 {
		t.Fatalf("readVarint() = %d, want %d", got, uint64(math.MaxUint64))
	}
}

// TestProtoReaderSint64 tests zigzag decoding.
func TestProtoReaderSint64(t *testing.T) {
	tests := []struct {
		encoded  uint64
		expected int64
	}{
		{0, 0},
		{1, -1},
		{2, 1},
		{3, -2},
		{4, 2},
	}

	for _, tt := range tests {
		// Encode as varint
		buf := make([]byte, 0, 10)
		v := tt.encoded
		for v >= 0x80 {
			buf = append(buf, byte(v)|0x80)
			v >>= 7
		}
		buf = append(buf, byte(v))

		r := &protoReader{data: buf, pos: 0}
		got, err := r.readSint64()
		if err != nil {
			t.Errorf("readSint64 error: %v", err)
			continue
		}
		if got != tt.expected {
			t.Errorf("readSint64 (encoded=%d) = %d, want %d", tt.encoded, got, tt.expected)
		}
	}
}

// Integration tests (require network access)
// These are skipped by default

// func TestWebSocketIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test")
// 	}
//
// 	ws, err := New()
// 	if err != nil {
// 		t.Fatalf("Failed to create WebSocket: %v", err)
// 	}
// 	defer ws.Close()
//
// 	if err := ws.Subscribe([]string{"AAPL"}); err != nil {
// 		t.Fatalf("Failed to subscribe: %v", err)
// 	}
//
// 	received := make(chan *models.PricingData, 1)
// 	go func() {
// 		ws.Listen(func(data *models.PricingData) {
// 			received <- data
// 		})
// 	}()
//
// 	select {
// 	case data := <-received:
// 		t.Logf("Received: %s $%.2f", data.ID, data.Price)
// 	case <-time.After(10 * time.Second):
// 		t.Error("Timeout waiting for message")
// 	}
// }

// TestSendSubscribeNilConn verifies that sendSubscribe returns an error
// instead of panicking when ws.conn is nil (e.g. during reconnect window).
// Reproduces: rodherz/go-yfinance#1
func TestSendSubscribeNilConn(t *testing.T) {
	ws, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	// Populate subscriptions without connecting, so conn stays nil.
	ws.subscriptions["AAPL"] = struct{}{}

	// Without the fix this panics with a nil pointer dereference.
	// The recover() turns the panic into a test failure with a clear message.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("sendSubscribe panicked with nil conn: %v", r)
		}
	}()

	err = ws.sendSubscribe([]string{"AAPL"})
	if err == nil {
		t.Fatal("expected error when conn is nil, got nil")
	}
}

// upgrader is the gorilla upgrader used by the test echo server.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// newTestWSServer starts an httptest WebSocket server that accepts and discards
// all incoming messages. It returns the server and the ws:// URL to connect to.
func newTestWSServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	return srv, url
}

// TestSendSubscribeConcurrent verifies that concurrent calls to sendSubscribe
// do not panic with "concurrent write to websocket connection".
// Without the writeMu fix, the -race detector flags a data race and gorilla
// panics when two goroutines call conn.WriteMessage simultaneously.
// Reproduces: rodherz/go-yfinance#2
func TestSendSubscribeConcurrent(t *testing.T) {
	srv, url := newTestWSServer(t)
	defer srv.Close()

	ws, err := New(WithURL(url), WithHeartbeatInterval(100*time.Millisecond))
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if err := ws.Connect(); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	defer func() { _ = ws.Close() }()

	const goroutines = 5
	const callsEach = 10

	// Catch panics from any goroutine so the test fails cleanly instead of
	// crashing the process.
	panicCh := make(chan any, goroutines)

	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCh <- r
				}
			}()
			sym := []string{"AAPL", "MSFT", "GOOGL"}[id%3]
			for range callsEach {
				if err := ws.sendSubscribe([]string{sym}); err != nil {
					// "not connected" during a reconnect window is acceptable.
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(panicCh)

	for p := range panicCh {
		t.Errorf("sendSubscribe panicked: %v", p)
	}
}

func TestListenCloseIsNormalAndIdempotent(t *testing.T) {
	srv, url := newTestWSServer(t)
	defer srv.Close()

	var handlerCalls atomic.Int32
	ws, err := New(
		WithURL(url),
		WithErrorHandler(func(error) { handlerCalls.Add(1) }),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	listenDone := make(chan error, 1)
	go func() { listenDone <- ws.Listen(nil) }()
	waitForWebSocketState(t, ws, func(ws *WebSocket) bool { return ws.isListening })

	if err := ws.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
	if err := ws.Close(); err != nil {
		t.Fatalf("second Close() error: %v", err)
	}

	select {
	case err := <-listenDone:
		if err != nil {
			t.Fatalf("Listen() after normal Close() = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Listen() did not stop after Close()")
	}
	if got := handlerCalls.Load(); got != 0 {
		t.Fatalf("error handler called %d times during normal close", got)
	}
	if err := ws.Connect(); !errors.Is(err, errWebSocketClosed) {
		t.Fatalf("Connect() after Close() = %v, want errWebSocketClosed", err)
	}
}

func TestCloseCancelsReconnect(t *testing.T) {
	var connections atomic.Int32
	closedFirst := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		connections.Add(1)
		_ = conn.Close()
		select {
		case <-closedFirst:
		default:
			close(closedFirst)
		}
	}))
	defer srv.Close()

	ws, err := New(
		WithURL("ws"+strings.TrimPrefix(srv.URL, "http")),
		WithReconnectDelay(5*time.Second),
	)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	listenDone := make(chan error, 1)
	go func() { listenDone <- ws.Listen(nil) }()

	select {
	case <-closedFirst:
	case <-time.After(time.Second):
		t.Fatal("server did not close the initial connection")
	}
	// Allow Listen to enter reconnect's cancellable delay.
	time.Sleep(20 * time.Millisecond)
	if err := ws.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	select {
	case err := <-listenDone:
		if err != nil {
			t.Fatalf("Listen() after closing during reconnect = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close() did not cancel reconnect promptly")
	}
	if got := connections.Load(); got != 1 {
		t.Fatalf("connection count = %d, want 1", got)
	}
}

func waitForWebSocketState(t *testing.T, ws *WebSocket, predicate func(*WebSocket) bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		ws.mu.RLock()
		ready := predicate(ws)
		ws.mu.RUnlock()
		if ready {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for websocket state")
}
