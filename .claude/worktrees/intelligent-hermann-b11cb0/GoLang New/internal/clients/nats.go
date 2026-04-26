package clients

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSClient wraps a NATS connection with structured logging and simple publish helpers.
type NATSClient struct {
	conn   *nats.Conn
	logger *slog.Logger
}

// NewNATSClient connects to the given NATS server URL and returns a ready client.
func NewNATSClient(url, name string, logger *slog.Logger) (*NATSClient, error) {
	nc, err := nats.Connect(
		url,
		nats.Name(name),
		nats.Timeout(10*time.Second),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Warn("NATS disconnected", "url", url, "error", err)
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			logger.Info("NATS reconnected", "url", c.ConnectedUrl())
		}),
		nats.ClosedHandler(func(c *nats.Conn) {
			if cerr := c.LastError(); cerr != nil {
				logger.Warn("NATS connection closed", "error", cerr)
				return
			}
			logger.Info("NATS connection closed")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect %s: %w", url, err)
	}

	logger.Info("NATS connected", "url", nc.ConnectedUrl(), "name", name)
	return &NATSClient{conn: nc, logger: logger}, nil
}

// Publish sends a message to the given subject.
func (n *NATSClient) Publish(subject string, payload []byte) {
	if err := n.conn.Publish(subject, payload); err != nil {
		n.logger.Error("NATS publish failed", "subject", subject, "error", err)
	}
}

// SubscribeRequests registers a request-reply handler on the given subject.
// handler is called only for messages that carry a reply subject (i.e. sent via
// nats.Request). It receives the reply address and the raw request payload.
// The returned cancel function unsubscribes when called (e.g. on context cancel).
func (n *NATSClient) SubscribeRequests(subject string, handler func(reply string, payload []byte)) (func(), error) {
	sub, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		if msg.Reply == "" {
			return // plain publish, not a request — ignore
		}
		handler(msg.Reply, msg.Data)
	})
	if err != nil {
		return nil, fmt.Errorf("nats subscribe %s: %w", subject, err)
	}
	n.logger.Info("NATS request handler registered", "subject", subject)
	return func() { _ = sub.Unsubscribe() }, nil
}

// SubscribeRequestsWithSubject registers a request-reply handler on the given
// subject and passes the matched request subject to the handler. This is
// useful for wildcard request handlers that need to inspect the concrete
// subject token (for example, tm_map/full.<chain>). The returned cancel
// function unsubscribes when called (e.g. on context cancel).
func (n *NATSClient) SubscribeRequestsWithSubject(subject string, handler func(requestSubject string, reply string, payload []byte)) (func(), error) {
	sub, err := n.conn.Subscribe(subject, func(msg *nats.Msg) {
		if msg.Reply == "" {
			return // plain publish, not a request — ignore
		}
		handler(msg.Subject, msg.Reply, msg.Data)
	})
	if err != nil {
		return nil, fmt.Errorf("nats subscribe %s: %w", subject, err)
	}
	n.logger.Info("NATS request handler registered", "subject", subject)
	return func() { _ = sub.Unsubscribe() }, nil
}

// Reply sends a response to the given reply subject.
func (n *NATSClient) Reply(replySubject string, payload []byte) {
	if err := n.conn.Publish(replySubject, payload); err != nil {
		n.logger.Error("NATS reply failed", "reply_subject", replySubject, "error", err)
	}
}

// Close drains pending messages and then closes the connection.
func (n *NATSClient) Close() {
	if err := n.conn.Drain(); err != nil {
		n.logger.Warn("NATS drain failed", "error", err)
	}
	n.conn.Close()
}
