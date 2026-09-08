---
title: Runtime behavior
description: HTTP headers, keepalives, concurrency, slow consumers, and shutdown semantics.
weight: 30
---

## Stream response {#response}

The package sets these Fiber response headers before streaming:

```text
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
Transfer-Encoding: chunked
```

The package sets response status `200` and sends an initial `:connected` comment so headers are flushed without waiting for a publish or heartbeat. Fiber and `fasthttp` read the response stream directly until it is canceled or a network write fails. `HEAD` requests return headers without registering a subscriber. A downstream server or proxy may transform hop-by-hop HTTP headers such as `Connection` and `Transfer-Encoding`.

## Keepalives {#keepalives}

Each connection owns a periodic ticker using the configured keepalive interval. It is stopped when the response stream closes and is not reset by application messages. When the response can be written, a tick sends:

```text
:keepalive
```

This is an SSE comment, not an application event; browser `EventSource` listeners do not receive it.

## Concurrency and payload ownership {#concurrency}

A server is intended to be shared by application goroutines. Publishing uses a channel, connection tracking uses `sync.Map`, and shutdown is guarded so repeated `Close` calls are safe.

`Publish` clones `Message.Data` before queueing. All publish methods also clone the event name and routing strings, including their backing storage. Subscription and handler methods copy their routing keys, so keys obtained from Fiber request buffers remain stable after the handler returns. `PublishJSON*` owns the byte slice produced by JSON marshaling and avoids an unnecessary second copy. Connections do not retain a `fiber.Ctx`.

## Slow consumers {#slow-consumers}

The hub offers a formatted message to each matching connection without blocking. If a connection queue has no free slot, that connection is removed and its response stream is canceled. Cancellation expires the transport write deadline to interrupt an already blocked network write; fasthttp retains responsibility for closing and recycling the connection. Buffered messages are discarded. Other matching clients continue normally.

This policy bounds the effect of one slow client but means delivery is not guaranteed. Applications that require complete history should publish state snapshots, provide a resynchronization endpoint, or use durable storage.

## Shutdown {#shutdown}

Call `Close` during application shutdown:

```go
sse := sseserver.New()
defer sse.Close()
```

Closing the server signals the hub, which removes active connections, cancels their response streams, and interrupts blocked writes. Response completion also unregisters the subscriber and stops its ticker. Subsequent publish or subscribe calls return `ErrServerClosed`. A zero-value server instead returns `ErrServerNotInitialized` from operations that require a hub.

`Close` signals shutdown without waiting for every HTTP connection to finish. It does not drain queued messages, close the Fiber application, or wait for external publishers. For an orderly shutdown, stop application publishers first, call `Close`, and then continue the Fiber shutdown sequence. A publish racing with shutdown may return `ErrServerClosed` or may return nil after queueing a message that is never delivered; treat nil as queue acceptance, not a delivery acknowledgement.
