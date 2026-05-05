package resilience

import (
	"context"
	"errors"
	"time"

	"github.com/sony/gobreaker/v2"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrTooManyRequests is returned when the circuit breaker is half-open and the max requests have been reached.
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

type CircuitBreakerOptions struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	ReadyToTrip   func(counts gobreaker.Counts) bool
	OnStateChange func(name string, from gobreaker.State, to gobreaker.State)
}

func DefaultCircuitBreakerOptions(name string) CircuitBreakerOptions {
	return CircuitBreakerOptions{
		Name:        name,
		MaxRequests: 1,                // Requests allowed in half-open state
		Interval:    time.Duration(0), // Never clear counts if 0
		Timeout:     60 * time.Second, // Time sitting in open before half-open transition
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip if we have more than 5 consecutive failures
			return counts.ConsecutiveFailures > 5
		},
	}
}

// CircuitBreaker wraps gobreaker.CircuitBreaker to provide a context-aware generic implementation.
type CircuitBreaker struct {
	cb *gobreaker.CircuitBreaker[any]
}

// NewCircuitBreaker initializes a new primitive circuit breaker.
func NewCircuitBreaker(opts CircuitBreakerOptions) *CircuitBreaker {
	if opts.Name == "" {
		opts.Name = "go-core-circuitbreaker"
	}

	settings := gobreaker.Settings{
		Name:          opts.Name,
		MaxRequests:   opts.MaxRequests,
		Interval:      opts.Interval,
		Timeout:       opts.Timeout,
		ReadyToTrip:   opts.ReadyToTrip,
		OnStateChange: opts.OnStateChange,
	}

	cb := gobreaker.NewCircuitBreaker[any](settings)
	return &CircuitBreaker{cb: cb}
}

// Do executes the given function safely inside the circuit breaker.
func (cb *CircuitBreaker) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if cb == nil || cb.cb == nil {
		return fn(ctx)
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	_, err := cb.cb.Execute(func() (any, error) {
		err := fn(ctx)
		if err != nil {
			// If context is canceled, do not count it as a target failure
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, nil // Ignored failure for cb counting
			}
		}
		return nil, err
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			return ErrCircuitOpen
		}
		if errors.Is(err, gobreaker.ErrTooManyRequests) {
			return ErrTooManyRequests
		}
		return err
	}

	return nil
}
