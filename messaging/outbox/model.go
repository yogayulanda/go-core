package outbox

import "time"

type Event struct {
	ID          int64
	EventID     string
	Topic       string
	Key         []byte
	Payload     []byte
	Headers     []byte
	Status      string
	Attempt     int
	LastError   string
	AvailableAt time.Time
	LockedUntil *time.Time
	CreatedAt   time.Time
	SentAt      *time.Time
}
