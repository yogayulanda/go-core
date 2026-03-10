package messaging

import "context"

type Handler func(ctx context.Context, msg Message) error

type Consumer interface {
	Start(ctx context.Context) error
	Close() error
}
