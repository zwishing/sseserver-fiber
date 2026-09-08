package sseserver

import (
	"bufio"
	"sync"
	"testing"
)

func TestHubDoesNotRetainAlreadyClosedSubscriber(t *testing.T) {
	h := newHub(defaultConfig())
	defer h.Shutdown()
	closed := newConnection(nil, h, "test", "")
	_ = closed.Close()
	h.Start()
	h.register <- closed
	// Accepting another registration proves the first one has been processed.
	barrier := newConnection(nil, h, "test", "")
	h.register <- barrier
	if _, exists := h.connections.Load(closed); exists {
		t.Fatal("hub retained a response that closed before registration completed")
	}
}

func TestHubRegisterAndCloseRace(t *testing.T) {
	h := newHub(defaultConfig())
	h.Start()
	defer h.Shutdown()
	var closers sync.WaitGroup
	for i := 0; i < 2000; i++ {
		c := newConnection(nil, h, "test", "")
		closers.Go(func() { _ = c.Close() })
		h.register <- c
	}
	closers.Wait()
	barrier := newConnection(nil, h, "test", "")
	h.register <- barrier
	h.connections.Range(func(key, _ any) bool {
		if key != barrier {
			t.Error("concurrent registration retained a closed subscriber")
		}
		return true
	})
}

func TestSlowSubscriberDoesNotBlockHealthySubscriber(t *testing.T) {
	cfg := defaultConfig()
	cfg.connectionBuffer = 1
	h := newHub(cfg)
	slow := newConnection(nil, h, "test", "")
	healthy := newConnection(nil, h, "test", "")
	defer slow.Close()
	defer healthy.Close()
	h.connections.Store(slow, true)
	h.connections.Store(healthy, true)
	reader := bufio.NewReader(healthy)
	_, _ = reader.ReadString('\n') // Initial connection comment.
	for _, payload := range []string{"first", "second"} {
		h._broadcastMessage(Message{Namespace: "test", Data: []byte(payload)})
		line, err := reader.ReadString('\n')
		if err != nil || line != "data: "+payload+"\n" {
			t.Fatalf("healthy subscriber: line=%q err=%v", line, err)
		}
		_, _ = reader.ReadString('\n')
	}
	if _, exists := h.connections.Load(slow); exists {
		t.Fatal("full subscriber queue was not evicted")
	}
	select {
	case <-slow.done:
	default:
		t.Fatal("slow subscriber was removed without canceling its stream")
	}
}
