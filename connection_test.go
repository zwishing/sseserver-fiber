package sseserver

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

func connectionCount(s *Server) int {
	n := 0
	s.hub.connections.Range(func(_, _ any) bool { n++; return true })
	return n
}

func TestLargeMessageSurvivesPartialReads(t *testing.T) {
	s := New()
	app := fiber.New()
	app.Get("/sse", s.Handler("test"))
	client, _ := streamClient(t, s, app, "GET", "/sse")
	response, err := http.ReadResponse(bufio.NewReader(client), nil)
	if err != nil {
		t.Fatal(err)
	}
	payload := strings.Repeat("  消息", 20000)
	if err := s.PublishEvent("test", "update", []byte(payload)); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(response.Body)
	var wire string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		wire += line
		if line == "\n" {
			break
		}
	}
	events, data := decodeEvents(wire)
	if len(data) != 1 || data[0] != payload || events[0] != "update" {
		t.Fatal("large message was truncated or changed by chunked/partial reads")
	}
}

func TestClosedStreamDiscardsBufferedMessages(t *testing.T) {
	h := newHub(defaultConfig())
	c := newConnection(nil, h, "test", "")
	h.connections.Store(c, true)
	h._broadcastMessage(Message{Namespace: "test", Data: []byte("queued")})
	_ = c.Close()
	if n, err := c.Read(make([]byte, 100)); n != 0 || err != io.EOF {
		t.Fatalf("read after close=(%d, %v), want (0, EOF)", n, err)
	}
}

func TestResponseCloseStopsTransportCancellation(t *testing.T) {
	transport, peer := net.Pipe()
	defer transport.Close()
	defer peer.Close()
	c := newConnection(transport, newHub(defaultConfig()), "test", "")
	c.cancel()
	_ = c.Close()
	_ = transport.SetWriteDeadline(time.Now().Add(time.Second))
	_ = peer.SetReadDeadline(time.Now().Add(time.Second))
	// A cancellation retry must no longer touch a transport fasthttp released.
	time.Sleep(30 * time.Millisecond)
	written := make(chan error, 1)
	go func() {
		_, err := transport.Write([]byte("ok"))
		written <- err
	}()
	var data [2]byte
	if _, err := io.ReadFull(peer, data[:]); err != nil {
		t.Fatalf("released transport was affected by SSE cancellation: %v", err)
	}
	if err := <-written; err != nil || string(data[:]) != "ok" {
		t.Fatalf("transport write after response release: data=%q err=%v", data, err)
	}
}

func waitFor(t *testing.T, description string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal(description)
		}
		time.Sleep(time.Millisecond)
	}
}

// net.Pipe exercises the complete HTTP path and produces deterministic backpressure
// without depending on OS socket buffer sizes or a listening port.
func streamClient(t *testing.T, s *Server, app *fiber.App, method, path string) (net.Conn, <-chan struct{}) {
	t.Helper()
	client, server := net.Pipe()
	finished := make(chan struct{})
	httpServer := &fasthttp.Server{Handler: app.Handler()}
	go func() {
		defer close(finished)
		_ = httpServer.ServeConn(server)
	}()
	t.Cleanup(func() {
		s.Close()
		_ = client.Close()
		_ = server.Close()
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Error("HTTP connection did not exit during cleanup")
		}
	})
	_ = client.SetDeadline(time.Now().Add(time.Second))
	if _, err := fmt.Fprintf(client, "%s %s HTTP/1.1\r\nHost: localhost\r\n\r\n", method, path); err != nil {
		t.Fatal(err)
	}
	return client, finished
}

func TestSubscriptionFlushesHeadersImmediately(t *testing.T) {
	s := New(WithKeepAliveInterval(time.Hour))
	app := fiber.New()
	app.Get("/sse", s.Handler("test"))
	client, _ := streamClient(t, s, app, "GET", "/sse")
	response, err := http.ReadResponse(bufio.NewReader(client), nil)
	if err != nil {
		t.Fatalf("headers unavailable before first publish/heartbeat: %v", err)
	}
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected SSE response: %s %v", response.Status, response.Header)
	}
}

func TestSubscriptionSendsPeriodicHeartbeats(t *testing.T) {
	s := New(WithKeepAliveInterval(10 * time.Millisecond))
	app := fiber.New()
	app.Get("/sse", s.Handler("test"))
	client, _ := streamClient(t, s, app, "GET", "/sse")
	response, err := http.ReadResponse(bufio.NewReader(client), nil)
	if err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(response.Body)
	if line, err := reader.ReadString('\n'); err != nil || line != ":connected\n" {
		t.Fatalf("initial comment = %q, err=%v", line, err)
	}
	for i := 0; i < 2; i++ {
		line, err := reader.ReadString('\n')
		if err != nil || line != ":keepalive\n" {
			t.Fatalf("heartbeat %d = %q, err=%v", i, line, err)
		}
	}
}

func TestDisconnectedSubscriberIsRemoved(t *testing.T) {
	s := New(WithKeepAliveInterval(10 * time.Millisecond))
	app := fiber.New()
	app.Get("/sse", s.Handler("test"))
	client, _ := streamClient(t, s, app, "GET", "/sse")
	waitFor(t, "subscriber not registered", func() bool { return connectionCount(s) == 1 })
	_ = client.Close()
	waitFor(t, "disconnected subscriber still registered", func() bool { return connectionCount(s) == 0 })
}

func TestSubscriptionCopiesRoutingKeys(t *testing.T) {
	s := New()
	defer s.Close()
	app := fiber.New()
	request := &fasthttp.RequestCtx{}
	request.Request.SetRequestURI("/original")
	ctx := app.AcquireCtx(request)
	if err := s.SubscribeWithTopic(ctx, ctx.Path(), ctx.Path()); err != nil {
		t.Fatal(err)
	}
	ctx.Path("/modified")
	app.ReleaseCtx(ctx)
	defer request.Response.Reset()
	if err := s.PublishEventWithTopic("/original", "/original", "update", []byte("secret")); err != nil {
		t.Fatal(err)
	}
	read := make(chan string, 1)
	go func() {
		reader := bufio.NewReader(request.Response.BodyStream())
		var wire string
		for {
			line, err := reader.ReadString('\n')
			wire += line
			if err != nil || line == "\n" {
				read <- wire
				return
			}
		}
	}()
	select {
	case wire := <-read:
		_, data := decodeEvents(wire)
		if len(data) != 1 || data[0] != "secret" {
			t.Fatalf("original subscriber did not receive its message: %q", wire)
		}
	case <-time.After(time.Second):
		t.Fatal("context reuse changed subscription routing")
	}
}

func TestCloseInterruptsBlockedNetworkWrite(t *testing.T) {
	for _, slowConsumer := range []bool{false, true} {
		t.Run(fmt.Sprintf("slow_consumer=%v", slowConsumer), func(t *testing.T) {
			s := New(WithKeepAliveInterval(time.Hour), WithConnectionBuffer(2))
			app := fiber.New()
			app.Get("/sse", s.Handler("test"))
			_, finished := streamClient(t, s, app, "GET", "/sse")
			waitFor(t, "subscriber not registered", func() bool { return connectionCount(s) == 1 })
			// The client never reads, so HTTP output is blocked even with no deadline.
			for i := 0; i < 20; i++ {
				if err := s.PublishEvent("test", "update", []byte("message")); err != nil {
					t.Fatal(err)
				}
				if !slowConsumer {
					break
				}
			}
			if !slowConsumer {
				s.Close()
			}
			waitFor(t, "closed/slow subscriber still registered", func() bool { return connectionCount(s) == 0 })
			select {
			case <-finished:
			case <-time.After(200 * time.Millisecond):
				t.Fatal("server is still blocked writing to the disconnected subscriber")
			}
		})
	}
}

func TestHeadDoesNotRegisterSubscriber(t *testing.T) {
	s := New()
	app := fiber.New()
	app.Get("/sse", s.Handler("test"))
	client, _ := streamClient(t, s, app, "HEAD", "/sse")
	response, err := http.ReadResponse(bufio.NewReader(client), &http.Request{Method: "HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatal(response.Status)
	}
	if data, err := io.ReadAll(response.Body); err != nil || len(data) != 0 {
		t.Fatalf("HEAD body=%q err=%v", data, err)
	}
	if n := connectionCount(s); n != 0 {
		t.Fatalf("HEAD created %d subscribers", n)
	}
}

type addressedConn struct{ net.Conn }

func (c addressedConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}

func TestClosePreservesFasthttpConnectionOwnership(t *testing.T) {
	s := New()
	defer s.Close()
	app := fiber.New()
	app.Get("/sse", func(ctx fiber.Ctx) error {
		ctx.Set("X-Padding", strings.Repeat("x", 8192))
		if err := s.Subscribe(ctx, "test"); err != nil {
			return err
		}
		// Force cancellation before fasthttp writes the response headers. Its
		// per-IP wrapper must remain usable until fasthttp releases it itself.
		s.Close()
		sub := ctx.RequestCtx().Response.BodyStream().(*connection)
		<-sub.done
		return nil
	})
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	httpServer := &fasthttp.Server{Handler: app.Handler(), MaxConnsPerIP: 1, WriteTimeout: time.Hour}
	result := make(chan error, 1)
	go func() { result <- httpServer.ServeConn(addressedConn{server}) }()
	_ = client.SetDeadline(time.Now().Add(time.Second))
	_, _ = io.WriteString(client, "GET /sse HTTP/1.1\r\nHost: localhost\r\n\r\n")
	select {
	case err := <-result:
		var panicErr *fasthttp.ErrBodyStreamWritePanic
		if errors.As(err, &panicErr) {
			t.Fatalf("closing SSE recycled fasthttp's transport before response handling finished: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled response did not exit")
	}
}
