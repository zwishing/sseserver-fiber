---
title: API reference
description: Public types, constructors, subscription methods, publishing methods, and errors.
weight: 10
---

Package import path:

```go
import sseserver "github.com/zwishing/sseserver-fiber"
```

## Types {#types}

```go
type Message struct {
	Event     string
	Data      []byte
	Namespace string
	Topic     string
}

type SSEMessage = Message
type Option func(*config) // config is internal; create options with With* functions.
type Server struct { /* unexported fields */ }
```

`SSEMessage` is retained as an alias for compatibility. Use `Message` in new code. Construct servers with `New`; the zero value does not start a hub.

## Constructor and lifecycle {#constructor}

```go
func New(opts ...Option) *Server
func (s *Server) Close()
```

`New` applies non-nil options, starts the internal hub, and returns a ready server. `Close` is safe to call more than once and is also safe on a nil or zero-value server.

## Subscriptions {#subscriptions}

```go
func (s *Server) Handler(namespace string) fiber.Handler
func (s *Server) HandlerWithTopic(namespace, topic string) fiber.Handler
func (s *Server) Subscribe(ctx fiber.Ctx, namespace string) error
func (s *Server) SubscribeWithTopic(ctx fiber.Ctx, namespace, topic string) error
```

`Handler*` is the direct form for route registration. `Subscribe*` is useful inside a custom Fiber handler after application-specific authentication, authorization, or parameter parsing.

```go
app.Get("/events/:tenant", func(c fiber.Ctx) error {
	tenant := c.Params("tenant")
	if !canSubscribe(c, tenant) {
		return fiber.ErrForbidden
	}
	return sse.SubscribeWithTopic(c, tenant, "progress")
})
```

Subscription calls copy the routing keys, install a long-lived response stream, and return after setup. The stream ends when canceled or when a response write fails; these causes are not exposed as distinct return errors from `Subscribe*`. `HEAD` requests do not create a subscription.

## Publishing {#publishing}

```go
func (s *Server) Publish(msg Message) error
func (s *Server) PublishEvent(namespace, event string, data []byte) error
func (s *Server) PublishEventWithTopic(namespace, topic, event string, data []byte) error
func (s *Server) PublishJSON(namespace, event string, payload any) error
func (s *Server) PublishJSONWithTopic(namespace, topic, event string, payload any) error
```

Namespace-only helpers publish with an empty topic. See [routing](../../guides/routing/) for the resulting broadcast behavior.

## Options {#options}

```go
func WithConnectionBuffer(size int) Option
func WithPublishBuffer(size int) Option
func WithKeepAliveInterval(interval time.Duration) Option
```

See [configuration](../configuration/) for defaults and tuning guidance.

## Errors {#errors}

```go
var ErrServerClosed = errors.New("sseserver: server closed")
var ErrServerNotInitialized = errors.New("sseserver: server not initialized")
var ErrInvalidEventName = errors.New("sseserver: event name must not contain CR or LF")
```

| Error | Meaning |
| --- | --- |
| `ErrServerClosed` | The receiver is nil, has been closed, or its hub shut down before an operation could proceed. |
| `ErrServerNotInitialized` | A zero-value `Server` was used without calling `New`. |
| `ErrInvalidEventName` | The published event name contains a carriage return or line feed. |

Use `errors.Is` when checking these errors. JSON marshal failures wrap the encoder error with `marshal SSE payload` context.

`Close` has no return value. It remains a safe no-op for nil and zero-value receivers; the error table applies to operations that require an available hub.
