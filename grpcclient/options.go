package grpcclient

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type Options struct {
	ServiceName          string
	DefaultDeadline      time.Duration
	Keepalive            keepalive.ClientParameters
	DialOptions          []grpc.DialOption
	TransportCredentials credentials.TransportCredentials
	MetricsEnabled       bool
}

type Option func(*Options)

func defaultOptions() Options {
	return Options{
		ServiceName:          "grpc-client",
		DefaultDeadline:      30 * time.Second, // Default deadline for gRPC calls
		TransportCredentials: insecure.NewCredentials(),
		Keepalive: keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: false,
		},
	}
}

func WithServiceName(name string) Option {
	return func(options *Options) { options.ServiceName = name }
}

func WithDefaultDeadline(timeout time.Duration) Option {
	return func(options *Options) { options.DefaultDeadline = timeout }
}

func WithKeepalive(parameters keepalive.ClientParameters) Option {
	return func(options *Options) { options.Keepalive = parameters }
}

func WithDialOptions(options ...grpc.DialOption) Option {
	return func(config *Options) { config.DialOptions = append(config.DialOptions, options...) }
}

func WithTransportCredentials(credentials credentials.TransportCredentials) Option {
	return func(options *Options) { options.TransportCredentials = credentials }
}

func WithMetricsEnabled(enabled bool) Option {
	return func(options *Options) { options.MetricsEnabled = enabled }
}
