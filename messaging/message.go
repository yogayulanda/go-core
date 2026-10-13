package messaging

import (
	"context"
	"time"
)

type Message struct {
	Topic   string
	Key     []byte
	Payload []byte
	Headers map[string]string
	EventID string
}

type CloudEventAttributes struct {
	SpecVersion string
	ID          string
	Type        string
	Source      string
	Time        time.Time
	Subject     string
	Sequence    string
}

type cloudEventContextKey struct{}

func CloudEventsFromContext(ctx context.Context) (CloudEventAttributes, bool) {
	if ctx == nil {
		return CloudEventAttributes{}, false
	}
	event, ok := ctx.Value(cloudEventContextKey{}).(CloudEventAttributes)
	return event, ok
}
