package sseserver

import (
	"bytes"
	"strings"
)

type Message struct {
	Event     string
	Data      []byte
	Namespace string
	Topic     string
}

// SSEMessage is kept as an alias for callers that still use the old type name.
type SSEMessage = Message

func (msg Message) clone(cloneData bool) Message {
	msg.Namespace = strings.Clone(msg.Namespace)
	msg.Topic = strings.Clone(msg.Topic)
	msg.Event = strings.Clone(msg.Event)
	if cloneData {
		msg.Data = bytes.Clone(msg.Data)
	}
	return msg
}

// sseFormat encodes a message into the SSE wire format.
func (msg Message) sseFormat() []byte {
	data := msg.Data
	if bytes.IndexByte(data, '\r') >= 0 {
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	}
	lineCount := bytes.Count(data, []byte{'\n'}) + 1
	b := make([]byte, 0, 7+len(msg.Event)+1+lineCount*6+len(data)+lineCount+1)

	if msg.Event != "" {
		b = append(b, "event: "...)
		b = append(b, msg.Event...)
		b = append(b, '\n')
	}

	for _, line := range bytes.Split(data, []byte{'\n'}) {
		b = append(b, "data: "...)
		b = append(b, line...)
		b = append(b, '\n')
	}

	b = append(b, '\n')
	return b
}
