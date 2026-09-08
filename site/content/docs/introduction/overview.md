---
title: Overview
description: What sseserver-fiber provides, how data flows, and when to choose it.
weight: 10
---

`sseserver-fiber` is a small in-process [Server-Sent Events](https://html.spec.whatwg.org/multipage/server-sent-events.html) broker for Fiber v3. It keeps HTTP event streams open and routes application messages to connected clients.

Use it when a Fiber application needs one-way, server-to-browser updates such as job progress, status changes, notifications, or live dashboards.

## What it provides {#provides}

- Fiber handlers for namespace-only or namespace-and-topic subscriptions.
- Raw-byte, JSON, and full-message publishing APIs.
- Exact namespace matching and optional topic targeting.
- Configurable publish and per-connection queues.
- Periodic SSE keepalive comments.
- Explicit, idempotent shutdown through `Server.Close`.

## Message flow {#message-flow}

```mermaid
flowchart LR
    A[Application code] -->|Publish| B[Server publish queue]
    B --> C[Hub]
    C -->|namespace + topic match| D[Connection queue]
    D --> E[Fiber response stream]
    E --> F[Browser EventSource]
```

`New` starts one hub goroutine. A subscription registers a connection with that hub, then Fiber streams queue entries to the client. A publish call enqueues a message; the hub formats it once and offers it to every matching connection.

## Delivery model {#delivery-model}

Delivery is live and best effort. Messages are not stored for later replay. A client receives only messages published while its connection is active, and a client whose connection queue fills is disconnected so it cannot stall other subscribers.

> [!IMPORTANT]
> The library is not a durable message broker. It does not persist events, coordinate multiple application processes, or replay missed messages. Add an external broker or event store when those guarantees are required.

Authentication and authorization remain Fiber application concerns. Apply middleware to the SSE route before registering a handler.

## Next step {#next-step}

Follow the [quick start](../../get-started/quickstart/) to run a complete publisher and browser subscriber.
