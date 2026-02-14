package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

func TestInTx_BeginError(t *testing.T) {
	t.Parallel()
	fb := &fakeTxBeginner{beginErr: fmt.Errorf("begin failed")}
	err := inTx(context.Background(), fb, func(tx *sql.Tx) error {
		return nil
	})
	if err == nil || err.Error() != "begin failed" {
		t.Errorf("expected 'begin failed', got %v", err)
	}
}

func TestInTx_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	called := false
	err := inTx(context.Background(), db, func(tx *sql.Tx) error {
		called = true
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("fn was not called")
	}
}

func TestInTx_FnError(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	fnErr := fmt.Errorf("fn failed")
	err := inTx(context.Background(), db, func(tx *sql.Tx) error {
		return fnErr
	})
	if err != fnErr {
		t.Errorf("expected fn error, got %v", err)
	}
}

func TestInTx_CommitError(t *testing.T) {
	t.Parallel()
	conn := &testConn{commitErr: fmt.Errorf("commit failed")}
	db := newTestDB(t, conn)
	err := inTx(context.Background(), db, func(tx *sql.Tx) error {
		return nil
	})
	if err == nil {
		t.Error("expected commit error")
	}
}
