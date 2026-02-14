package repository

import (
	"context"
	sqlDriver "database/sql/driver"
	"errors"
	"fmt"
	"testing"
)

func TestBuildOrderSQL_Empty(t *testing.T) {
	t.Parallel()
	if sql := buildOrderSQL(nil); sql != "" {
		t.Errorf("expected empty, got %q", sql)
	}
}

func TestBuildOrderSQL_Single(t *testing.T) {
	t.Parallel()
	orders := []orderClause{{column: "name", dir: Asc}}
	if sql := buildOrderSQL(orders); sql != " ORDER BY name ASC" {
		t.Errorf("got %q", sql)
	}
}

func TestBuildOrderSQL_Multiple(t *testing.T) {
	t.Parallel()
	orders := []orderClause{{column: "name", dir: Asc}, {column: "id", dir: Desc}}
	if sql := buildOrderSQL(orders); sql != " ORDER BY name ASC, id DESC" {
		t.Errorf("got %q", sql)
	}
}

func TestQuery_CombinedSpec_Empty(t *testing.T) {
	t.Parallel()
	q := &Query[string]{}
	if q.combinedSpec() != nil {
		t.Error("expected nil")
	}
}

func TestQuery_CombinedSpec_Single(t *testing.T) {
	t.Parallel()
	s := Eq("x", 1)
	q := &Query[string]{specs: []Spec{s}}
	if q.combinedSpec() != s {
		t.Error("expected same spec")
	}
}

func TestQuery_CombinedSpec_Multiple(t *testing.T) {
	t.Parallel()
	q := &Query[string]{specs: []Spec{Eq("a", 1), Eq("b", 2)}}
	if q.combinedSpec() == nil {
		t.Error("expected non-nil")
	}
}

func TestQuery_EnsurePKOrder_AlreadyPresent(t *testing.T) {
	t.Parallel()
	r := &Repository[string]{table: Table{PrimaryKey: "id"}}
	q := &Query[string]{repo: r, orderCols: []orderClause{{column: "id", dir: Desc}}}
	if len(q.ensurePKOrder()) != 1 {
		t.Error("should not add duplicate PK")
	}
}

func TestQuery_EnsurePKOrder_NotPresent(t *testing.T) {
	t.Parallel()
	r := &Repository[string]{table: Table{PrimaryKey: "id"}}
	q := &Query[string]{repo: r, orderCols: []orderClause{{column: "name", dir: Asc}}}
	orders := q.ensurePKOrder()
	if len(orders) != 2 || orders[1].column != "id" {
		t.Error("expected PK appended")
	}
}

func TestQuery_Chainable(t *testing.T) {
	t.Parallel()
	q := &Query[string]{}
	q.Where(Eq("a", 1)).OrderBy("x", Desc).Limit(10).Offset(5).PageSize(20)
	if len(q.specs) != 1 || len(q.orderCols) != 1 {
		t.Error("chain failed")
	}
	if q.limit == nil || *q.limit != 10 || q.offset == nil || *q.offset != 5 {
		t.Error("limit/offset not set")
	}
}

func TestQuery_AfterBefore(t *testing.T) {
	t.Parallel()
	q := &Query[string]{}
	q.After("abc")
	if q.cursor != "abc" || !q.forward {
		t.Error("After failed")
	}
	q.Before("def")
	if q.cursor != "def" || q.forward {
		t.Error("Before failed")
	}
}

func TestQuery_All_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}, {"b"}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	items, err := repo.Query(context.Background()).All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2, got %d", len(items))
	}
}

func TestQuery_First_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"first"}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	item, err := repo.Query(context.Background()).First()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item != "first" {
		t.Errorf("expected 'first', got %q", item)
	}
}

func TestQuery_First_NotFound(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: nil},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	_, err := repo.Query(context.Background()).First()
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestQuery_First_Error(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: errForTest("fail")}}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	_, err := repo.Query(context.Background()).First()
	if err == nil {
		t.Error("expected error")
	}
}

func errForTest(msg string) error { return errors.New(msg) }

func TestQuery_Count_WithSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"count"}, rows: [][]sqlDriver.Value{{int64(7)}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	count, err := repo.Query(context.Background()).Where(Eq("id", "x")).Count()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 7 {
		t.Errorf("expected 7, got %d", count)
	}
}

func TestQuery_Count_NoSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"count"}, rows: [][]sqlDriver.Value{{int64(3)}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	count, err := repo.Query(context.Background()).Count()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestQuery_Exists_WithSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"exists"}, rows: [][]sqlDriver.Value{{true}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	exists, err := repo.Query(context.Background()).Where(Eq("id", "x")).Exists()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true")
	}
}

func TestQuery_Exists_NoSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"exists"}, rows: [][]sqlDriver.Value{{true}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	exists, err := repo.Query(context.Background()).Exists()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true")
	}
}

func TestQuery_Page_DefaultSize(t *testing.T) {
	t.Parallel()
	rows := make([][]sqlDriver.Value, 21)
	for i := range rows {
		rows[i] = []sqlDriver.Value{fmt.Sprintf("item%d", i)}
	}
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: rows},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	ext := func(s string) map[string]any { return map[string]any{"id": s} }
	page, err := repo.Query(context.Background()).Page(ext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 20 {
		t.Errorf("expected 20 items, got %d", len(page.Items))
	}
	if !page.HasMore {
		t.Error("expected HasMore")
	}
}

func TestQuery_Page_WithCursor(t *testing.T) {
	t.Parallel()
	cursor := EncodeCursor(Cursor{Values: map[string]any{"id": "last"}})
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"next1"}, {"next2"}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	ext := func(s string) map[string]any { return map[string]any{"id": s} }
	page, err := repo.Query(context.Background()).PageSize(5).After(cursor).Page(ext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.HasMore {
		t.Error("expected no more")
	}
	if len(page.Items) != 2 {
		t.Errorf("expected 2, got %d", len(page.Items))
	}
}

func TestQuery_Page_InvalidCursor(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	ext := func(s string) map[string]any { return map[string]any{"id": s} }
	_, err := repo.Query(context.Background()).After("!!!bad!!!").Page(ext)
	if !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestQuery_Page_NoItems(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: nil},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	ext := func(s string) map[string]any { return map[string]any{"id": s} }
	page, err := repo.Query(context.Background()).PageSize(5).Page(ext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.HasMore || page.NextCursor != "" || len(page.Items) != 0 {
		t.Error("expected empty page")
	}
}

func TestQuery_BuildSQL_NoSpecNoOrder(t *testing.T) {
	t.Parallel()
	r := &Repository[string]{table: simpleTable, dialect: Postgres()}
	q := &Query[string]{repo: r, forward: true}
	sql, args := q.buildSQL()
	if sql != "SELECT id FROM items" || len(args) != 0 {
		t.Errorf("got %q, args=%v", sql, args)
	}
}

func TestQuery_BuildSQL_WithSpecOrderLimitOffset(t *testing.T) {
	t.Parallel()
	r := &Repository[string]{table: simpleTable, dialect: Postgres()}
	lim, off := int64(10), int64(5)
	q := &Query[string]{
		repo: r, specs: []Spec{Eq("id", "x")},
		orderCols: []orderClause{{column: "id", dir: Asc}},
		limit:     &lim, offset: &off, forward: true,
	}
	_, args := q.buildSQL()
	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d", len(args))
	}
}

func TestQuery_Page_WithSoftDelete(t *testing.T) {
	t.Parallel()
	tbl := Table{Name: "t", PrimaryKey: "id", Columns: []string{"id"}, SoftDelete: "del"}
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	repo := newSimpleTestRepo(t, conn, tbl)
	ext := func(s string) map[string]any { return map[string]any{"id": s} }
	page, err := repo.Query(context.Background()).PageSize(5).Page(ext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Errorf("expected 1, got %d", len(page.Items))
	}
}

func TestQuery_Page_CursorWithNoExtraSpec(t *testing.T) {
	t.Parallel()
	cursor := EncodeCursor(Cursor{Values: map[string]any{"id": "z"}})
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	repo := newSimpleTestRepo(t, conn, simpleTable)
	ext := func(s string) map[string]any { return map[string]any{"id": s} }
	page, err := repo.Query(context.Background()).PageSize(5).After(cursor).Page(ext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Errorf("expected 1, got %d", len(page.Items))
	}
}
