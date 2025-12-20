// Package live provides real-time WebSocket streaming for Yahoo Finance data.
//
// # Overview
//
// The live package enables real-time price updates via Yahoo Finance's
// WebSocket streaming API. Messages are delivered as protobuf-encoded
// pricing data.
//
// # Basic Usage
//
//	ws, err := live.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer ws.Close()
//
//	// Subscribe to symbols
//	ws.Subscribe([]string{"AAPL", "MSFT", "GOOGL"})
//
//	// Listen for updates
//	ws.Listen(func(data *models.PricingData) {
//	    fmt.Printf("%s: $%.2f\n", data.ID, data.Price)
//	})
//
// # With Ticker Integration
//
//	ticker, _ := ticker.New("AAPL")
//	ticker.Live(func(data *models.PricingData) {
//	    fmt.Printf("Price: $%.2f\n", data.Price)
//	})
//
// # Thread Safety
//
// All WebSocket methods are safe for concurrent use from multiple goroutines.
package live

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wnjoon/go-yfinance/pkg/models"
)

const (
	// DefaultURL is the Yahoo Finance WebSocket endpoint.
	DefaultURL = "wss://streamer.finance.yahoo.com/?version=2"

	// DefaultHeartbeatInterval is the interval for re-sending subscriptions.
	DefaultHeartbeatInterval = 15 * time.Second

	// DefaultReconnectDelay is the delay before attempting reconnection.
	DefaultReconnectDelay = 3 * time.Second
)

// MessageHandler is a callback function for handling pricing data.
type MessageHandler func(*models.PricingData)

// ErrorHandler is a callback function for handling errors.
type ErrorHandler func(error)

// WebSocket represents a WebSocket client for Yahoo Finance streaming.
type WebSocket struct {
	url               string
	conn              *websocket.Conn
	subscriptions     map[string]struct{}
	messageHandler    MessageHandler
	errorHandler      ErrorHandler
	heartbeatInterval time.Duration
	reconnectDelay    time.Duration

	mu           sync.RWMutex
	done         chan struct{}
	heartbeatDone chan struct{}
	isConnected  bool
	isListening  bool
}

// Option is a function that configures a WebSocket.
type Option func(*WebSocket)

// WithURL sets a custom WebSocket URL.
func WithURL(url string) Option {
	return func(ws *WebSocket) {
		ws.url = url
	}
}

// WithHeartbeatInterval sets the heartbeat interval.
func WithHeartbeatInterval(d time.Duration) Option {
	return func(ws *WebSocket) {
		ws.heartbeatInterval = d
	}
}

// WithReconnectDelay sets the reconnection delay.
func WithReconnectDelay(d time.Duration) Option {
	return func(ws *WebSocket) {
		ws.reconnectDelay = d
	}
}

// WithErrorHandler sets the error handler callback.
func WithErrorHandler(handler ErrorHandler) Option {
	return func(ws *WebSocket) {
		ws.errorHandler = handler
	}
}

// New creates a new WebSocket client.
//
// Example:
//
//	ws, err := live.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer ws.Close()
func New(opts ...Option) (*WebSocket, error) {
	ws := &WebSocket{
		url:               DefaultURL,
		subscriptions:     make(map[string]struct{}),
		heartbeatInterval: DefaultHeartbeatInterval,
		reconnectDelay:    DefaultReconnectDelay,
		done:              make(chan struct{}),
	}

	for _, opt := range opts {
		opt(ws)
	}

	return ws, nil
}

// Connect establishes the WebSocket connection.
func (ws *WebSocket) Connect() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.isConnected {
		return nil
	}

	conn, _, err := websocket.DefaultDialer.Dial(ws.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	ws.conn = conn
	ws.isConnected = true
	return nil
}

// Subscribe adds symbols to the subscription list and sends subscribe message.
//
// Example:
//
//	ws.Subscribe([]string{"AAPL", "MSFT"})
func (ws *WebSocket) Subscribe(symbols []string) error {
	if err := ws.Connect(); err != nil {
		return err
	}

	ws.mu.Lock()
	for _, sym := range symbols {
		ws.subscriptions[strings.ToUpper(sym)] = struct{}{}
	}
	symbolList := ws.getSubscriptionList()
	ws.mu.Unlock()

	return ws.sendSubscribe(symbolList)
}

// Unsubscribe removes symbols from the subscription list.
//
// Example:
//
//	ws.Unsubscribe([]string{"AAPL"})
func (ws *WebSocket) Unsubscribe(symbols []string) error {
	if err := ws.Connect(); err != nil {
		return err
	}

	ws.mu.Lock()
	for _, sym := range symbols {
		delete(ws.subscriptions, strings.ToUpper(sym))
	}
	ws.mu.Unlock()

	msg := map[string]interface{}{
		"unsubscribe": symbols,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.conn.WriteMessage(websocket.TextMessage, data)
}

// Listen starts listening for messages and calls the handler for each message.
// This method blocks until Close() is called or an unrecoverable error occurs.
//
// Example:
//
//	ws.Listen(func(data *models.PricingData) {
//	    fmt.Printf("%s: $%.2f\n", data.ID, data.Price)
//	})
func (ws *WebSocket) Listen(handler MessageHandler) error {
	if err := ws.Connect(); err != nil {
		return err
	}

	ws.mu.Lock()
	ws.messageHandler = handler
	ws.isListening = true
	ws.heartbeatDone = make(chan struct{})
	ws.mu.Unlock()

	// Start heartbeat goroutine
	go ws.heartbeatLoop()

	// Message loop
	for {
		select {
		case <-ws.done:
			return nil
		default:
			if err := ws.readMessage(); err != nil {
				ws.mu.RLock()
				handler := ws.errorHandler
				ws.mu.RUnlock()

				if handler != nil {
					handler(err)
				}

				// Attempt reconnection
				if ws.reconnect() != nil {
					return err
				}
			}
		}
	}
}

// ListenAsync starts listening in a separate goroutine.
// Returns immediately. Use Close() to stop listening.
//
// Example:
//
//	ws.ListenAsync(func(data *models.PricingData) {
//	    fmt.Printf("%s: $%.2f\n", data.ID, data.Price)
//	})
//	// ... do other work ...
//	ws.Close()
func (ws *WebSocket) ListenAsync(handler MessageHandler) error {
	go func() {
		_ = ws.Listen(handler)
	}()
	return nil
}

// Close closes the WebSocket connection.
func (ws *WebSocket) Close() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if !ws.isConnected {
		return nil
	}

	// Signal goroutines to stop
	close(ws.done)
	if ws.heartbeatDone != nil {
		close(ws.heartbeatDone)
	}

	ws.isConnected = false
	ws.isListening = false

	if ws.conn != nil {
		return ws.conn.Close()
	}
	return nil
}

// IsConnected returns true if the WebSocket is connected.
func (ws *WebSocket) IsConnected() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.isConnected
}

// Subscriptions returns the current list of subscribed symbols.
func (ws *WebSocket) Subscriptions() []string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.getSubscriptionList()
}

// getSubscriptionList returns the subscription list (must be called with lock held).
func (ws *WebSocket) getSubscriptionList() []string {
	list := make([]string, 0, len(ws.subscriptions))
	for sym := range ws.subscriptions {
		list = append(list, sym)
	}
	return list
}

// sendSubscribe sends a subscribe message.
func (ws *WebSocket) sendSubscribe(symbols []string) error {
	if len(symbols) == 0 {
		return nil
	}

	msg := map[string]interface{}{
		"subscribe": symbols,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.conn.WriteMessage(websocket.TextMessage, data)
}

// readMessage reads and processes a single message.
func (ws *WebSocket) readMessage() error {
	ws.mu.RLock()
	conn := ws.conn
	handler := ws.messageHandler
	ws.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		return err
	}

	// Parse JSON wrapper
	var wrapper struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(message, &wrapper); err != nil {
		return fmt.Errorf("failed to parse message wrapper: %w", err)
	}

	if wrapper.Message == "" {
		return nil // Empty message, skip
	}

	// Decode protobuf
	pricingData, err := decodeBase64Message(wrapper.Message)
	if err != nil {
		return fmt.Errorf("failed to decode pricing data: %w", err)
	}

	// Call handler
	if handler != nil {
		handler(pricingData)
	}

	return nil
}

// heartbeatLoop sends periodic subscription messages to keep connection alive.
func (ws *WebSocket) heartbeatLoop() {
	ticker := time.NewTicker(ws.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ws.heartbeatDone:
			return
		case <-ws.done:
			return
		case <-ticker.C:
			ws.mu.RLock()
			symbols := ws.getSubscriptionList()
			ws.mu.RUnlock()

			if len(symbols) > 0 {
				_ = ws.sendSubscribe(symbols)
			}
		}
	}
}

// reconnect attempts to reconnect after a connection failure.
func (ws *WebSocket) reconnect() error {
	ws.mu.Lock()
	ws.isConnected = false
	if ws.conn != nil {
		ws.conn.Close()
		ws.conn = nil
	}
	subscriptions := ws.getSubscriptionList()
	ws.mu.Unlock()

	time.Sleep(ws.reconnectDelay)

	if err := ws.Connect(); err != nil {
		return err
	}

	// Re-subscribe
	if len(subscriptions) > 0 {
		return ws.sendSubscribe(subscriptions)
	}
	return nil
}
