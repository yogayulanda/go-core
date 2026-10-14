package httpclient

import (
	"time"

	"github.com/yogayulanda/go-core/resilience"
)

// ClientOptions holds the configuration for the HTTP Client.
type ClientOptions struct {
	Timeout             time.Duration
	IdleConnTimeout     time.Duration
	RetryOptions        *resilience.RetryOptions
	CircuitBreaker      *resilience.CircuitBreakerOptions
	Debug               bool
	UserAgent           string
	EnableTracing       bool
	InternalPropagation bool
	BodyLogging         bool
	BodyMasker          func([]byte) []byte
}

// DefaultClientOptions provides sensible defaults for the HTTP client.
func DefaultClientOptions() ClientOptions {
	cbOpts := resilience.DefaultCircuitBreakerOptions("http-client-cb")

	return ClientOptions{
		Timeout:         30 * time.Second,
		IdleConnTimeout: 30 * time.Second,
		RetryOptions:    nil,
		CircuitBreaker:  &cbOpts,
		Debug:           false,
		UserAgent:       "go-core-http-client/1.0",
		EnableTracing:   true,
	}
}

// Option is a functional option for configuring ClientOptions.
type Option func(*ClientOptions)

func WithTimeout(timeout time.Duration) Option {
	return func(o *ClientOptions) {
		o.Timeout = timeout
	}
}

func WithIdleConnTimeout(timeout time.Duration) Option {
	return func(o *ClientOptions) {
		o.IdleConnTimeout = timeout
	}
}

func WithRetry(opts resilience.RetryOptions) Option {
	return func(o *ClientOptions) {
		o.RetryOptions = &opts
	}
}

func WithInternalPropagation() Option {
	return func(o *ClientOptions) {
		o.InternalPropagation = true
	}
}

func WithBodyLogging(masker func([]byte) []byte) Option {
	return func(o *ClientOptions) {
		if masker == nil {
			o.BodyLogging = false
			o.BodyMasker = nil
			return
		}
		o.BodyLogging = true
		o.BodyMasker = masker
	}
}

func WithCircuitBreaker(opts resilience.CircuitBreakerOptions) Option {
	return func(o *ClientOptions) {
		o.CircuitBreaker = &opts
	}
}

func WithDebug(debug bool) Option {
	return func(o *ClientOptions) {
		o.Debug = debug
	}
}

func WithUserAgent(agent string) Option {
	return func(o *ClientOptions) {
		o.UserAgent = agent
	}
}

func WithoutTracing() Option {
	return func(o *ClientOptions) {
		o.EnableTracing = false
	}
}
