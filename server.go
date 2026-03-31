package sseserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gofiber/fiber/v3"
)

var (
	ErrServerClosed         = errors.New("sseserver: server closed")
	ErrServerNotInitialized = errors.New("sseserver: server not initialized")
)

// Server manages SSE subscriptions and message publishing.
type Server struct {
	hub       *hub
	closeOnce sync.Once
	closed    atomic.Bool
}

// New creates a new SSE server instance.
func New(opts ...Option) *Server {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	s := &Server{
		hub: newHub(cfg),
	}
	s.hub.Start()
	return s
}

// Handler returns a Fiber handler that subscribes the request to a namespace.
func (s *Server) Handler(namespace string) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		return s.Subscribe(ctx, namespace)
	}
}

// Subscribe registers the current request as an SSE subscriber.
func (s *Server) Subscribe(ctx fiber.Ctx, namespace string) error {
	h, err := s.availableHub()
	if err != nil {
		return err
	}
	return connect(ctx, h, namespace)
}

// Publish sends a message to all subscribers in the same namespace.
func (s *Server) Publish(msg Message) error {
	return s.publish(msg, true)
}

func (s *Server) publish(msg Message, cloneData bool) error {
	h, err := s.availableHub()
	if err != nil {
		return err
	}

	if cloneData {
		msg = msg.clone()
	}

	select {
	case <-h.shutdown:
		return ErrServerClosed
	case h.broadcast <- msg:
		return nil
	}
}

// PublishEvent publishes raw bytes with an event name.
func (s *Server) PublishEvent(namespace, event string, data []byte) error {
	return s.Publish(Message{
		Event:     event,
		Data:      data,
		Namespace: namespace,
	})
}

// PublishJSON marshals a payload and publishes it as SSE data.
func (s *Server) PublishJSON(namespace, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal SSE payload: %w", err)
	}

	return s.publish(Message{
		Event:     event,
		Data:      data,
		Namespace: namespace,
	}, false)
}

// Close shuts down the server and disconnects all subscribers.
func (s *Server) Close() {
	if s == nil {
		return
	}

	s.closeOnce.Do(func() {
		s.closed.Store(true)
		if s.hub != nil {
			s.hub.Shutdown()
		}
	})
}

func (s *Server) availableHub() (*hub, error) {
	if s == nil {
		return nil, ErrServerClosed
	}
	if s.closed.Load() {
		return nil, ErrServerClosed
	}
	if s.hub == nil {
		return nil, ErrServerNotInitialized
	}
	return s.hub, nil
}
