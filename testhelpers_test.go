package repository

import (
	"context"
	"database/sql"
	sqlDriver "database/sql/driver"
	"fmt"
	"io"
	"sync"
	"testing"
)

type testQueryResult struct {
	columns []string
	rows    [][]sqlDriver.Value
	err     error
}

type testExecResult struct {
	lastInsertId int64
	rowsAffected int64
	err          error
}

type testConn struct {
	mu        sync.Mutex
	queries   []testQueryResult
	execs     []testExecResult
	qIdx      int
	eIdx      int
	beginErr  error
	commitErr error
}

func (c *testConn) Prepare(_ string) (sqlDriver.Stmt, error) {
	return &testStmt{conn: c}, nil
}

func (c *testConn) Close() error { return nil }

func (c *testConn) Begin() (sqlDriver.Tx, error) {
	if c.beginErr != nil {
		return nil, c.beginErr
	}
	return &testTxDriver{conn: c}, nil
}

type testStmt struct{ conn *testConn }

func (s *testStmt) Close() error  { return nil }
func (s *testStmt) NumInput() int { return -1 }

func (s *testStmt) Exec(_ []sqlDriver.Value) (sqlDriver.Result, error) {
	s.conn.mu.Lock()
	defer s.conn.mu.Unlock()
	if s.conn.eIdx >= len(s.conn.execs) {
		return nil, fmt.Errorf("no more exec results")
	}
	r := s.conn.execs[s.conn.eIdx]
	s.conn.eIdx++
	if r.err != nil {
		return nil, r.err
	}
	return &testDriverResult{lastID: r.lastInsertId, affected: r.rowsAffected}, nil
}

func (s *testStmt) Query(_ []sqlDriver.Value) (sqlDriver.Rows, error) {
	s.conn.mu.Lock()
	defer s.conn.mu.Unlock()
	if s.conn.qIdx >= len(s.conn.queries) {
		return nil, fmt.Errorf("no more query results")
	}
	r := s.conn.queries[s.conn.qIdx]
	s.conn.qIdx++
	if r.err != nil {
		return nil, r.err
	}
	return &testDriverRows{columns: r.columns, data: r.rows}, nil
}

type testDriverRows struct {
	columns []string
	data    [][]sqlDriver.Value
	pos     int
}

func (r *testDriverRows) Columns() []string { return r.columns }
func (r *testDriverRows) Close() error      { return nil }

func (r *testDriverRows) Next(dest []sqlDriver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

type testDriverResult struct {
	lastID   int64
	affected int64
}

func (r *testDriverResult) LastInsertId() (int64, error) { return r.lastID, nil }
func (r *testDriverResult) RowsAffected() (int64, error) { return r.affected, nil }

type testTxDriver struct{ conn *testConn }

func (t *testTxDriver) Commit() error {
	if t.conn.commitErr != nil {
		return t.conn.commitErr
	}
	return nil
}

func (t *testTxDriver) Rollback() error { return nil }

type testConnector struct{ conn *testConn }

func (c *testConnector) Connect(_ context.Context) (sqlDriver.Conn, error) {
	return c.conn, nil
}

func (c *testConnector) Driver() sqlDriver.Driver { return &dummyFakeDriver{} }

type dummyFakeDriver struct{}

func (d *dummyFakeDriver) Open(_ string) (sqlDriver.Conn, error) {
	return nil, fmt.Errorf("not implemented")
}

func newTestDB(t *testing.T, conn *testConn) *sql.DB {
	t.Helper()
	db := sql.OpenDB(&testConnector{conn: conn})
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type fakeResult struct {
	rowsAffected int64
	err          error
}

func (f *fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (f *fakeResult) RowsAffected() (int64, error) { return f.rowsAffected, f.err }

var _ sql.Result = (*fakeResult)(nil)

type fakeTxBeginner struct {
	beginErr error
}

func (f *fakeTxBeginner) BeginTx(_ context.Context, _ *sql.TxOptions) (*sql.Tx, error) {
	return nil, f.beginErr
}

var simpleTable = Table{
	Name:       "items",
	PrimaryKey: []string{"id"},
	Columns:    []string{"id"},
}

func simpleScan(sc Scanner) (string, error) {
	var s string
	return s, sc.Scan(&s)
}

func simpleValues(s string) []any { return []any{s} }

func newSimpleTestRepo(t *testing.T, conn *testConn, tbl Table) *Repository[string] {
	t.Helper()
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: tbl, Scan: simpleScan, Values: simpleValues}
	return New(db, Postgres(), Simple(cfg))
}

type tSnap struct {
	id    string
	name  string
	items []string
}

func compositeScanRoot(sc Scanner) (*tSnap, error) {
	s := &tSnap{}
	return s, sc.Scan(&s.id, &s.name)
}

func compositeScanChild(_ string, sc Scanner, snap *tSnap) error {
	var itemID, orderID, value string
	if err := sc.Scan(&itemID, &orderID, &value); err != nil {
		return err
	}
	snap.items = append(snap.items, value)
	return nil
}

func compositeBuild(s *tSnap) (string, error) {
	return s.id + ":" + s.name, nil
}

func compositeExtractPK(s *tSnap) string { return s.id }

var compositeTable = Table{
	Name:       "orders",
	PrimaryKey: []string{"id"},
	Columns:    []string{"id", "name"},
}

var itemsRelation = Relation{
	Table:      "items",
	ForeignKey: "order_id",
	PrimaryKey: "item_id",
	Columns:    []string{"item_id", "order_id", "value"},
	OnSave:     DeleteAndReinsert,
}

func newCompositeDriver(
	rels []Relation, tbl Table,
	decompose func(string) CompositeValues,
) *compositeDriver[string, *tSnap] {
	if decompose == nil {
		decompose = func(s string) CompositeValues {
			return CompositeValues{Root: []any{s, "name"}}
		}
	}
	return &compositeDriver[string, *tSnap]{
		table:     tbl,
		relations: rels,
		dialect:   Postgres(),
		scanRoot:  compositeScanRoot,
		scanChild: compositeScanChild,
		build:     compositeBuild,
		decompose: decompose,
		extractPK: compositeExtractPK,
	}
}
