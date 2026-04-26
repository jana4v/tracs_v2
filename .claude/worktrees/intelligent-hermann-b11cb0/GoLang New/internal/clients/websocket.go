package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"github.com/mainframe/tm-system/internal/models"
)

// WSSubscriber manages a persistent WebSocket connection with auto-reconnect.
// Pattern preserved from old ReceiveScTm.go:56-73 with context support and exponential backoff.
type WSSubscriber struct {
	URL            string
	ChainName      string
	OnMessage      func(ctx context.Context, msg []byte) error
	OnConnect      func(ctx context.Context)
	OnDisconnect   func(ctx context.Context, err error)
	Logger         *slog.Logger
	// SendSubscribe controls whether a {"action":"subscribe","paramList":[""]} message
	// is sent after connecting. TM chains require this handshake to begin streaming;
	// SCOS/SMON chains push data immediately upon connection (no request needed).
	SendSubscribe  bool
}

// Run connects to the WebSocket and processes messages in a loop.
// On disconnect, it retries with exponential backoff.
// It sends a subscribe request on each successful connection (preserved from old code).
func (ws *WSSubscriber) Run(ctx context.Context) error {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := ws.connectAndRead(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if ws.OnDisconnect != nil {
				ws.OnDisconnect(ctx, err)
			}
			ws.Logger.Warn("WebSocket disconnected, reconnecting",
				"chain", ws.ChainName,
				"url", ws.URL,
				"backoff", backoff,
				"error", err,
			)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, maxBackoff)
	}
}

func (ws *WSSubscriber) connectAndRead(ctx context.Context) error {
	u, err := url.Parse(ws.URL)
	if err != nil {
		return fmt.Errorf("parse URL: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial %s: %w", ws.URL, err)
	}
	defer conn.Close()

	ws.Logger.Info("WebSocket connected", "chain", ws.ChainName, "url", ws.URL)
	if ws.OnConnect != nil {
		ws.OnConnect(ctx)
	}

	// TM chains require a subscribe handshake before data flows.
	// SCOS/SMON chains push data immediately on connect — no request needed.
	if ws.SendSubscribe {
		req := models.ReqMessage{
			Action:    "subscribe",
			ParamList: []string{""},
		}
		reqJSON, _ := json.Marshal(req)
		if err := conn.WriteMessage(websocket.TextMessage, reqJSON); err != nil {
			return fmt.Errorf("send subscribe: %w", err)
		}
	}

	// Read messages until error or context cancellation
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		if err := ws.OnMessage(ctx, msg); err != nil {
			ws.Logger.Error("message handler error",
				"chain", ws.ChainName,
				"error", err,
			)
		}
	}
}
