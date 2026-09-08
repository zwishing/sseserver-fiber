---
title: 运行时行为
description: HTTP 响应头、保活、并发、慢消费者与关闭语义。
weight: 30
---

## 事件流响应 {#response}

这个包会在开始流式写入前设置以下 Fiber 响应头：

```text
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
Transfer-Encoding: chunked
```

这个包把响应状态码设置为 `200`，并发送初始 `:connected` 注释，让响应头立即刷新，无需等待发布消息或心跳。Fiber 与 `fasthttp` 直接读取响应流，直到流被取消或网络写入失败。`HEAD` 请求只返回响应头，不会注册订阅者。下游服务器或代理可能改写 `Connection`、`Transfer-Encoding` 等逐跳 HTTP 响应头。

## 保活 {#keepalives}

每条连接都按配置的保活间隔运行一个周期 ticker，并在响应流关闭时停止它；应用消息不会重置它。在响应可以写入时，每次 tick 发送：

```text
:keepalive
```

它是 SSE 注释而不是应用事件；浏览器的 `EventSource` 监听器不会收到它。

## 并发与载荷所有权 {#concurrency}

服务可以由多个应用 goroutine 共享。发布使用 channel，连接跟踪使用 `sync.Map`，关闭流程带有保护，因此重复调用 `Close` 是安全的。

`Publish` 会在入队前复制 `Message.Data`。所有发布方法还会复制事件名和路由字符串的底层数据；订阅与 handler 方法也会复制路由键，因此从 Fiber 请求缓冲区取得的路由键在 handler 返回后仍然稳定。`PublishJSON*` 拥有 JSON 编码产生的字节切片，因此不需要再次复制。连接不会长期持有 `fiber.Ctx`。

## 慢消费者 {#slow-consumers}

Hub 会以非阻塞方式把格式化消息交给每条匹配连接。如果某条连接的队列没有空位，该连接会被移除并取消响应流。取消时会将底层连接的写截止时间设为当前时间，解除已经阻塞的网络写入；最终的连接关闭与回收仍由 fasthttp 完成。缓冲中的消息会被丢弃，其他匹配客户端不受影响。

这项策略限制了单个慢客户端的影响，但也意味着投递并非有保证。要求完整历史的应用应发布状态快照、提供重新同步端点，或使用持久化存储。

## 关闭 {#shutdown}

在应用关闭时调用 `Close`：

```go
sse := sseserver.New()
defer sse.Close()
```

关闭服务会通知 Hub，由 Hub 移除活动连接、取消响应流并解除阻塞写。响应结束也会注销订阅者并停止 ticker。后续发布或订阅调用返回 `ErrServerClosed`；需要 Hub 的操作如果使用零值服务，则返回 `ErrServerNotInitialized`。

`Close` 发出关闭信号后返回，不会等待所有 HTTP 连接退出，也不会排空已入队消息、关闭 Fiber 应用或等待外部发布方。为了有序关闭，应先停止应用发布方，再调用 `Close`，最后继续 Fiber 的关闭流程。与关闭竞态的发布可能返回 `ErrServerClosed`，也可能在消息入队但永远不会送达后返回 nil；nil 只表示队列接受，不是送达确认。
