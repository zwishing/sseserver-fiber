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

	app.Get("/sse", sse.HandlerWithTopic("progress", "tenant-a"))

	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)
		defer ticker.Stop()

		for i := 1; i <= 100; i++ {
			<-ticker.C
			_ = sse.PublishEventWithTopic("progress", "tenant-a", "processing-percent", []byte(fmt.Sprintf("%d%%", i)))
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

## Options

- `WithConnectionBuffer(size int)`
- `WithKeepAliveInterval(interval time.Duration)`
