package sseserver

import (
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

const httpOK = 200

// connection is the response body. fasthttp reads it directly, so there is no
// extra stream-writer goroutine or pipe that can remain blocked after shutdown.
type connection struct {
	hub         *hub
	send        chan []byte
	done        chan struct{}
	keepalive   *time.Ticker
	namespace   string
	topic       string
	pending     []byte // Only the response reader accesses pending.
	cancelOnce  sync.Once
	transportMu sync.Mutex
	transport   net.Conn
	abortTimer  *time.Timer
}

func newConnection(transport net.Conn, h *hub, namespace, topic string) *connection {
	return &connection{
		hub:       h,
		transport: transport,
		send:      make(chan []byte, h.config.connectionBuffer),
		done:      make(chan struct{}),
		keepalive: time.NewTicker(h.config.keepAlive),
		namespace: strings.Clone(namespace),
		topic:     strings.Clone(topic),
		pending:   []byte(":connected\n"),
	}
}

func (c *connection) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(c.pending) == 0 {
		select {
		case <-c.done:
		case c.pending = <-c.send:
		case <-c.keepalive.C:
			c.pending = []byte(":keepalive\n")
		}
	}
	// Closing takes priority over draining buffered messages.
	select {
	case <-c.done:
		return 0, io.EOF
	default:
	}
	n := copy(p, c.pending)
	c.pending = c.pending[n:]
	return n, nil
}

// Close is called by fasthttp when it releases the response body. Relinquish
// the transport before its connection wrapper can be returned to a pool.
func (c *connection) Close() error {
	c.transportMu.Lock()
	c.transport = nil
	if c.abortTimer != nil {
		c.abortTimer.Stop()
	}
	c.transportMu.Unlock()
	c.cancel()
	return nil
}

func (c *connection) cancel() {
	c.cancelOnce.Do(func() {
		close(c.done)
		c.hub.connections.Delete(c)
		c.keepalive.Stop()
		c.interruptWrite()
	})
}

func (c *connection) interruptWrite() {
	c.transportMu.Lock()
	defer c.transportMu.Unlock()
	if c.transport != nil {
		// Do not Close the transport here: fasthttp may use a pooled wrapper
		// that must remain valid until its response handling has finished.
		_ = c.transport.SetWriteDeadline(time.Now())
		// fasthttp can overwrite the deadline after the handler returns, and
		// large headers can block before the first body Read. Retry until its
		// response Close stops this timer and relinquishes the transport.
		c.abortTimer = time.AfterFunc(10*time.Millisecond, c.interruptWrite)
	}
}

func setupSSEHeaders(c fiber.Ctx) {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
}

func connect(ctx fiber.Ctx, h *hub, namespace, topic string) error {
	setupSSEHeaders(ctx)
	request := ctx.Status(httpOK).RequestCtx()
	// Fiber automatically routes HEAD to GET handlers. No stream is needed.
	if request.IsHead() {
		return nil
	}
	conn := newConnection(request.Conn(), h, namespace, topic)
	select {
	case <-h.shutdown:
		conn.keepalive.Stop()
		return ErrServerClosed
	case h.register <- conn:
	}
	// The initial comment flushes headers without waiting for a publish or tick.
	request.SetBodyStream(conn, -1)
	return nil
}
