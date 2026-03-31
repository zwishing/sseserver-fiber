package sseserver

import (
	"bufio"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

const httpOK = 200

type connection struct {
	ctx       fiber.Ctx
	created   time.Time
	send      chan []byte
	namespace string
	msgsSent  uint64
	closeOnce sync.Once
}

func newConnection(ctx fiber.Ctx, namespace string, bufferSize int) *connection {
	return &connection{
		ctx:       ctx,
		send:      make(chan []byte, bufferSize),
		created:   time.Now(),
		namespace: namespace,
	}
}

type connectionStatus struct {
	Path      string `json:"request_path"`
	Namespace string `json:"namespace"`
	Created   int64  `json:"created_at"`
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
	MsgsSent  uint64 `json:"msgs_sent"`
}

func (c *connection) Status() connectionStatus {
	return connectionStatus{
		Path:      c.ctx.Path(),
		Namespace: c.namespace,
		Created:   c.created.Unix(),
		ClientIP:  c.ctx.IP(),
		UserAgent: c.ctx.Get("User-Agent"),
		MsgsSent:  atomic.LoadUint64(&c.msgsSent),
	}
}

func (c *connection) closeSend() {
	c.closeOnce.Do(func() {
		close(c.send)
	})
}

func write(w *bufio.Writer, data []byte) error {
	if _, err := w.Write(data); err != nil {
		return err
	}
	return w.Flush()
}

func (c *connection) writer(h *hub) {
	keepaliveTickler := time.NewTicker(h.config.keepAlive)
	keepaliveMsg := []byte(":keepalive\n")
	defer keepaliveTickler.Stop()

	c.ctx.Status(httpOK).RequestCtx().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		for {
			select {
			case msg, ok := <-c.send:
				if !ok {
					return
				}
				if err := write(w, msg); err != nil {
					select {
					case h.unregister <- c:
					case <-h.shutdown:
					}
					return
				}
				atomic.AddUint64(&c.msgsSent, 1)

			case <-keepaliveTickler.C:
				if err := write(w, keepaliveMsg); err != nil {
					select {
					case h.unregister <- c:
					case <-h.shutdown:
					}
					return
				}
			}
		}
	}))
}

func setupSSEHeaders(c fiber.Ctx) {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
}

func connect(c fiber.Ctx, h *hub, namespace string) error {
	setupSSEHeaders(c)

	conn := newConnection(c, namespace, h.config.connectionBuffer)

	select {
	case <-h.shutdown:
		return ErrServerClosed
	case h.register <- conn:
	}

	conn.writer(h)
	return nil
}
