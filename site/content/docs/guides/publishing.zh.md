---
title: 发布事件
description: 选择原始消息、具名事件或 JSON 发布 API，并理解载荷处理方式。
weight: 20
---

所有发布方法最终都会把一个 `Message` 放入队列。请选择与手头载荷最匹配的高层方法。

## 具名字节事件 {#byte-events}

载荷已经编码时，使用 `PublishEvent` 或 `PublishEventWithTopic`：

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

入队前，库会复制 `Data`。因此 `Publish` 或 `PublishEvent*` 返回后，调用方可以复用或修改原始字节切片。

## JSON 事件 {#json-events}

使用 `PublishJSON` 或 `PublishJSONWithTopic` 编码 Go 值：

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

JSON 编码失败时，错误会被包装为 `marshal SSE payload: ...`。服务生命周期错误仍然可以使用 `errors.Is` 判断。

## 完整消息 {#full-message}

路由字段和可选事件名需要动态组装时，使用 `Publish`：

```go
err := sse.Publish(sseserver.Message{
	Namespace: "tenant-a",
	Topic:     "audit",
	Event:     "record-created",
	Data:      []byte(`{"id":"record-7"}`),
})
```

`SSEMessage` 是 `Message` 的兼容别名；新代码应使用 `Message`。

## SSE 格式 {#formatting}

`Event` 非空时，事件流包含 `event:` 行，浏览器应使用相同名称注册监听器。`Event` 为空时，浏览器会派发标准 `message` 事件。所有发布方法都会拒绝包含 CR 或 LF 的事件名，并返回 `ErrInvalidEventName`。

多行载荷的每一行都会按 SSE 格式转换为独立的 `data:` 字段。CR 与 CRLF 会规范化为 LF，行首空格与空行会保留。如果应用需要保留原始回车字符，请使用 JSON 编码，在客户端解码后还原：

```text
event: update
data: first line
data: second line

```

这个库不会添加事件 ID 或重试字段。应用若需要重放或断点续传语义，必须在包外定义并存储相关信息。
