package ws

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"github.com/ybina/polymarket-go/client/clob"
	"github.com/ybina/polymarket-go/client/clob/clob_types"
	"github.com/ybina/polymarket-go/client/config"
	"github.com/ybina/polymarket-go/client/endpoint"
	"github.com/ybina/polymarket-go/client/types"
)

type WebSocketClientOptions struct {
	AssetIDs []string

	Markets []string

	AutoReconnect bool

	ReconnectDelay time.Duration

	MaxReconnectAttempts int

	Debug bool

	Logger *log.Logger

	ProxyUrl string
}

// MessageHandler is a callback function for handling messages
type MessageHandler func(msg types.MarketChannelMessage)

// BookMessageHandler handles book messages
type BookMessageHandler func(msg *types.BookMessage)

// PriceChangeMessageHandler handles price change messages
type PriceChangeMessageHandler func(msg *types.PriceChangeMessage)

// TickSizeChangeMessageHandler handles tick size change messages
type TickSizeChangeMessageHandler func(msg *types.TickSizeChangeMessage)

// LastTradePriceMessageHandler handles last trade price messages
type LastTradePriceMessageHandler func(msg *types.LastTradePriceMessage)

// WebSocketCallbacks holds callback functions for different events
type WebSocketCallbacks struct {
	OnBook           BookMessageHandler
	OnPriceChange    PriceChangeMessageHandler
	OnTickSizeChange TickSizeChangeMessageHandler
	OnLastTradePrice LastTradePriceMessageHandler
	OnMessage        MessageHandler
	OnError          func(error)
	OnConnect        func()
	OnDisconnect     func(code int, reason string)
	OnReconnect      func(attempt int)
}

type WebSocketClient struct {
	mu                sync.RWMutex
	writeMu           sync.Mutex
	reconnectMu       sync.Mutex
	closedOnce        sync.Once
	clobClient        *clob.ClobClient
	options           *WebSocketClientOptions
	callbacks         *WebSocketCallbacks
	conn              *websocket.Conn
	pingTicker        *time.Ticker
	reconnectTimer    *time.Timer
	done              chan struct{}
	reconnectAttempts int
	isConnecting      bool
	shouldReconnect   bool
	logger            *log.Logger
}

func NewWebSocketClient(clobClient *clob.ClobClient, options *WebSocketClientOptions) *WebSocketClient {
	if options == nil {
		options = &WebSocketClientOptions{}
	}

	if options.AutoReconnect && options.ReconnectDelay == 0 {
		options.ReconnectDelay = 5 * time.Second
	}

	logger := options.Logger
	if logger == nil {
		logger = log.Default()
	}

	return &WebSocketClient{
		clobClient:      clobClient,
		options:         options,
		callbacks:       &WebSocketCallbacks{},
		done:            nil,
		shouldReconnect: true,
		logger:          logger,
	}
}

func (ws *WebSocketClient) On(callbacks *WebSocketCallbacks) *WebSocketClient {
	ws.callbacks = callbacks
	return ws
}

func (ws *WebSocketClient) Connect() error {
	ws.mu.Lock()
	if ws.isConnecting || (ws.conn != nil && ws.IsConnected()) {
		ws.mu.Unlock()
		log.Printf("Already connected or connecting")
		return nil
	}
	ws.isConnecting = true
	ws.shouldReconnect = true
	ws.mu.Unlock()

	option := clob_types.ClobOption{
		TurnkeyAccount: common.Address{},
		SafeAccount:    common.Address{},
	}
	apiKey, err := ws.clobClient.DeriveApiKey(nil, option)
	if err != nil {
		ws.mu.Lock()
		ws.isConnecting = false
		ws.mu.Unlock()
		return fmt.Errorf("failed to derive API key: %w", err)
	}
	log.Printf("API key derived:%v\n", apiKey.Key)

	fullURL := fmt.Sprintf("%s/ws/market", endpoint.WsUrl)
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		NextProtos: []string{"http/1.1"},
	}
	dialer := websocket.Dialer{TLSClientConfig: tlsConfig}
	if ws.options.ProxyUrl != "" {
		proxyUrl, err := url.Parse(ws.options.ProxyUrl)
		if err != nil {
			ws.mu.Lock()
			ws.isConnecting = false
			ws.mu.Unlock()
			return fmt.Errorf("failed to parse proxy url: %w", err)
		}
		dialer.Proxy = http.ProxyURL(proxyUrl)
	}

	conn, _, err := dialer.Dial(fullURL, nil)
	if err != nil {
		ws.mu.Lock()
		ws.isConnecting = false
		ws.mu.Unlock()
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	ws.mu.Lock()
	ws.conn = conn
	ws.isConnecting = false
	ws.reconnectAttempts = 0

	ws.done = make(chan struct{})
	ws.closedOnce = sync.Once{}
	ws.mu.Unlock()

	conn.SetPongHandler(func(appData string) error {
		log.Printf("Received CONTROL PONG: %s\n", appData)
		return nil
	})
	conn.SetPingHandler(func(appData string) error {
		log.Printf("Received CONTROL PING: %s\n", appData)
		_ = ws.withConnWrite(func(c *websocket.Conn) error {
			return c.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(5*time.Second))
		})
		return nil
	})

	log.Printf("WebSocket connected \n")

	if err = ws.sendInitialSubscription(); err != nil {
		ws.forceCloseWithReason(-1, fmt.Sprintf("send subscription failed: %v", err))
		ws.handleDisconnect(-1, fmt.Sprintf("send subscription failed: %v", err))
		return fmt.Errorf("failed to send subscription: %w", err)
	}

	go ws.handleMessages()
	go ws.pingWorker()

	if ws.callbacks.OnConnect != nil {
		ws.callbacks.OnConnect()
	}
	return nil
}

func (ws *WebSocketClient) Disconnect() {
	ws.mu.Lock()
	ws.shouldReconnect = false
	ws.mu.Unlock()

	ws.cleanup()

	ws.forceCloseWithReason(websocket.CloseNormalClosure, "client disconnect")
	ws.signalDone()
}

func (ws *WebSocketClient) signalDone() {
	ws.mu.RLock()
	done := ws.done
	ws.mu.RUnlock()
	if done == nil {
		return
	}
	ws.closedOnce.Do(func() {
		close(done)
	})
}

func (ws *WebSocketClient) forceCloseWithReason(code int, reason string) {
	ws.mu.Lock()
	conn := ws.conn
	ws.conn = nil
	ws.mu.Unlock()

	if conn == nil {
		return
	}
	_ = conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason),
		time.Now().Add(2*time.Second),
	)
	_ = conn.Close()
}

func (ws *WebSocketClient) Subscribe(assetIDs []string) error {
	ws.mu.Lock()
	ws.options.AssetIDs = append(ws.options.AssetIDs, assetIDs...)
	ws.mu.Unlock()

	if ws.IsConnected() {
		return ws.sendSubscription(assetIDs)
	}

	return nil
}

func (ws *WebSocketClient) Unsubscribe(assetIDs []string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	filtered := make([]string, 0, len(ws.options.AssetIDs))
	for _, id := range ws.options.AssetIDs {
		shouldKeep := true
		for _, unsubID := range assetIDs {
			if id == unsubID {
				shouldKeep = false
				break
			}
		}
		if shouldKeep {
			filtered = append(filtered, id)
		}
	}
	ws.options.AssetIDs = filtered
}

func (ws *WebSocketClient) IsConnected() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.conn != nil
}

func (ws *WebSocketClient) Wait() {
	<-ws.done
}

func (ws *WebSocketClient) sendInitialSubscription() error {
	ws.mu.RLock()
	conn := ws.conn
	assetIDs := ws.options.AssetIDs
	ws.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}
	if assetIDs == nil || len(assetIDs) <= 0 {
		return nil
	}
	message := map[string]interface{}{
		"assets_ids": assetIDs,
		"type":       "market",
	}
	log.Printf("subscribe request: %v\n", message)
	return ws.withConnWrite(func(conn *websocket.Conn) error {
		return conn.WriteJSON(message)
	})
}

func (ws *WebSocketClient) sendSubscription(tokenIds []string) error {
	ws.mu.RLock()
	conn := ws.conn
	ws.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	message := map[string]interface{}{
		"assets_ids": tokenIds,
		"operation":  "subscribe",
	}

	log.Printf("subscribe request: %v\n", message)
	return ws.withConnWrite(func(conn *websocket.Conn) error {
		return conn.WriteJSON(message)
	})
}

func (ws *WebSocketClient) handleMessages() {
	defer func() {
		log.Printf("Message handler stopped \n")
	}()

	for {
		ws.mu.RLock()
		conn := ws.conn
		done := ws.done
		ws.mu.RUnlock()

		if conn == nil {
			return
		}

		select {
		case <-done:
			return
		default:
		}

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if ce, ok := err.(*websocket.CloseError); ok {
				log.Printf("ReadMessage CloseError: code= %v, reason= %v\n", ce.Code, ce.Text)
				ws.handleDisconnect(ce.Code, ce.Text)
			} else {
				log.Printf("ReadMessage error: %v\n", err)
				ws.handleDisconnect(-1, err.Error())
			}
			return
		}

		if messageType == websocket.TextMessage {
			txt := string(message)

			if txt == "PONG" {
				log.Printf("Received TEXT PONG \n")
				continue
			}
			if txt == "ping" || txt == "PING" {
				log.Printf("Received TEXT PING, reply TEXT PONG \n")
				_ = ws.withConnWrite(func(c *websocket.Conn) error {
					reply := "pong"
					if txt == "PING" {
						reply = "PONG"
					}
					return c.WriteMessage(websocket.TextMessage, []byte(reply))
				})
				continue
			}
			ws.processMessage(message)
		}
	}
}

func (ws *WebSocketClient) processMessage(data []byte) {
	var messages []json.RawMessage
	if err := json.Unmarshal(data, &messages); err == nil {
		for _, msgData := range messages {
			ws.parseAndDispatch(msgData)
		}
	} else {
		ws.parseAndDispatch(data)
	}
}

func (ws *WebSocketClient) parseAndDispatch(data []byte) {
	msg, err := types.ParseMarketChannelMessage(data)
	if err != nil {
		ws.handleError(fmt.Errorf("failed to parse message: %w", err))
		log.Printf("Raw message: %s", string(data))
		return
	}

	// Call specific handlers based on message type
	switch msg.GetEventType() {
	case types.EventTypeBook:
		if bookMsg, ok := types.AsBookMessage(msg); ok && ws.callbacks.OnBook != nil {
			ws.callbacks.OnBook(bookMsg)
		}
	case types.EventTypePriceChange:
		if pcMsg, ok := types.AsPriceChangeMessage(msg); ok && ws.callbacks.OnPriceChange != nil {
			ws.callbacks.OnPriceChange(pcMsg)
		}
	case types.EventTypeTickSizeChange:
		if tsMsg, ok := types.AsTickSizeChangeMessage(msg); ok && ws.callbacks.OnTickSizeChange != nil {
			ws.callbacks.OnTickSizeChange(tsMsg)
		}
	case types.EventTypeLastTradePrice:
		if ltMsg, ok := types.AsLastTradePriceMessage(msg); ok && ws.callbacks.OnLastTradePrice != nil {
			ws.callbacks.OnLastTradePrice(ltMsg)
		}
	}
	if ws.callbacks.OnMessage != nil {
		ws.callbacks.OnMessage(msg)
	}
}

func (ws *WebSocketClient) pingWorker() {
	ws.mu.Lock()
	ws.pingTicker = time.NewTicker(config.GetWsPingInterval())
	ticker := ws.pingTicker
	ws.mu.Unlock()

	defer ticker.Stop()

	for {
		ws.mu.RLock()
		done := ws.done
		ws.mu.RUnlock()

		select {
		case <-done:
			return
		case <-ticker.C:
			err := ws.withConnWrite(func(conn *websocket.Conn) error {
				return conn.WriteMessage(websocket.TextMessage, []byte("PING"))
			})
			if err != nil {
				log.Printf("failed to send ping: %v\n", err.Error())
				ws.handleDisconnect(-1, "ping send failed: "+err.Error())
				return
			}
			log.Printf("Sent TEXT PING \n")
		}
	}
}

func (ws *WebSocketClient) handleError(err error) {
	if ws.callbacks.OnError != nil {
		ws.callbacks.OnError(err)
	} else {
		log.Printf("Error: %v\n", err)
	}
}

func (ws *WebSocketClient) handleDisconnect(code int, reason string) {

	ws.cleanup()

	ws.forceCloseWithReason(code, reason)

	ws.signalDone()

	if ws.callbacks.OnDisconnect != nil {
		ws.callbacks.OnDisconnect(code, reason)
	}

	ws.mu.RLock()
	shouldReconnect := ws.shouldReconnect
	autoReconnect := ws.options.AutoReconnect
	ws.mu.RUnlock()

	if shouldReconnect && autoReconnect {
		ws.tryReconnect()
	}
}

func (ws *WebSocketClient) tryReconnect() {
	ws.reconnectMu.Lock()
	defer ws.reconnectMu.Unlock()
	ws.mu.RLock()
	hasTimer := ws.reconnectTimer != nil
	ws.mu.RUnlock()
	if hasTimer {
		return
	}

	ws.mu.Lock()
	if ws.options.MaxReconnectAttempts > 0 && ws.reconnectAttempts >= ws.options.MaxReconnectAttempts {
		ws.mu.Unlock()
		log.Println("Max reconnect attempts reached")
		return
	}

	ws.reconnectAttempts++
	attempt := ws.reconnectAttempts
	delay := ws.options.ReconnectDelay
	if delay <= 0 {
		delay = 5 * time.Second
	}
	ws.mu.Unlock()

	log.Printf(fmt.Sprintf("Scheduling reconnect attempt %d after %s...\n", attempt, delay))

	if ws.callbacks.OnReconnect != nil {
		ws.callbacks.OnReconnect(attempt)
	}

	ws.mu.Lock()
	ws.reconnectTimer = time.AfterFunc(delay, func() {
		ws.mu.Lock()
		ws.reconnectTimer = nil
		ws.mu.Unlock()

		log.Printf(fmt.Sprintf("Attempting reconnect %d... \n", attempt))
		if err := ws.Connect(); err != nil {
			log.Printf("Reconnect failed: %v\n", err)
			ws.handleDisconnect(-1, "reconnect failed: "+err.Error())
		}
	})
	ws.mu.Unlock()
}

func (ws *WebSocketClient) cleanup() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.pingTicker != nil {
		ws.pingTicker.Stop()
		ws.pingTicker = nil
	}

	if ws.reconnectTimer != nil {
		ws.reconnectTimer.Stop()
		ws.reconnectTimer = nil
	}
}

func (ws *WebSocketClient) withConnWrite(fn func(conn *websocket.Conn) error) error {
	ws.mu.RLock()
	conn := ws.conn
	ws.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("not connected")
	}

	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()
	return fn(conn)
}
