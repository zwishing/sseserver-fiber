package sseserver

import (
	"sync"
	"time"
)

// hub owns active connections and routes messages by namespace.
type hub struct {
	broadcast    chan Message     // Inbound publish queue
	connections  sync.Map         // Active connections
	register     chan *connection // Registration requests
	unregister   chan *connection // Unregistration requests
	shutdown     chan struct{}    // Shutdown signal
	shutdownOnce sync.Once        // Ensures idempotent shutdown
	config       config           // Server configuration
	sentMsgs     uint64           // Total published messages
	startupTime  time.Time        // Creation time
}

// newHub creates a hub instance.
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
		case c := <-h.unregister:
			// Unregister a connection.
			h._unregisterConn(c)
		case msg := <-h.broadcast:
			// Broadcast a message and increment the counter.
			h.sentMsgs++
			h._broadcastMessage(msg)
		}
	}
}

// _unregisterConn removes a connection from the hub.
func (h *hub) _unregisterConn(c *connection) {
	h.connections.Delete(c)
}

// _shutdownConn removes a connection and closes its send queue.
func (h *hub) _shutdownConn(c *connection) {
	// Unregister first to avoid sends to a closed queue.
	h._unregisterConn(c)
	// Closing the queue lets the stream writer exit.
	c.closeSend()
}

// _broadcastMessage sends a formatted message to matching subscribers.
func (h *hub) _broadcastMessage(msg Message) {
	formattedMsg := msg.sseFormat()
	h.connections.Range(func(k, v interface{}) bool {
		c := k.(*connection)
		if msg.Namespace == c.namespace && (msg.Topic == "" || msg.Topic == c.topic) {
			select {
			case c.send <- formattedMsg:
			default:
				// Drop slow consumers when their queue is full.
				h._shutdownConn(c)
			}
		}
		return true
	})
}
