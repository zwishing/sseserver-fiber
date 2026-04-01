package sseserver

import (
	"errors"
	"testing"
	"time"
)

func TestNewAppliesOptions(t *testing.T) {
	s := New(
		WithConnectionBuffer(64),
		WithPublishBuffer(32),
		WithKeepAliveInterval(3*time.Second),
	)
	defer s.Close()

	if got := s.hub.config.connectionBuffer; got != 64 {
		t.Fatalf("connection buffer = %d, want 64", got)
	}

	if got := s.hub.config.publishBuffer; got != 32 {
		t.Fatalf("publish buffer = %d, want 32", got)
	}

	if got := cap(s.hub.broadcast); got != 32 {
		t.Fatalf("broadcast channel capacity = %d, want 32", got)
	}

	if got := s.hub.config.keepAlive; got != 3*time.Second {
		t.Fatalf("keepalive = %s, want 3s", got)
	}
}

func TestServerPublishAfterClose(t *testing.T) {
	s := New()
	s.Close()

	err := s.Publish(Message{
		Namespace: "progress",
		Data:      []byte("1%"),
	})
	if !errors.Is(err, ErrServerClosed) {
		t.Fatalf("Publish() error = %v, want %v", err, ErrServerClosed)
	}
}

func TestZeroValueServerReturnsInitializationError(t *testing.T) {
	var s Server

	if err := s.Publish(Message{Namespace: "progress", Data: []byte("1%")}); !errors.Is(err, ErrServerNotInitialized) {
		t.Fatalf("Publish() error = %v, want %v", err, ErrServerNotInitialized)
	}

	if err := s.Subscribe(nil, "progress"); !errors.Is(err, ErrServerNotInitialized) {
		t.Fatalf("Subscribe() error = %v, want %v", err, ErrServerNotInitialized)
	}

	if err := s.SubscribeWithTopic(nil, "progress", "tenant-a"); !errors.Is(err, ErrServerNotInitialized) {
		t.Fatalf("SubscribeWithTopic() error = %v, want %v", err, ErrServerNotInitialized)
	}
}

func TestZeroValueServerCloseDoesNotPanic(t *testing.T) {
	var s Server
	s.Close()
}

func TestMessageFormatMultiline(t *testing.T) {
	msg := Message{
		Event: "update",
		Data:  []byte("line1\nline2"),
	}

	got := string(msg.sseFormat())
	want := "event:update\ndata:line1\ndata:line2\n\n"
	if got != want {
		t.Fatalf("sseFormat() = %q, want %q", got, want)
	}
}

func TestHubBroadcastMatchesNamespaceExactly(t *testing.T) {
	h := newHub(defaultConfig())
	exact := &connection{send: make(chan []byte, 1), namespace: "sse"}
	other := &connection{send: make(chan []byte, 1), namespace: "sse2"}

	h.connections.Store(exact, true)
	h.connections.Store(other, true)

	h._broadcastMessage(Message{
		Namespace: "sse",
		Data:      []byte("payload"),
	})

	select {
	case <-exact.send:
	default:
		t.Fatal("expected exact namespace subscriber to receive message")
	}

	select {
	case <-other.send:
		t.Fatal("unexpected message delivered to different namespace")
	default:
	}
}

func TestHubBroadcastMatchesNamespaceAndTopic(t *testing.T) {
	h := newHub(defaultConfig())
	match := &connection{send: make(chan []byte, 1), namespace: "sse", topic: "order"}
	otherTopic := &connection{send: make(chan []byte, 1), namespace: "sse", topic: "payment"}

	h.connections.Store(match, true)
	h.connections.Store(otherTopic, true)

	h._broadcastMessage(Message{
		Namespace: "sse",
		Topic:     "order",
		Data:      []byte("payload"),
	})

	select {
	case <-match.send:
	default:
		t.Fatal("expected same namespace/topic subscriber to receive message")
	}

	select {
	case <-otherTopic.send:
		t.Fatal("unexpected message delivered to different topic")
	default:
	}
}

func TestHubBroadcastWithoutTopicBroadcastsToNamespace(t *testing.T) {
	h := newHub(defaultConfig())
	firstTopic := &connection{send: make(chan []byte, 1), namespace: "sse", topic: "order"}
	secondTopic := &connection{send: make(chan []byte, 1), namespace: "sse", topic: "payment"}

	h.connections.Store(firstTopic, true)
	h.connections.Store(secondTopic, true)

	h._broadcastMessage(Message{
		Namespace: "sse",
		Data:      []byte("payload"),
	})

	select {
	case <-firstTopic.send:
	default:
		t.Fatal("expected namespace subscriber with topic order to receive message")
	}

	select {
	case <-secondTopic.send:
	default:
		t.Fatal("expected namespace subscriber with topic payment to receive message")
	}
}
