package outbox

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"

	"github.com/yogayulanda/go-core/messaging"
	"github.com/yogayulanda/go-core/observability"
)

func BenchmarkPublishTxInMemoryStore(b *testing.B) {
	store := &outboxBenchmarkStore{}
	db := sql.OpenDB(outboxBenchmarkConnector{store: store})
	db.SetMaxOpenConns(1)
	b.Cleanup(func() { _ = db.Close() })

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = tx.Rollback() })

	ctx := observability.EnsureFieldBag(context.Background())
	ctx = observability.WithRequestID(ctx, "benchmark-request-1")
	if err := observability.SetUserID(ctx, "benchmark-user-1"); err != nil {
		b.Fatal(err)
	}
	publisher := NewPublisher(WithPublisherDriver("mysql"))
	message := messaging.Message{
		Topic:   "benchmark-topic",
		Key:     []byte("benchmark-key"),
		Payload: []byte(`{"event":"benchmark"}`),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := publisher.PublishTx(ctx, tx, message); err != nil {
			b.Fatal(err)
		}
	}
}

type outboxBenchmarkStore struct {
	mu    sync.Mutex
	rows  int64
	bytes int64
}

type outboxBenchmarkConnector struct {
	store *outboxBenchmarkStore
}

func (c outboxBenchmarkConnector) Connect(context.Context) (driver.Conn, error) {
	return outboxBenchmarkConn(c), nil
}

func (c outboxBenchmarkConnector) Driver() driver.Driver {
	return outboxBenchmarkDriver(c)
}

type outboxBenchmarkDriver struct {
	store *outboxBenchmarkStore
}

func (d outboxBenchmarkDriver) Open(string) (driver.Conn, error) {
	return outboxBenchmarkConn(d), nil
}

type outboxBenchmarkConn struct {
	store *outboxBenchmarkStore
}

func (outboxBenchmarkConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepared statements are not supported")
}

func (outboxBenchmarkConn) Close() error { return nil }

func (c outboxBenchmarkConn) Begin() (driver.Tx, error) {
	return outboxBenchmarkTx{}, nil
}

func (c outboxBenchmarkConn) BeginTx(ctx context.Context, _ driver.TxOptions) (driver.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return outboxBenchmarkTx{}, nil
}

func (c outboxBenchmarkConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if query == "" || len(args) == 0 {
		return nil, errors.New("invalid outbox insert")
	}
	bytes := int64(len(query))
	for _, arg := range args {
		if value, ok := arg.Value.([]byte); ok {
			bytes += int64(len(value))
		}
	}
	c.store.mu.Lock()
	c.store.rows++
	c.store.bytes += bytes
	c.store.mu.Unlock()
	return driver.RowsAffected(1), nil
}

type outboxBenchmarkTx struct{}

func (outboxBenchmarkTx) Commit() error   { return nil }
func (outboxBenchmarkTx) Rollback() error { return nil }
