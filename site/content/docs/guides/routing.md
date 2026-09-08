---
title: Namespace and topic routing
description: Choose routing keys and predict exactly which subscribers receive a message.
weight: 10
---

Every connection has a namespace and a topic. A namespace is required by the public subscription APIs; a topic is optional.

## Subscribe {#subscribe}

Use `Handler` when one route represents an entire namespace:

```go
app.Get("/events", sse.Handler("tenant-a"))
```

Use `HandlerWithTopic` when a route should receive one topic inside a namespace:

```go
app.Get("/events/progress", sse.HandlerWithTopic("tenant-a", "progress"))
app.Get("/events/audit", sse.HandlerWithTopic("tenant-a", "audit"))
```

The first argument is always the namespace; the second is always the topic.

## Publish targets {#publish-targets}

Routing first requires an exact namespace match. The message topic then controls the breadth of delivery:

| Published message | Subscriber `tenant-a / ""` | Subscriber `tenant-a / progress` | Subscriber `tenant-a / audit` | Subscriber `tenant-b / progress` |
| --- | ---: | ---: | ---: | ---: |
| `tenant-a / ""` | ✓ | ✓ | ✓ | — |
| `tenant-a / progress` | — | ✓ | — | — |
| `tenant-a / audit` | — | — | ✓ | — |

An empty **message** topic broadcasts to every connection in the matching namespace, including topic-specific connections. A non-empty message topic is delivered only to connections with the same topic. Therefore a namespace-only subscription does not receive topic-specific publishes.

```go
// Broadcast to every tenant-a connection, regardless of its subscribed topic.
err := sse.PublishEvent("tenant-a", "maintenance", []byte("starting"))

// Deliver only to tenant-a connections subscribed to progress.
err = sse.PublishEventWithTopic(
	"tenant-a",
	"progress",
	"processing-percent",
	[]byte("50%"),
)
```

## Choose routing keys {#choose-keys}

Use namespaces for the strongest application-level partition, such as a tenant, account, workspace, or job. Use topics for streams within that partition, such as `progress`, `audit`, or `notifications`.

Keep keys stable and canonical. Matching is case-sensitive and exact; `Tenant-A` and `tenant-a` are different namespaces.

The package does not validate routing keys. Empty namespaces, whitespace, and arbitrary characters are matched literally. Validate and normalize untrusted input in the application; prefer non-empty namespaces with a documented character and length policy.

> [!WARNING]
> Namespace and topic strings are routing labels, not security boundaries. Authenticate the request and verify that it may subscribe to the selected labels before calling `Handler`, `HandlerWithTopic`, `Subscribe`, or `SubscribeWithTopic`.
