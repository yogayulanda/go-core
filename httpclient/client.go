package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
	"github.com/yogayulanda/go-core/resilience"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Client is a wrapper around resty.Client that provides tracing, logging, and resilience.
type Client struct {
	client *resty.Client
	cb     *resilience.CircuitBreaker
	opts   ClientOptions
	log    logger.Logger
}

// NewClient initializes a new HTTP client with the provided options.
func NewClient(log logger.Logger, options ...Option) *Client {
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
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		baseTransport = &http.Transport{}
	}
	transport := baseTransport.Clone()
	transport.IdleConnTimeout = opts.IdleConnTimeout
	var roundTripper http.RoundTripper = transport

	if opts.EnableTracing {
		roundTripper = otelhttp.NewTransport(roundTripper)
	}
	restyClient.SetTransport(roundTripper)

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

	restyClient.OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
		if state, ok := req.Context().Value(httpTraceStateKey{}).(*httpTraceState); ok {
			state.wroteHeaders.Store(false)
			trace := newHTTPTrace(state)
			req.SetContext(httptraceContext(req.Context(), trace))
		}
		if opts.InternalPropagation {
			propagateIdentity(req, req.Context())
		}
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
		log:    log,
	}
}

// Request returns a new resty.Request that you can use to perform HTTP calls synchronously.
// If a Context is provided, it will be executed within the CircuitBreaker wrapper (if configured).
func (c *Client) Request() *resty.Request {
	return c.client.R()
}

// Do executes a Resty request synchronously using context and wrapped inside CircuitBreaker.
func (c *Client) Do(ctx context.Context, req *resty.Request, method, url string) (*resty.Response, error) {
	ctx = observability.EnsureFieldBag(ctx)
	traceState := &httpTraceState{}
	ctx = context.WithValue(ctx, httpTraceStateKey{}, traceState)
	req.SetContext(ctx)
	start := time.Now()

	var resp *resty.Response
	var err error
	var transportErr error
	var receivedServerError bool

	fn := func(ctx context.Context) error {
		resp, err = req.Execute(method, url)
		if err != nil {
			transportErr = err
			return err
		}
		if resp.StatusCode() >= 500 {
			receivedServerError = true
			return fmt.Errorf("http client error: %d", resp.StatusCode())
		}
		return nil
	}

	if c.cb != nil {
		err = c.cb.Do(ctx, fn)
	} else {
		err = fn(ctx)
	}
	if transportErr != nil {
		err = transportErr
	}

	if receivedServerError && transportErr == nil && resp != nil {
		err = nil
	}
	var resultErr error
	if err != nil {
		var kind Kind
		if errors.Is(err, resilience.ErrCircuitOpen) || errors.Is(err, resilience.ErrTooManyRequests) {
			kind = TransportKindCircuitOpen
			transportErr = err
		} else {
			if transportErr == nil {
				transportErr = err
			}
			kind = classifyTransportKind(transportErr)
		}
		resultErr = &TransportError{Phase: transportPhase(traceState.wroteHeaders.Load()), Kind: kind, Err: transportErr}
	}
	if resultErr != nil && resp != nil && resp.RawResponse == nil {
		resp = nil
	}
	logHTTPCall(c.log, ctx, req, method, url, resp, resultErr, time.Since(start), c.opts)
	return resp, resultErr
}

func propagateIdentity(req *resty.Request, ctx context.Context) {
	if requestID := observability.GetRequestID(ctx); requestID != "" && req.Header.Get("X-Request-ID") == "" {
		req.SetHeader("X-Request-ID", requestID)
	}
	if userID := observability.GetUserID(ctx); userID != "" && req.Header.Get("X-User-ID") == "" {
		req.SetHeader("X-User-ID", userID)
	}
	labels := observability.Labels(ctx)
	for _, name := range observability.PropagatedLabels() {
		if value := labels[name]; value != "" && req.Header.Get(observability.LabelMetadataPrefix+name) == "" {
			req.SetHeader(observability.LabelMetadataPrefix+name, value)
		}
	}
}

func logHTTPCall(log logger.Logger, ctx context.Context, req *resty.Request, method, rawURL string, resp *resty.Response, callErr error, duration time.Duration, opts ClientOptions) {
	status := "failed"
	metadata := map[string]interface{}{"method": method, "url": sanitizedURL(rawURL), "duration_ms": duration.Milliseconds()}
	if resp != nil {
		metadata["status_code"] = resp.StatusCode()
		if resp.StatusCode() < http.StatusBadRequest {
			status = "success"
		}
	}
	if callErr != nil {
		status = "failed"
		var transportError *TransportError
		if errors.As(callErr, &transportError) {
			metadata["phase"] = string(transportError.Phase)
			metadata["kind"] = string(transportError.Kind)
		}
	}
	if opts.BodyLogging && opts.BodyMasker != nil {
		requestBody := req.Body
		if resp != nil && resp.Request != nil {
			requestBody = resp.Request.Body
		}
		if body := maskedBody(requestBody, opts.BodyMasker); body != "" {
			metadata["request_body"] = body
		}
		if resp != nil {
			if body := string(opts.BodyMasker(append([]byte(nil), resp.Body()...))); body != "" {
				metadata["response_body"] = body
			}
		}
	}
	log.LogService(ctx, logger.ServiceLog{Operation: "httpclient_call", Status: status, DurationMs: duration.Milliseconds(), Metadata: metadata})
}

func sanitizedURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "<invalid-url>"
	}
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	parsed.User = nil
	return parsed.String()
}

func maskedBody(value interface{}, masker func([]byte) []byte) string {
	var raw []byte
	switch body := value.(type) {
	case nil:
		return ""
	case []byte:
		raw = append([]byte(nil), body...)
	case string:
		raw = []byte(body)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return ""
		}
		raw = encoded
	}
	return string(masker(raw))
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
