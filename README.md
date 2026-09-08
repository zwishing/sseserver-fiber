# SSEServer for Fiber

`sseserver-fiber` is a small SSE broker for Fiber. It manages subscriber connections,
keeps streams alive, and routes messages by namespace and topic.

## Installation

```bash
go get github.com/zwishing/sseserver-fiber
```

## Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/zwishing/sseserver-fiber"
)

func main() {
	app := fiber.New()
	sse := sseserver.New()
	defer sse.Close()

	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Cache-Control"},
	}))

	app.Get("/sse", sse.HandlerWithTopic("tenant-a", "progress"))

	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)
		defer ticker.Stop()

		for i := 1; i <= 100; i++ {
			<-ticker.C
			_ = sse.PublishEventWithTopic("tenant-a", "progress", "processing-percent", []byte(fmt.Sprintf("%d%%", i)))
		}
	}()

	if err := app.Listen(":8080"); err != nil {
		panic(err)
	}
}
```

## API

- `sseserver.New(opts ...Option) *Server`
- `(*Server).Handler(namespace string) fiber.Handler`
- `(*Server).HandlerWithTopic(namespace, topic string) fiber.Handler`
- `(*Server).Subscribe(ctx fiber.Ctx, namespace string) error`
- `(*Server).SubscribeWithTopic(ctx fiber.Ctx, namespace, topic string) error`
- `(*Server).Publish(msg sseserver.Message) error`
- `(*Server).PublishEvent(namespace, event string, data []byte) error`
- `(*Server).PublishEventWithTopic(namespace, topic, event string, data []byte) error`
- `(*Server).PublishJSON(namespace, event string, payload any) error`
- `(*Server).PublishJSONWithTopic(namespace, topic, event string, payload any) error`
- `(*Server).Close()`

`Handler` / `PublishEvent` / `PublishJSON` remain namespace-only shortcuts and internally use an empty topic.

## Runtime behavior

- Subscriptions copy namespace/topic keys, including values read from Fiber requests. Publish methods copy event and routing strings; raw publish methods also copy the payload bytes.
- New streams send a `:connected` comment immediately, followed by periodic `:keepalive` comments. `HEAD` requests return headers without creating a subscription.
- Event names containing CR or LF return `ErrInvalidEventName`. Payload CR/CRLF line endings become LF; leading spaces and empty lines are preserved. Use JSON when the application must preserve literal carriage returns.
- Full subscriber queues cancel that subscriber's stream and interrupt blocked writes. `Close` signals shutdown and discards pending messages; it does not wait for all HTTP connections to finish or shut down the Fiber app.

## Options

- `WithConnectionBuffer(size int)`
- `WithPublishBuffer(size int)`
- `WithKeepAliveInterval(interval time.Duration)`

## Documentation

The bilingual OINK documentation site lives in [`site/`](site/README.md). It
includes a quick start, routing and publishing guides, the complete public API,
configuration defaults, and runtime behavior.
