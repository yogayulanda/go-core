package httpclient

import (
	"time"

	"github.com/yogayulanda/go-core/resilience"
)

// ClientOptions holds the configuration for the HTTP Client.
type ClientOptions struct {
	Timeout          time.Duration
	RetryOptions     *resilience.RetryOptions
	CircuitBreaker   *resilience.CircuitBreakerOptions
	Debug            bool
	UserAgent        string
	EnableTracing    bool
}

// DefaultClientOptions provides sensible defaults for the HTTP client.
func DefaultClientOptions() ClientOptions {
	retryOpts := resilience.DefaultRetryOptions()
	cbOpts := resilience.DefaultCircuitBreakerOptions("http-client-cb")

	return ClientOptions{
		Timeout:        30 * time.Second,
		RetryOptions:   &retryOpts,
		CircuitBreaker: &cbOpts,
		Debug:          false,
		UserAgent:      "go-core-http-client/1.0",
		EnableTracing:  true,
	}
}

// Option is a functional option for configuring ClientOptions.
type Option func(*ClientOptions)

func WithTimeout(timeout time.Duration) Option {
	return func(o *ClientOptions) {
		o.Timeout = timeout
	}
}

func WithRetryOptions(opts resilience.RetryOptions) Option {
	return func(o *ClientOptions) {
		o.RetryOptions = &opts
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
