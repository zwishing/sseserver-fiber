package sseserver

import "bytes"

// Message is an SSE payload routed by namespace.
type Message struct {
	Event     string
	Data      []byte
	Namespace string
}

// SSEMessage is kept as an alias for callers that still use the old type name.
type SSEMessage = Message

func (msg Message) clone() Message {
	cloned := msg
	cloned.Data = bytes.Clone(msg.Data)
	return cloned
}

// sseFormat encodes a message into the SSE wire format.
func (msg Message) sseFormat() []byte {
	lineCount := bytes.Count(msg.Data, []byte{'\n'}) + 1
	b := make([]byte, 0, 6+len(msg.Event)+1+lineCount*5+len(msg.Data)+lineCount+1)

	if msg.Event != "" {
		b = append(b, "event:"...)
		b = append(b, msg.Event...)
		b = append(b, '\n')
	}

	for _, line := range bytes.Split(msg.Data, []byte{'\n'}) {
		b = append(b, "data:"...)
		b = append(b, line...)
		b = append(b, '\n')
	}

	b = append(b, '\n')
	return b
}
