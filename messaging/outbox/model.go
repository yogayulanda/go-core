package outbox

import "time"

type Event struct {
	ID        string
	Topic     string
	Key       []byte
	Payload   []byte
	Headers   []byte
	Status    string
	CreatedAt time.Time
	SentAt    *time.Time
}
