package sseserver

import (
	"sync"
	"time"
)

// Hub 管理所有活跃的客户端连接，并负责将消息广播到匹配命名空间的连接
//
// Hub 是 Server 的"核心"，但保持私有以隐藏实现细节
type hub struct {
	broadcast    chan Message     // 需要广播的入站消息
	connections  sync.Map         // 已注册的连接
	register     chan *connection // 连接注册请求通道
	unregister   chan *connection // 连接注销请求通道
	shutdown     chan struct{}    // 内部关闭通知通道
	shutdownOnce sync.Once        // 保证 Shutdown 幂等
	config       config           // Server 配置
	sentMsgs     uint64           // 启动以来广播的消息数
	startupTime  time.Time        // Hub 创建时间
}

// 创建新的 Hub 实例
func newHub(cfg config) *hub {
	return &hub{
		broadcast:   make(chan Message, cfg.publishBuffer),
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		shutdown:    make(chan struct{}),
		config:      cfg,
		startupTime: time.Now(),
	}
}

// Shutdown 用于取消 hub 运行循环
//
// 目前仅在测试中使用，未来可能需要在生产环境中更优雅地关闭 Server
// TODO: 在更广泛暴露此接口之前需要深入思考
func (h *hub) Shutdown() {
	h.shutdownOnce.Do(func() {
		close(h.shutdown)
	})
}

// Start 在后台 goroutine 中启动 hub 的主运行循环
func (h *hub) Start() {
	go h.run()
}

// hub 的主运行循环
func (h *hub) run() {
	for {
		select {
		case <-h.shutdown:
			// 关闭时断开所有连接
			h.connections.Range(func(k, v interface{}) bool {
				h._shutdownConn(k.(*connection))
				return true
			})
			return
		case c := <-h.register:
			// 注册新连接
			h.connections.Store(c, true)
		case c := <-h.unregister:
			// 注销连接
			h._unregisterConn(c)
		case msg := <-h.broadcast:
			// 广播消息并增加计数
			h.sentMsgs++
			h._broadcastMessage(msg)
		}
	}
}

// _unregisterConn 从 hub 中移除客户端连接
// 可以安全地多次调用同一连接
func (h *hub) _unregisterConn(c *connection) {
	h.connections.Delete(c)
}

// _shutdownConn 从 hub 中移除客户端连接并关闭它
// 对每个连接只能调用一次，以避免 panic!
func (h *hub) _shutdownConn(c *connection) {
	// 为了最大程度的安全，在关闭连接前必须先从 hub 中注销
	// 这样可以避免向已关闭的通道发送数据导致 panic
	h._unregisterConn(c)
	// 关闭连接的发送通道，这将导致其退出事件循环并返回到 HTTP 处理器
	c.closeSend()
}

// _broadcastMessage 向所有匹配的客户端广播消息
// 如果由于任何客户端的发送缓冲区已满而失败，将关闭该连接
func (h *hub) _broadcastMessage(msg Message) {
	formattedMsg := msg.sseFormat()
	h.connections.Range(func(k, v interface{}) bool {
		c := k.(*connection)
		if msg.Namespace == c.namespace {
			select {
			case c.send <- formattedMsg:
			default:
				// 发送失败时关闭连接
				h._shutdownConn(c)
				/*
					我们已经关闭了发送通道，理论上连接应该会清理，
					但如果连接死锁了怎么办？

					关闭发送通道会导致 handleFunc 退出，Go会认为
					我们已经完成了连接处理...但如果连接卡住了呢？

					TODO: 研究在 http.Handler 中使用 panic 来强制处理

					我们需要确保始终关闭 HTTP 连接，
					这样服务器就不会用尽最大打开套接字数
				*/
			}
		}
		return true
	})
}
