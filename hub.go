package sseserver

import "sync"

// hub owns active connections and routes messages by namespace.
type hub struct {
	broadcast    chan Message     // Inbound publish queue
	connections  sync.Map         // Active connections
	register     chan *connection // Registration requests
	shutdown     chan struct{}    // Shutdown signal
	shutdownOnce sync.Once        // Ensures idempotent shutdown
	config       config           // Server configuration
}

// newHub creates a hub instance.
func newHub(cfg config) *hub {
	return &hub{
		broadcast: make(chan Message, cfg.publishBuffer),
		register:  make(chan *connection),
		shutdown:  make(chan struct{}),
		config:    cfg,
	}
}

// Shutdown stops the hub loop.
func (h *hub) Shutdown() {
	h.shutdownOnce.Do(func() {
		close(h.shutdown)
	})
}

// Start launches the hub loop.
func (h *hub) Start() {
	go h.run()
}

// run is the main hub loop.
func (h *hub) run() {
	for {
		select {
		case <-h.shutdown:
			// Disconnect all clients on shutdown.
			h.connections.Range(func(k, v interface{}) bool {
				h._shutdownConn(k.(*connection))
				return true
			})
			return
		case c := <-h.register:
			// Register a new connection.
			h.connections.Store(c, true)
			// The response can finish before this goroutine completes registration.
			select {
			case <-c.done:
				h.connections.Delete(c)
			default:
			}
		case msg := <-h.broadcast:
			h._broadcastMessage(msg)
		}
	}
}

// _shutdownConn removes a connection and interrupts its response stream.
func (h *hub) _shutdownConn(c *connection) {
	c.cancel()
}

// _broadcastMessage sends a formatted message to matching subscribers.
func (h *hub) _broadcastMessage(msg Message) {
	formattedMsg := msg.sseFormat()
	h.connections.Range(func(k, v interface{}) bool {
		c := k.(*connection)
		if msg.Namespace == c.namespace && (msg.Topic == "" || msg.Topic == c.topic) {
			select {
			case <-c.done:
			case c.send <- formattedMsg:
			default:
				// Drop slow consumers when their queue is full.
				h._shutdownConn(c)
			}
		}
		return true
	})
}
