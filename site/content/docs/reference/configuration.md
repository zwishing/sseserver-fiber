---
title: Configuration
description: Server options, defaults, validation, and queue-sizing guidance.
weight: 20
---

Pass functional options to `New`:

```go
sse := sseserver.New(
	sseserver.WithConnectionBuffer(128),
	sseserver.WithPublishBuffer(512),
	sseserver.WithKeepAliveInterval(10*time.Second),
)
defer sse.Close()
```

## Defaults {#defaults}

| Option | Default | Scope | Behavior |
| --- | ---: | --- | --- |
| `WithConnectionBuffer` | `256` messages | Each connection | Capacity between the hub and one response stream. |
| `WithPublishBuffer` | `256` messages | One server | Capacity between publishers and the hub. |
| `WithKeepAliveInterval` | `15s` | Each connection | Period of the SSE `:keepalive` ticker. |

Values less than or equal to zero are ignored, leaving the corresponding default in effect.

## Tune connection buffers {#connection-buffer}

A larger connection buffer absorbs short client or network stalls at the cost of memory per active connection. When this queue becomes full, the hub disconnects that slow consumer rather than blocking delivery to every other client.

Choose a capacity that covers expected short bursts, not indefinite outages. Reconnect and resynchronize clients at the application layer if missing an event matters.

## Tune the publish buffer {#publish-buffer}

The publish buffer absorbs bursts from application goroutines. A publish call waits when this queue is full until space becomes available or shutdown is observed. Increasing it smooths larger bursts but does not increase the rate at which the hub can fan out messages.

Shutdown is not a drain barrier. A `Publish` racing with `Close` may return `ErrServerClosed`, or it may enqueue and return nil while the hub is exiting; nil means accepted by the in-process queue, not delivered to a client. Stop publishers before closing the server, and use application-level acknowledgements when delivery confirmation matters.

## Tune keepalives {#keepalive}

Keepalive comments help intermediaries and clients observe that a stream is still active. Each connection has a periodic ticker; application messages do not reset it. Choose an interval shorter than the idle timeout of proxies in front of the application. Very short intervals increase network traffic across every connection.
