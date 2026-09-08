package sseserver

import (
	"errors"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

// Decode SSE fields independently of the encoder, including the optional space
// after a colon and all three line endings defined by the SSE protocol.
func decodeEvents(wire string) (events, payloads []string) {
	wire = strings.ReplaceAll(strings.ReplaceAll(wire, "\r\n", "\n"), "\r", "\n")
	event, data := "", ""
	for _, line := range strings.Split(wire, "\n") {
		if line == "" {
			if data != "" {
				events = append(events, event)
				payloads = append(payloads, strings.TrimSuffix(data, "\n"))
			}
			event, data = "", ""
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			event = value
		case "data":
			data += value + "\n"
		}
	}
	return
}

func TestMessageFormatPreservesContent(t *testing.T) {
	for _, tt := range []struct{ name, input, want string }{
		{"empty", "", ""},
		{"spaces", " hello\n  world", " hello\n  world"},
		{"LF", "a\nb", "a\nb"},
		{"CR", "a\rb", "a\nb"},
		{"CRLF", "a\r\nb", "a\nb"},
		{"blank lines", "\r\na\r\r\nb\n", "\na\n\nb\n"},
		{"field injection", "hello\revent: forged\r\rdata: other", "hello\nevent: forged\n\ndata: other"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			events, data := decodeEvents(string((Message{Event: " update", Data: []byte(tt.input)}).sseFormat()))
			if len(data) != 1 || data[0] != tt.want || events[0] != " update" {
				t.Fatalf("events=%q data=%q; want one event %q with data %q", events, data, " update", tt.want)
			}
		})
	}
}

func TestPublishRejectsEventNewlines(t *testing.T) {
	s := New()
	defer s.Close()
	for _, event := range []string{"update\revent:forged", "update\ndata:forged\n\nevent:admin"} {
		for _, publish := range []func() error{
			func() error { return s.Publish(Message{Event: event}) },
			func() error { return s.PublishEvent("ns", event, nil) },
			func() error { return s.PublishEventWithTopic("ns", "topic", event, nil) },
			func() error { return s.PublishJSON("ns", event, nil) },
			func() error { return s.PublishJSONWithTopic("ns", "topic", event, nil) },
		} {
			if err := publish(); !errors.Is(err, ErrInvalidEventName) {
				t.Errorf("event %q: error=%v, want ErrInvalidEventName", event, err)
			}
		}
	}
}

func TestPublishOwnsRequestStrings(t *testing.T) {
	for _, jsonPayload := range []bool{false, true} {
		// A stopped hub makes queue inspection deterministic, without racing its reader.
		s := &Server{hub: newHub(defaultConfig())}
		app := fiber.New()
		request := &fasthttp.RequestCtx{}
		request.Request.SetRequestURI("/original")
		ctx := app.AcquireCtx(request)
		key := ctx.Path()
		data := []byte("original")
		var err error
		if jsonPayload {
			err = s.PublishJSONWithTopic(key, key, key, "original")
		} else {
			err = s.Publish(Message{Namespace: key, Topic: key, Event: key, Data: data})
		}
		if err != nil {
			t.Fatal(err)
		}
		ctx.Path("/modified")
		app.ReleaseCtx(ctx)
		copy(data, "modified")
		got := <-s.hub.broadcast
		if got.Namespace != "/original" || got.Topic != "/original" || got.Event != "/original" {
			t.Errorf("json=%v: queued routing/event strings changed: %+v", jsonPayload, got)
		}
		want := "original"
		if jsonPayload {
			want = `"original"`
		}
		if string(got.Data) != want {
			t.Errorf("json=%v: queued data=%q, want %q", jsonPayload, got.Data, want)
		}
	}
}
