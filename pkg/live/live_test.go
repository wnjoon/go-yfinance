package live

import (
	"encoding/base64"
	"encoding/binary"
	"math"
	"testing"
	"time"

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
	buf = append(buf, (1<<3)|2)        // tag = (1 << 3) | 2
	buf = append(buf, 4)               // length = 4
	buf = append(buf, "AAPL"...)       // value

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
	buf = append(buf, (7<<3)|0)
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
