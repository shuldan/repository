package repository

import (
	"context"
	sqlDriver "database/sql/driver"
	"errors"
	"fmt"
	"testing"
)

func TestNew_CreatesRepository(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg))
	if repo == nil {
		t.Error("expected non-nil repository")
	}
}

func TestRepository_Find_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"abc"}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	result, err := repo.Find(context.Background(), "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "abc" {
		t.Errorf("expected 'abc', got %q", result)
	}
}

func TestRepository_Find_NotFound(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: nil},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	_, err := repo.Find(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRepository_Find_OtherError(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("db error")}}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	_, err := repo.Find(context.Background(), "x")
	if err == nil {
		t.Error("expected error")
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("should not be ErrNotFound")
	}
}

func TestRepository_FindBy_WithSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}, {"b"}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	items, err := repo.FindBy(context.Background(), Eq("id", "a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2, got %d", len(items))
	}
}

func TestRepository_FindBy_NilSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	items, err := repo.FindBy(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1, got %d", len(items))
	}
}

func TestRepository_ExistsBy_WithSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"exists"}, rows: [][]sqlDriver.Value{{true}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	exists, err := repo.ExistsBy(context.Background(), Eq("id", "a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true")
	}
}

func TestRepository_ExistsBy_NilSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"exists"}, rows: [][]sqlDriver.Value{{true}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	exists, err := repo.ExistsBy(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true")
	}
}

func TestRepository_CountBy_WithSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"count"}, rows: [][]sqlDriver.Value{{int64(5)}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	count, err := repo.CountBy(context.Background(), Eq("id", "a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5, got %d", count)
	}
}

func TestRepository_CountBy_NilSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"count"}, rows: [][]sqlDriver.Value{{int64(3)}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	count, err := repo.CountBy(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestRepository_Save_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	if err := repo.Save(context.Background(), "val"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRepository_Delete_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	if err := repo.Delete(context.Background(), "id"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRepository_WithSoftDelete_NoSoftDelete(t *testing.T) {
	t.Parallel()
	r := &Repository[string]{table: Table{Name: "t", PrimaryKey: "id"}}
	original := Eq("x", 1)
	if r.withSoftDelete(original) != original {
		t.Error("expected same spec")
	}
}

func TestRepository_WithSoftDelete_NilSpec(t *testing.T) {
	t.Parallel()
	r := &Repository[string]{table: Table{Name: "t", PrimaryKey: "id", SoftDelete: "del"}}
	result := r.withSoftDelete(nil)
	if result == nil {
		t.Fatal("expected non-nil")
	}
	sql, _, _ := result.ToSQL(Postgres(), 1)
	if sql != "del IS NULL" {
		t.Errorf("got %q", sql)
	}
}

func TestRepository_WithSoftDelete_WithSpec(t *testing.T) {
	t.Parallel()
	r := &Repository[string]{table: Table{Name: "t", PrimaryKey: "id", SoftDelete: "del"}}
	result := r.withSoftDelete(Eq("x", 1))
	if result == nil {
		t.Fatal("expected non-nil")
	}
}

func TestRepository_Query_ReturnsQuery(t *testing.T) {
	t.Parallel()
	r := &Repository[string]{
		table:   simpleTable,
		dialect: Postgres(),
	}
	q := r.Query(context.TODO())
	if q == nil || q.repo != r || !q.forward {
		t.Error("invalid query")
	}
}

func TestRepository_Find_WithSoftDelete(t *testing.T) {
	t.Parallel()
	tbl := Table{Name: "t", PrimaryKey: "id", Columns: []string{"id"}, SoftDelete: "del"}
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	repo := newSimpleTestRepo(t, conn, tbl)
	result, err := repo.Find(context.Background(), "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "a" {
		t.Errorf("expected 'a', got %q", result)
	}
}

func TestRepository_SaveTx_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg))
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := repo.SaveTx(context.Background(), tx, "val"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRepository_DeleteTx_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg))
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := repo.DeleteTx(context.Background(), tx, "id"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
