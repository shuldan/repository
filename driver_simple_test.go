package repository

import (
	"context"
	"database/sql"
	sqlDriver "database/sql/driver"
	"errors"
	"fmt"
	"testing"
)

func TestSimpleDriver_CheckVersion_NoVersionCol(t *testing.T) {
	t.Parallel()
	d := &simpleDriver[string]{table: Table{Name: "t", PrimaryKey: []string{"id"}}}
	if err := d.checkVersion(&fakeResult{rowsAffected: 0}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestSimpleDriver_CheckVersion_RowsAffected(t *testing.T) {
	t.Parallel()
	d := &simpleDriver[string]{table: Table{Name: "t", PrimaryKey: []string{"id"}, VersionColumn: "v"}}
	if err := d.checkVersion(&fakeResult{rowsAffected: 1}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestSimpleDriver_CheckVersion_Zero(t *testing.T) {
	t.Parallel()
	d := &simpleDriver[string]{table: Table{Name: "t", PrimaryKey: []string{"id"}, VersionColumn: "v"}}
	if err := d.checkVersion(&fakeResult{rowsAffected: 0}); !errors.Is(err, ErrConcurrentModification) {
		t.Errorf("expected ErrConcurrentModification, got %v", err)
	}
}

func TestSimpleDriver_CheckVersion_Error(t *testing.T) {
	t.Parallel()
	d := &simpleDriver[string]{table: Table{Name: "t", PrimaryKey: []string{"id"}, VersionColumn: "v"}}
	if err := d.checkVersion(&fakeResult{err: fmt.Errorf("fail")}); err == nil {
		t.Error("expected error")
	}
}

func TestSimpleDriver_FindOne_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"abc"}}},
	}}
	db := newTestDB(t, conn)
	d := &simpleDriver[string]{table: simpleTable, dialect: Postgres(), scan: simpleScan}
	result, err := d.findOne(context.Background(), db, "SELECT id FROM items WHERE id=$1", []any{"abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "abc" {
		t.Errorf("expected 'abc', got %q", result)
	}
}

func TestSimpleDriver_FindOne_NoRows(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: nil},
	}}
	db := newTestDB(t, conn)
	d := &simpleDriver[string]{table: simpleTable, dialect: Postgres(), scan: simpleScan}
	_, err := d.findOne(context.Background(), db, "SELECT id FROM items WHERE id=$1", []any{"x"})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestSimpleDriver_FindMany_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}, {"b"}, {"c"}}},
	}}
	db := newTestDB(t, conn)
	d := &simpleDriver[string]{table: simpleTable, dialect: Postgres(), scan: simpleScan}
	items, err := d.findMany(context.Background(), db, "SELECT id FROM items", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}

func TestSimpleDriver_FindMany_Empty(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: nil},
	}}
	db := newTestDB(t, conn)
	d := &simpleDriver[string]{table: simpleTable, dialect: Postgres(), scan: simpleScan}
	items, err := d.findMany(context.Background(), db, "SELECT id FROM items", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestSimpleDriver_FindMany_QueryError(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("query fail")}}}
	db := newTestDB(t, conn)
	d := &simpleDriver[string]{table: simpleTable, dialect: Postgres(), scan: simpleScan}
	_, err := d.findMany(context.Background(), db, "SELECT id FROM items", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestSimpleDriver_Save_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	d := &simpleDriver[string]{table: simpleTable, dialect: Postgres(), scan: simpleScan, values: simpleValues}
	err := d.save(context.Background(), nil, db, "val")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSimpleDriver_Save_ExecError(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{err: fmt.Errorf("exec fail")}}}
	db := newTestDB(t, conn)
	d := &simpleDriver[string]{table: simpleTable, dialect: Postgres(), scan: simpleScan, values: simpleValues}
	err := d.save(context.Background(), nil, db, "val")
	if err == nil {
		t.Error("expected error")
	}
}

func TestSimpleDriver_Save_VersionConflict(t *testing.T) {
	t.Parallel()
	tbl := Table{Name: "t", PrimaryKey: []string{"id"}, Columns: []string{"id"}, VersionColumn: "v"}
	conn := &testConn{execs: []testExecResult{{rowsAffected: 0}}}
	db := newTestDB(t, conn)
	d := &simpleDriver[string]{table: tbl, dialect: Postgres(), scan: simpleScan, values: simpleValues}
	err := d.save(context.Background(), nil, db, "val")
	if !errors.Is(err, ErrConcurrentModification) {
		t.Errorf("expected ErrConcurrentModification, got %v", err)
	}
}

func TestSimpleDriver_Delete_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	d := &simpleDriver[string]{table: simpleTable, dialect: Postgres()}
	var ids []any
	ids = append(ids, "id1")
	err := d.delete(context.Background(), nil, db, ids)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
