---
title: 快速开始
description: 运行一个支持主题的 SSE 端点，并使用 EventSource 消费消息。
weight: 10
---

## 环境要求 {#requirements}

- Go 1.25 或更高版本。
- 一个 Fiber v3 应用。
- 用于测试事件流的浏览器或 `curl`。

## 安装 {#install}

在应用模块中执行：

```bash
go get github.com/zwishing/sseserver-fiber
```

## 创建服务 {#create-server}

下面的程序暴露 `/sse`，让每条连接订阅命名空间 `tenant-a` 与主题 `progress`，然后每秒发布一次进度事件。

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

`HandlerWithTopic` 与 `PublishEventWithTopic` 的参数都是先传 `namespace`，再传 `topic`。两端的值必须匹配，消息才会被投递。

## 使用 curl 验证 {#verify-curl}

运行应用，然后打开另一个终端：

```bash
curl -N http://localhost:8080/sse
```

具名事件会以 SSE 线格式出现：

```text
event:processing-percent
data:10%

event:processing-percent
data:20%
```

默认连接 ticker 每 15 秒写入一条保活注释（`:keepalive`）；发布应用事件不会重置这个 ticker。

## 在浏览器中消费 {#browser}

```html
<p id="progress">等待中…</p>
<script>
  const source = new EventSource("http://localhost:8080/sse");

  source.addEventListener("processing-percent", (event) => {
    document.querySelector("#progress").textContent = event.data;
  });

  source.onerror = () => {
    document.querySelector("#progress").textContent = "正在重连…";
  };
</script>
```

如果页面和 API 不同源，请在 SSE 路由前配置 Fiber CORS，并显式列出可信来源：

```go
import "github.com/gofiber/fiber/v3/middleware/cors"

app.Use("/sse", cors.New(cors.Config{
	AllowOrigins:     []string{"https://app.example.com"},
	AllowMethods:     []string{"GET"},
	AllowCredentials: true, // 仅在事件流使用 Cookie 时开启。
}))
app.Get("/sse", sse.HandlerWithTopic("tenant-a", "progress"))
```

跨域使用 Cookie 认证时，请通过 `new EventSource(url, { withCredentials: true })` 创建浏览器客户端。携带凭据的请求不能使用通配来源。如果请求同时属于跨站请求，会话 Cookie 还必须允许跨站发送——通常需要在 HTTPS 下设置 `SameSite=None; Secure`；浏览器隐私策略仍可能阻止第三方 Cookie。详见 [Set-Cookie 参考](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie)。同源页面不需要 CORS 中间件。

## 下一步 {#next-step}

在增加多个消息受众之前，请阅读[命名空间与主题路由](../../guides/routing/)。
