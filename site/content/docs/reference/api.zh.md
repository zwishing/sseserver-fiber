---
title: API 参考
description: 公开类型、构造函数、订阅方法、发布方法与错误。
weight: 10
---

包导入路径：

```go
import sseserver "github.com/zwishing/sseserver-fiber"
```

## 类型 {#types}

```go
type Message struct {
	Event     string
	Data      []byte
	Namespace string
	Topic     string
}

type SSEMessage = Message
type Option func(*config) // config 是内部类型；使用 With* 函数创建选项。
type Server struct { /* 未导出字段 */ }
```

`SSEMessage` 作为兼容别名保留；新代码请使用 `Message`。服务必须通过 `New` 构造，零值不会启动 Hub。

## 构造与生命周期 {#constructor}

```go
func New(opts ...Option) *Server
func (s *Server) Close()
```

`New` 应用所有非 nil 选项，启动内部 Hub，并返回可用服务。`Close` 可以安全地重复调用，也可以用于 nil 或零值服务。

## 订阅 {#subscriptions}

```go
func (s *Server) Handler(namespace string) fiber.Handler
func (s *Server) HandlerWithTopic(namespace, topic string) fiber.Handler
func (s *Server) Subscribe(ctx fiber.Ctx, namespace string) error
func (s *Server) SubscribeWithTopic(ctx fiber.Ctx, namespace, topic string) error
```

`Handler*` 适合直接注册路由。需要先执行应用级认证、鉴权或参数解析时，可以在自定义 Fiber 处理器中调用 `Subscribe*`。

```go
app.Get("/events/:tenant", func(c fiber.Ctx) error {
	tenant := c.Params("tenant")
	if !canSubscribe(c, tenant) {
		return fiber.ErrForbidden
	}
	return sse.SubscribeWithTopic(c, tenant, "progress")
})
```

订阅调用会复制路由键、安装长连接响应流，并在设置完成后返回。流会在被取消或响应写入失败时结束；`Subscribe*` 不会把这些原因作为可区分的返回错误暴露出来。`HEAD` 请求不会创建订阅。

## 发布 {#publishing}

```go
func (s *Server) Publish(msg Message) error
func (s *Server) PublishEvent(namespace, event string, data []byte) error
func (s *Server) PublishEventWithTopic(namespace, topic, event string, data []byte) error
func (s *Server) PublishJSON(namespace, event string, payload any) error
func (s *Server) PublishJSONWithTopic(namespace, topic, event string, payload any) error
```

只接收命名空间的辅助方法会使用空主题发布。由此产生的广播行为见[路由指南](../../guides/routing/)。

## 选项 {#options}

```go
func WithConnectionBuffer(size int) Option
func WithPublishBuffer(size int) Option
func WithKeepAliveInterval(interval time.Duration) Option
```

默认值和调优建议见[配置](../configuration/)。

## 错误 {#errors}

```go
var ErrServerClosed = errors.New("sseserver: server closed")
var ErrServerNotInitialized = errors.New("sseserver: server not initialized")
var ErrInvalidEventName = errors.New("sseserver: event name must not contain CR or LF")
```

| 错误 | 含义 |
| --- | --- |
| `ErrServerClosed` | 接收者为 nil、已关闭，或操作执行前 Hub 已停止。 |
| `ErrServerNotInitialized` | 未调用 `New` 就使用了零值 `Server`。 |
| `ErrInvalidEventName` | 发布的事件名包含回车或换行。 |

判断这些错误时请使用 `errors.Is`。JSON 编码失败时，编码器错误会带上 `marshal SSE payload` 上下文。

`Close` 没有返回值；用于 nil 或零值接收者时仍然是安全的空操作。上表只适用于需要可用 Hub 的操作。
