package httpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http/httptrace"
	"net/url"
	"sync/atomic"
)

type Phase string

type Kind string

const (
	PhaseBeforeSend Phase = "before_send"
	PhaseAfterSend  Phase = "after_send"

	TransportKindTimeout     Kind = "timeout"
	TransportKindConnection  Kind = "connection"
	TransportKindTLS         Kind = "tls"
	TransportKindDNS         Kind = "dns"
	TransportKindCircuitOpen Kind = "circuit_open"
	TransportKindCanceled    Kind = "canceled"
	TransportKindReadBody    Kind = "read_body"
	TransportKindOther       Kind = "other"
)

type TransportError struct {
	Phase Phase
	Kind  Kind
	Err   error
}

func (e *TransportError) Error() string {
	if e == nil || e.Err == nil {
		return "http transport error"
	}
	return e.Err.Error()
}

func (e *TransportError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type httpTraceStateKey struct{}

type httpTraceState struct {
	wroteHeaders atomic.Bool
}

func newHTTPTrace(state *httpTraceState) *httptrace.ClientTrace {
	return &httptrace.ClientTrace{WroteHeaders: func() { state.wroteHeaders.Store(true) }}
}

func httptraceContext(ctx context.Context, trace *httptrace.ClientTrace) context.Context {
	return httptrace.WithClientTrace(ctx, trace)
}

func transportPhase(wroteHeaders bool) Phase {
	if wroteHeaders {
		return PhaseAfterSend
	}
	return PhaseBeforeSend
}

func classifyTransportKind(err error) Kind {
	if err == nil {
		return TransportKindOther
	}
	if errors.Is(err, context.Canceled) {
		return TransportKindCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return TransportKindTimeout
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return TransportKindDNS
	}
	var certErr x509.UnknownAuthorityError
	var hostnameErr x509.HostnameError
	var invalidErr x509.CertificateInvalidError
	var tlsHeaderErr tls.RecordHeaderError
	var tlsVerifyErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) || errors.As(err, &hostnameErr) || errors.As(err, &invalidErr) || errors.As(err, &tlsHeaderErr) || errors.As(err, &tlsVerifyErr) {
		return TransportKindTLS
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return TransportKindReadBody
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return classifyTransportKind(urlErr.Err)
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return TransportKindConnection
	}
	var timeoutErr net.Error
	if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
		return TransportKindTimeout
	}
	return TransportKindOther
}
