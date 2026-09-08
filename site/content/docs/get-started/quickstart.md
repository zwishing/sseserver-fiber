---
title: Quick start
description: Run a topic-aware SSE endpoint and consume it with EventSource.
weight: 10
---

## Requirements {#requirements}

- Go 1.25 or newer.
- A Fiber v3 application.
- A browser or `curl` for testing the event stream.

## Install {#install}

From your application module:

```bash
go get github.com/zwishing/sseserver-fiber
```

## Create the server {#create-server}

The following program exposes `/sse`, subscribes each connection to namespace `tenant-a` and topic `progress`, then publishes progress events once per second.

```go
package main

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	sseserver "github.com/zwishing/sseserver-fiber"
)

func main() {
	app := fiber.New()
	sse := sseserver.New()
	defer sse.Close()

	app.Get("/sse", sse.HandlerWithTopic("tenant-a", "progress"))

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for percent := 10; percent <= 100; percent += 10 {
			<-ticker.C
			err := sse.PublishEventWithTopic(
				"tenant-a",
				"progress",
				"processing-percent",
				[]byte(fmt.Sprintf("%d%%", percent)),
			)
			if err != nil {
				return
			}
		}
	}()

	if err := app.Listen(":8080"); err != nil {
		panic(err)
	}
}
```

`HandlerWithTopic` and `PublishEventWithTopic` both take `namespace` before `topic`. The values must match for delivery.

## Verify with curl {#verify-curl}

Run the application, then open another terminal:

```bash
curl -N http://localhost:8080/sse
```

Named events appear in SSE wire format:

```text
event:processing-percent
data:10%

event:processing-percent
data:20%
```

The default connection ticker writes keepalive comments (`:keepalive`) every 15 seconds. Publishing an application event does not reset that ticker.

## Consume from a browser {#browser}

```html
<p id="progress">Waiting…</p>
<script>
  const source = new EventSource("http://localhost:8080/sse");

  source.addEventListener("processing-percent", (event) => {
    document.querySelector("#progress").textContent = event.data;
  });

  source.onerror = () => {
    document.querySelector("#progress").textContent = "Reconnecting…";
  };
</script>
```

If the page and API use different origins, configure Fiber CORS before the SSE route. List trusted origins explicitly:

```go
import "github.com/gofiber/fiber/v3/middleware/cors"

app.Use("/sse", cors.New(cors.Config{
	AllowOrigins:     []string{"https://app.example.com"},
	AllowMethods:     []string{"GET"},
	AllowCredentials: true, // Only when the stream uses cookies.
}))
app.Get("/sse", sse.HandlerWithTopic("tenant-a", "progress"))
```

For cross-origin cookie authentication, construct the browser client with `new EventSource(url, { withCredentials: true })`. Do not combine credentialed requests with a wildcard origin. If the request is also cross-site, the session cookie must be eligible for cross-site requests—normally `SameSite=None; Secure` over HTTPS—and browser privacy policy may still block third-party cookies. See the [Set-Cookie reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie). A same-origin page needs no CORS middleware.

## Next step {#next-step}

Read [namespace and topic routing](../../guides/routing/) before adding multiple event audiences.
