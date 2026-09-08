---
title: 命名空间与主题路由
description: 选择路由键，并准确判断哪些订阅者会收到消息。
weight: 10
---

每条连接都有命名空间与主题。公开订阅 API 要求提供命名空间，而主题是可选的。

## 订阅 {#subscribe}

当一条路由代表整个命名空间时，使用 `Handler`：

```go
app.Get("/events", sse.Handler("tenant-a"))
```

当路由只接收命名空间中的一个主题时，使用 `HandlerWithTopic`：

```go
app.Get("/events/progress", sse.HandlerWithTopic("tenant-a", "progress"))
app.Get("/events/audit", sse.HandlerWithTopic("tenant-a", "audit"))
```

第一个参数始终是命名空间，第二个参数始终是主题。

## 发布目标 {#publish-targets}

路由首先要求命名空间精确匹配，然后由消息主题控制投递范围：

| 发布的消息 | 订阅 `tenant-a / ""` | 订阅 `tenant-a / progress` | 订阅 `tenant-a / audit` | 订阅 `tenant-b / progress` |
| --- | ---: | ---: | ---: | ---: |
| `tenant-a / ""` | ✓ | ✓ | ✓ | — |
| `tenant-a / progress` | — | ✓ | — | — |
| `tenant-a / audit` | — | — | ✓ | — |

**消息**主题为空时，会广播给匹配命名空间中的所有连接，包括订阅了具体主题的连接。消息主题非空时，只投递给主题相同的连接。因此，只订阅命名空间的连接不会收到主题定向消息。

```go
// 广播给 tenant-a 下的所有连接，不考虑连接订阅的主题。
err := sse.PublishEvent("tenant-a", "maintenance", []byte("starting"))

// 只投递给 tenant-a 下订阅 progress 的连接。
err = sse.PublishEventWithTopic(
	"tenant-a",
	"progress",
	"processing-percent",
	[]byte("50%"),
)
```

## 选择路由键 {#choose-keys}

把命名空间用于最强的应用级分区，例如租户、账户、工作区或任务；把主题用于分区内的事件流，例如 `progress`、`audit` 或 `notifications`。

路由键应保持稳定且规范。匹配区分大小写并要求完全一致；`Tenant-A` 和 `tenant-a` 是不同的命名空间。

这个包不会校验路由键。空命名空间、空白字符和任意字符都会按字面值匹配。应用应校验并规范化不可信输入；建议使用非空命名空间，并明确规定允许的字符与长度。

> [!WARNING]
> 命名空间和主题只是路由标签，不是安全边界。在调用 `Handler`、`HandlerWithTopic`、`Subscribe` 或 `SubscribeWithTopic` 前，应认证请求，并验证它有权订阅所选标签。
