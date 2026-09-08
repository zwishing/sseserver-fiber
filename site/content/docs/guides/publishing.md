---
title: Publish events
description: Select the raw, named-event, or JSON publishing API and understand payload handling.
weight: 20
---

All publishing methods eventually enqueue a `Message`. Choose the highest-level method that represents the payload you already have.

## Named byte events {#byte-events}

Use `PublishEvent` or `PublishEventWithTopic` when the payload is already encoded:

```go
if err := sse.PublishEvent("jobs", "heartbeat", []byte("ready")); err != nil {
	return err
}

if err := sse.PublishEventWithTopic(
	"tenant-a",
	"progress",
	"processing-percent",
	[]byte("75%"),
); err != nil {
	return err
}
```

The library copies `Data` before enqueueing it, so the caller may reuse or modify the original byte slice after `Publish` or `PublishEvent*` returns.

## JSON events {#json-events}

Use `PublishJSON` or `PublishJSONWithTopic` to marshal a Go value:

```go
payload := struct {
	JobID   string `json:"job_id"`
	Percent int    `json:"percent"`
}{
	JobID:   "job-42",
	Percent: 75,
}

if err := sse.PublishJSONWithTopic(
	"tenant-a",
	"progress",
	"job-progress",
	payload,
); err != nil {
	return err
}
```

JSON marshal failures are wrapped as `marshal SSE payload: ...`. Server lifecycle errors remain discoverable with `errors.Is`.

## Full messages {#full-message}

Use `Publish` when routing fields and the optional event name are assembled dynamically:

```go
err := sse.Publish(sseserver.Message{
	Namespace: "tenant-a",
	Topic:     "audit",
	Event:     "record-created",
	Data:      []byte(`{"id":"record-7"}`),
})
```

`SSEMessage` is a compatibility alias for `Message`; new code should use `Message`.

## SSE formatting {#formatting}

When `Event` is non-empty, the stream includes an `event:` line and browser clients should register a listener with that name. When it is empty, browsers dispatch the standard `message` event. Event names containing CR or LF are rejected with `ErrInvalidEventName` by every publish method.

Each line in a multiline payload becomes a separate `data:` field as required by SSE framing. CR and CRLF line endings are normalized to LF, and leading spaces and empty lines are preserved. Use JSON if the original carriage-return characters must be preserved in the decoded application value:

```text
event: update
data: first line
data: second line

```

The library does not add event IDs or retry fields. If an application needs replay or resume semantics, it must define and store them outside this package.
