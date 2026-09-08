---
title: 配置
description: 服务选项、默认值、校验规则与队列容量建议。
weight: 20
---

把函数式选项传给 `New`：

```go
sse := sseserver.New(
	sseserver.WithConnectionBuffer(128),
	sseserver.WithPublishBuffer(512),
	sseserver.WithKeepAliveInterval(10*time.Second),
)
defer sse.Close()
```

## 默认值 {#defaults}

| 选项 | 默认值 | 作用域 | 行为 |
| --- | ---: | --- | --- |
| `WithConnectionBuffer` | `256` 条消息 | 每条连接 | Hub 与单个响应流之间的队列容量。 |
| `WithPublishBuffer` | `256` 条消息 | 每个服务 | 发布方与 Hub 之间的队列容量。 |
| `WithKeepAliveInterval` | `15s` | 每条连接 | SSE `:keepalive` ticker 的周期。 |

小于或等于零的值会被忽略，相应配置继续使用默认值。

## 调整连接队列 {#connection-buffer}

更大的连接队列能吸收短暂的客户端或网络停顿，代价是每条活动连接占用更多内存。队列填满后，Hub 会断开这个慢消费者，而不是阻塞所有其他客户端的投递。

容量只需覆盖预期的短时突发，不应试图容纳无限期中断。如果遗漏事件很重要，应在应用层让客户端重连并重新同步状态。

## 调整发布队列 {#publish-buffer}

发布队列用于吸收应用 goroutine 的突发写入。队列已满时，发布调用会等待，直到出现空位或观察到关闭信号。增大容量可以平滑更大的突发，但不会提高 Hub 的扇出速度。

关闭并不是排空屏障。与 `Close` 竞态的 `Publish` 可能返回 `ErrServerClosed`，也可能在 Hub 正在退出时完成入队并返回 nil；nil 只表示进程内队列接受了消息，不表示客户端已经收到。关闭服务前应先停止发布方；需要确认送达时，应使用应用级确认机制。

## 调整保活间隔 {#keepalive}

保活注释让中间代理和客户端知道事件流仍然存活。每条连接都有一个周期 ticker，应用消息不会重置它。该间隔应短于应用前置代理的空闲超时。间隔过短会增加每条连接的网络流量。
