// Package coretest contains reusable test-only helpers for go-core consumers.
package coretest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	coreerrors "github.com/yogayulanda/go-core/errors"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
)

type logBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *logBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

type RecordingLogger struct {
	logger logger.Logger
	output *logBuffer
}

type Entry struct {
	Message string
	Fields  map[string]any
}

func NewRecordingLogger(options ...logger.Option) (*RecordingLogger, error) {
	output := &logBuffer{}
	options = append(options, logger.WithEnvironment("dev"), logger.WithOutput(output))
	log, err := logger.New("coretest", "debug", options...)
	if err != nil {
		return nil, err
	}
	return &RecordingLogger{logger: log, output: output}, nil
}

func (r *RecordingLogger) Info(ctx context.Context, message string, fields ...logger.Field) {
	r.logger.Info(ctx, message, fields...)
}
func (r *RecordingLogger) Error(ctx context.Context, message string, fields ...logger.Field) {
	r.logger.Error(ctx, message, fields...)
}
func (r *RecordingLogger) Debug(ctx context.Context, message string, fields ...logger.Field) {
	r.logger.Debug(ctx, message, fields...)
}
func (r *RecordingLogger) Warn(ctx context.Context, message string, fields ...logger.Field) {
	r.logger.Warn(ctx, message, fields...)
}
func (r *RecordingLogger) LogService(ctx context.Context, entry logger.ServiceLog) {
	r.logger.LogService(ctx, entry)
}
func (r *RecordingLogger) LogDB(ctx context.Context, entry logger.DBLog) { r.logger.LogDB(ctx, entry) }
func (r *RecordingLogger) LogEvent(ctx context.Context, entry logger.EventLog) {
	r.logger.LogEvent(ctx, entry)
}
func (r *RecordingLogger) LogTransaction(ctx context.Context, entry logger.TransactionLog) {
	r.logger.LogTransaction(ctx, entry)
}
func (r *RecordingLogger) WithComponent(component string) logger.Logger {
	return &recordingLoggerComponent{Logger: r.logger.WithComponent(component)}
}

func (r *RecordingLogger) Entries() []Entry {
	var entries []Entry
	for _, line := range bytes.Split([]byte(r.output.String()), []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var fields map[string]any
		if json.Unmarshal(line, &fields) == nil {
			message, _ := fields["message"].(string)
			entries = append(entries, Entry{Message: message, Fields: fields})
		}
	}
	return entries
}

func (r *RecordingLogger) Find(message string) []Entry {
	var matches []Entry
	for _, entry := range r.Entries() {
		if entry.Message == message || bytes.Contains([]byte(entry.Message), []byte(message)) {
			matches = append(matches, entry)
		}
	}
	return matches
}

func (r *RecordingLogger) Fields() []map[string]any {
	entries := r.Entries()
	fields := make([]map[string]any, len(entries))
	for index, entry := range entries {
		fields[index] = entry.Fields
	}
	return fields
}

type recordingLoggerComponent struct{ logger.Logger }

type FakePublisher struct {
	mu       sync.Mutex
	Messages []messaging.Message
	Err      error
}

func NewFakePublisher() *FakePublisher { return &FakePublisher{} }

func (p *FakePublisher) Publish(_ context.Context, message messaging.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	message.Headers = cloneHeaders(message.Headers)
	message.Key = append([]byte(nil), message.Key...)
	message.Payload = append([]byte(nil), message.Payload...)
	p.Messages = append(p.Messages, message)
	return p.Err
}
func (*FakePublisher) Close() error { return nil }

func cloneHeaders(headers map[string]string) map[string]string {
	cloned := make(map[string]string, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}

func WithIsolatedRegistry(t testing.TB) {
	t.Helper()
	state := coreerrors.SnapshotRegistry()
	t.Cleanup(func() { coreerrors.RestoreRegistry(state) })
}

func RefusingAddr(t testing.TB) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}

func SilentServer(t testing.TB) *httptest.Server {
	t.Helper()
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(func() { close(release); server.Close() })
	return server
}

func CloseAfterHeadersServer(t testing.TB) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijacking unsupported", http.StatusInternalServerError)
			return
		}
		connection, _, err := hijacker.Hijack()
		if err != nil {
			return
		}
		_, _ = fmt.Fprint(connection, "HTTP/1.1 200 OK\r\nContent-Length: 8\r\n\r\n")
		_ = connection.Close()
	}))
	t.Cleanup(server.Close)
	return server
}

func StatusServer(t testing.TB, status int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
	t.Cleanup(server.Close)
	return server
}

var _ logger.Logger = (*RecordingLogger)(nil)
var _ messaging.Publisher = (*FakePublisher)(nil)
