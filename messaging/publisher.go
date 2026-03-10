package messaging

import "context"

type Publisher interface {
	Publish(ctx context.Context, msg Message) error
	Close() error
}
