package dbtx

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"
)

func TestInjectFromContext(t *testing.T) {
	ctx := context.Background()
	tx, ok := FromContext(ctx)
	if ok || tx != nil {
		t.Fatalf("expected no tx in fresh context")
	}
}

func TestWithTx_NilArgs(t *testing.T) {
	err := WithTx(context.Background(), nil, func(context.Context) error { return nil })
	if err == nil {
		t.Fatalf("expected error for nil db")
	}

	db := openTestDB(t, &txDriverState{})
	t.Cleanup(func() { _ = db.Close() })
	err = WithTx(context.Background(), db, nil)
	if err == nil {
		t.Fatalf("expected error for nil fn")
	}
}

func TestWithTx_SuccessCommit(t *testing.T) {
	state := &txDriverState{}
	db := openTestDB(t, state)
	t.Cleanup(func() { _ = db.Close() })

	err := WithTx(context.Background(), db, func(txCtx context.Context) error {
		tx, ok := FromContext(txCtx)
		if !ok || tx == nil {
			t.Fatalf("expected tx in context")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.beginCount != 1 || state.commitCount != 1 || state.rollbackCount != 0 {
		t.Fatalf("unexpected counters: begin=%d commit=%d rollback=%d", state.beginCount, state.commitCount, state.rollbackCount)
	}
}

func TestWithTx_FnErrorRollback(t *testing.T) {
	state := &txDriverState{}
	db := openTestDB(t, state)
	t.Cleanup(func() { _ = db.Close() })

	expected := errors.New("fn failed")
	err := WithTx(context.Background(), db, func(context.Context) error { return expected })
	if !errors.Is(err, expected) {
		t.Fatalf("expected fn error, got %v", err)
	}
	if state.beginCount != 1 || state.commitCount != 0 || state.rollbackCount != 1 {
		t.Fatalf("unexpected counters: begin=%d commit=%d rollback=%d", state.beginCount, state.commitCount, state.rollbackCount)
	}
}

func TestWithTx_PanicRollback(t *testing.T) {
	state := &txDriverState{}
	db := openTestDB(t, state)
	t.Cleanup(func() { _ = db.Close() })

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic to propagate")
		}
		if state.rollbackCount != 1 {
			t.Fatalf("expected rollback on panic")
		}
	}()

	_ = WithTx(context.Background(), db, func(context.Context) error {
		panic("boom")
	})
}

func TestWithTx_CommitError(t *testing.T) {
	state := &txDriverState{commitErr: errors.New("commit failed")}
	db := openTestDB(t, state)
	t.Cleanup(func() { _ = db.Close() })

	err := WithTx(context.Background(), db, func(context.Context) error { return nil })
	if err == nil {
		t.Fatalf("expected commit error")
	}
}

var (
	registerOnce  sync.Once
	activeTxState *txDriverState
)

func openTestDB(t *testing.T, state *txDriverState) *sql.DB {
	t.Helper()

	registerOnce.Do(func() {
		sql.Register("dbtx_test_driver", txTestDriver{})
	})
	activeTxState = state

	db, err := sql.Open("dbtx_test_driver", "test")
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	return db
}

type txDriverState struct {
	beginCount    int
	commitCount   int
	rollbackCount int
	commitErr     error
	rollbackErr   error
}

type txTestDriver struct{}

func (txTestDriver) Open(string) (driver.Conn, error) {
	return &txTestConn{state: activeTxState}, nil
}

type txTestConn struct {
	state *txDriverState
}

func (c *txTestConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (c *txTestConn) Close() error                        { return nil }
func (c *txTestConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *txTestConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return nil, io.EOF
}
func (c *txTestConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}
func (c *txTestConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	c.state.beginCount++
	return &txTestTx{state: c.state}, nil
}

type txTestTx struct {
	state *txDriverState
}

func (t *txTestTx) Commit() error {
	t.state.commitCount++
	return t.state.commitErr
}

func (t *txTestTx) Rollback() error {
	t.state.rollbackCount++
	return t.state.rollbackErr
}
