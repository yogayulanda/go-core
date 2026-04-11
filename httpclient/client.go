package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/resilience"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

// Client is a wrapper around resty.Client that provides tracing, logging, and resilience.
type Client struct {
	client *resty.Client
	cb     *resilience.CircuitBreaker
	opts   ClientOptions
}

// NewClient initializes a new HTTP client with the provided options.
func NewClient(options ...Option) *Client {
	opts := DefaultClientOptions()
	for _, opt := range options {
		opt(&opts)
	}

	restyClient := resty.New()
	restyClient.SetTimeout(opts.Timeout)
	restyClient.SetHeader("User-Agent", opts.UserAgent)

	if opts.Debug {
		restyClient.SetDebug(true)
	}

	if opts.EnableTracing {
		// Use otelhttp transport to automatically propagate tracing headers
		transport := otelhttp.NewTransport(http.DefaultTransport)
		restyClient.SetTransport(transport)
	}

	if opts.RetryOptions != nil {
		restyClient.SetRetryCount(opts.RetryOptions.MaxAttempts)
		restyClient.SetRetryWaitTime(opts.RetryOptions.BaseDelay)
		restyClient.SetRetryMaxWaitTime(opts.RetryOptions.MaxDelay)
		restyClient.AddRetryCondition(func(r *resty.Response, err error) bool {
			// Leverage resilience retryable logic
			if opts.RetryOptions.Retryable != nil && err != nil {
				return opts.RetryOptions.Retryable(err)
			}
			return r.StatusCode() >= 500 || r.StatusCode() == 429
		})
	}

	restyClient.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		reqCtx := req.Context()
		if reqCtx == nil {
			reqCtx = context.Background()
		}

		logger.ServiceLog(reqCtx, "httpclient_request",
			zap.String("method", req.Method),
			zap.String("url", req.URL),
		)
		return nil
	})

	restyClient.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		reqCtx := resp.Request.Context()
		if reqCtx == nil {
			reqCtx = context.Background()
		}

		logger.ServiceLog(reqCtx, "httpclient_response",
			zap.String("method", resp.Request.Method),
			zap.String("url", resp.Request.URL),
			zap.Int("status_code", resp.StatusCode()),
			zap.Duration("duration", resp.Time()),
		)
		return nil
	})

	var cb *resilience.CircuitBreaker
	if opts.CircuitBreaker != nil {
		cb = resilience.NewCircuitBreaker(*opts.CircuitBreaker)
	}

	return &Client{
		client: restyClient,
		cb:     cb,
		opts:   opts,
	}
}

// Request returns a new resty.Request that you can use to perform HTTP calls synchronously.
// If a Context is provided, it will be executed within the CircuitBreaker wrapper (if configured).
func (c *Client) Request() *resty.Request {
	return c.client.R()
}

// Do executes a Resty request synchronously using context and wrapped inside CircuitBreaker.
func (c *Client) Do(ctx context.Context, req *resty.Request, method, url string) (*resty.Response, error) {
	req.SetContext(ctx)

	var resp *resty.Response
	var err error

	fn := func(ctx context.Context) error {
		resp, err = req.Execute(method, url)
		if err != nil {
			return err
		}
		if resp.StatusCode() >= 500 {
			// Trigger circuit breaker on server errors
			return fmt.Errorf("http client error: %d", resp.StatusCode())
		}
		return nil
	}

	if c.cb != nil {
		err = c.cb.Do(ctx, fn)
	} else {
		err = fn(ctx)
	}

	return resp, err
}

// Get is a convenience method for simple GET requests.
func (c *Client) Get(ctx context.Context, url string) (*resty.Response, error) {
	return c.Do(ctx, c.Request(), resty.MethodGet, url)
}

// Post is a convenience method for simple POST requests.
func (c *Client) Post(ctx context.Context, url string, body interface{}) (*resty.Response, error) {
	req := c.Request().SetBody(body)
	return c.Do(ctx, req, resty.MethodPost, url)
}
