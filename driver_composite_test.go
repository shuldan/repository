package repository

import (
	"context"
	sqlDriver "database/sql/driver"
	"fmt"
	"testing"
)

func TestCompositeDriver_CheckVersion_NoColumn(t *testing.T) {
	t.Parallel()
	d := newCompositeDriver(nil, compositeTable, nil)
	if err := d.checkVersion(&fakeResult{rowsAffected: 0}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestCompositeDriver_CheckVersion_Success(t *testing.T) {
	t.Parallel()
	tbl := compositeTable
	tbl.VersionColumn = "ver"
	d := newCompositeDriver(nil, tbl, nil)
	if err := d.checkVersion(&fakeResult{rowsAffected: 1}); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestCompositeDriver_CheckVersion_ConcurrentMod(t *testing.T) {
	t.Parallel()
	tbl := compositeTable
	tbl.VersionColumn = "ver"
	d := newCompositeDriver(nil, tbl, nil)
	if err := d.checkVersion(&fakeResult{rowsAffected: 0}); err != ErrConcurrentModification {
		t.Errorf("expected ErrConcurrentModification, got %v", err)
	}
}

func TestCompositeDriver_CheckVersion_RowsError(t *testing.T) {
	t.Parallel()
	tbl := compositeTable
	tbl.VersionColumn = "ver"
	d := newCompositeDriver(nil, tbl, nil)
	if err := d.checkVersion(&fakeResult{err: fmt.Errorf("fail")}); err == nil {
		t.Error("expected error")
	}
}

func TestCompositeDriver_FindOne_WithRelations(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id", "name"}, rows: [][]sqlDriver.Value{{"o1", "Order1"}}},
		{columns: []string{"item_id", "order_id", "value"}, rows: [][]sqlDriver.Value{
			{"i1", "o1", "val1"}, {"i2", "o1", "val2"},
		}},
	}}
	db := newTestDB(t, conn)
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, nil)
	result, err := d.findOne(context.Background(), db, "SELECT id, name FROM orders WHERE id=$1", []any{"o1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "o1:Order1" {
		t.Errorf("expected 'o1:Order1', got %q", result)
	}
}

func TestCompositeDriver_FindOne_ScanError(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{columns: []string{"id", "name"}, rows: nil}}}
	db := newTestDB(t, conn)
	d := newCompositeDriver(nil, compositeTable, nil)
	_, err := d.findOne(context.Background(), db, "SELECT id, name FROM orders WHERE id=$1", []any{"x"})
	if err == nil {
		t.Error("expected error for no rows")
	}
}

func TestCompositeDriver_FindMany_NoRelations(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id", "name"}, rows: [][]sqlDriver.Value{{"o1", "A"}, {"o2", "B"}}},
	}}
	db := newTestDB(t, conn)
	d := newCompositeDriver(nil, compositeTable, nil)
	items, err := d.findMany(context.Background(), db, "SELECT id, name FROM orders", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestCompositeDriver_FindMany_WithRelations(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id", "name"}, rows: [][]sqlDriver.Value{{"o1", "A"}, {"o2", "B"}}},
		{columns: []string{"item_id", "order_id", "value"}, rows: [][]sqlDriver.Value{
			{"i1", "o1", "v1"}, {"i2", "o2", "v2"},
		}},
	}}
	db := newTestDB(t, conn)
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, nil)
	items, err := d.findMany(context.Background(), db, "SELECT id, name FROM orders", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2, got %d", len(items))
	}
}

func TestCompositeDriver_FindMany_Empty(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id", "name"}, rows: nil},
	}}
	db := newTestDB(t, conn)
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, nil)
	items, err := d.findMany(context.Background(), db, "SELECT id, name FROM orders", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0, got %d", len(items))
	}
}

func TestCompositeDriver_FindMany_QueryError(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("fail")}}}
	db := newTestDB(t, conn)
	d := newCompositeDriver(nil, compositeTable, nil)
	_, err := d.findMany(context.Background(), db, "SELECT id, name FROM orders", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestCompositeDriver_Save_NoRelations(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	d := newCompositeDriver(nil, compositeTable, nil)
	err := d.save(context.Background(), db, db, "o1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompositeDriver_Save_NoRelations_ExecError(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{err: fmt.Errorf("exec fail")}}}
	db := newTestDB(t, conn)
	d := newCompositeDriver(nil, compositeTable, nil)
	err := d.save(context.Background(), db, db, "o1")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCompositeDriver_Save_WithChildren_DeleteReinsert(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{
		{rowsAffected: 1},
		{rowsAffected: 0},
		{rowsAffected: 1},
	}}
	db := newTestDB(t, conn)
	decompose := func(s string) CompositeValues {
		return CompositeValues{
			Root:     []any{s, "name"},
			Children: map[string][][]any{"items": {{"i1", s, "v1"}}},
		}
	}
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, decompose)
	err := d.save(context.Background(), nil, db, "o1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompositeDriver_Save_WithChildren_Upsert(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{
		{rowsAffected: 1},
		{rowsAffected: 1},
	}}
	db := newTestDB(t, conn)
	rel := itemsRelation
	rel.OnSave = Upsert
	decompose := func(s string) CompositeValues {
		return CompositeValues{
			Root:     []any{s, "name"},
			Children: map[string][][]any{"items": {{"i1", s, "v1"}}},
		}
	}
	d := newCompositeDriver([]Relation{rel}, compositeTable, decompose)
	err := d.save(context.Background(), nil, db, "o1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompositeDriver_Save_WithTx(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{
		{rowsAffected: 1},
		{rowsAffected: 0},
	}}
	db := newTestDB(t, conn)
	decompose := func(s string) CompositeValues {
		return CompositeValues{Root: []any{s, "name"}, Children: map[string][][]any{}}
	}
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, decompose)
	err := d.save(context.Background(), db, db, "o1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompositeDriver_Delete_NoRelations(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	d := newCompositeDriver(nil, compositeTable, nil)
	err := d.delete(context.Background(), db, db, "o1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompositeDriver_Delete_SoftDelete(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	tbl := compositeTable
	tbl.SoftDelete = "deleted_at"
	d := newCompositeDriver([]Relation{itemsRelation}, tbl, nil)
	err := d.delete(context.Background(), db, db, "o1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompositeDriver_Delete_WithRelations_NoTx(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{
		{rowsAffected: 1},
		{rowsAffected: 1},
	}}
	db := newTestDB(t, conn)
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, nil)
	err := d.delete(context.Background(), nil, db, "o1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompositeDriver_Delete_WithRelations_WithTx(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{
		{rowsAffected: 1},
		{rowsAffected: 1},
	}}
	db := newTestDB(t, conn)
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, nil)
	err := d.delete(context.Background(), db, db, "o1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCompositeDriver_DeleteWithChildren_Error(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{err: fmt.Errorf("del fail")}}}
	db := newTestDB(t, conn)
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, nil)
	err := d.deleteWithChildren(context.Background(), db, "o1")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCompositeDriver_SaveWithChildren_DeleteError(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{
		{rowsAffected: 1},
		{err: fmt.Errorf("del fail")},
	}}
	db := newTestDB(t, conn)
	decompose := func(s string) CompositeValues {
		return CompositeValues{Root: []any{s, "name"}, Children: map[string][][]any{}}
	}
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, decompose)
	err := d.saveWithChildren(context.Background(), db, decompose("o1"))
	if err == nil {
		t.Error("expected error")
	}
}

func TestCompositeDriver_SaveWithChildren_RootExecError(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{err: fmt.Errorf("root fail")}}}
	db := newTestDB(t, conn)
	decompose := func(s string) CompositeValues {
		return CompositeValues{Root: []any{s, "name"}}
	}
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, decompose)
	err := d.saveWithChildren(context.Background(), db, decompose("o1"))
	if err == nil {
		t.Error("expected error")
	}
}

func TestCompositeDriver_SaveWithChildren_UpsertChildError(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{
		{rowsAffected: 1},
		{err: fmt.Errorf("upsert child fail")},
	}}
	db := newTestDB(t, conn)
	rel := itemsRelation
	rel.OnSave = Upsert
	decompose := func(s string) CompositeValues {
		return CompositeValues{
			Root:     []any{s, "name"},
			Children: map[string][][]any{"items": {{"i1", s, "v1"}}},
		}
	}
	d := newCompositeDriver([]Relation{rel}, compositeTable, decompose)
	err := d.saveWithChildren(context.Background(), db, decompose("o1"))
	if err == nil {
		t.Error("expected error")
	}
}

func TestCompositeDriver_BatchLoadChildren_EmptyIDs(t *testing.T) {
	t.Parallel()
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, nil)
	err := d.batchLoadChildren(context.Background(), nil, itemsRelation, nil, nil)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestCompositeDriver_BatchLoadChildren_FKNotFound(t *testing.T) {
	t.Parallel()
	rel := Relation{Table: "items", ForeignKey: "missing", Columns: []string{"a", "b"}}
	d := newCompositeDriver(nil, compositeTable, nil)
	err := d.batchLoadChildren(context.Background(), nil, rel, []string{"1"}, nil)
	if err == nil {
		t.Error("expected error for missing FK")
	}
}

func TestCompositeDriver_FindOne_LoadChildrenError(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id", "name"}, rows: [][]sqlDriver.Value{{"o1", "A"}}},
		{err: fmt.Errorf("child query fail")},
	}}
	db := newTestDB(t, conn)
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, nil)
	_, err := d.findOne(context.Background(), db, "SELECT id, name FROM orders WHERE id=$1", []any{"o1"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestCompositeDriver_SaveWithChildren_InsertChildError(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{
		{rowsAffected: 1},
		{rowsAffected: 0},
		{err: fmt.Errorf("batch insert fail")},
	}}
	db := newTestDB(t, conn)
	decompose := func(s string) CompositeValues {
		return CompositeValues{
			Root:     []any{s, "name"},
			Children: map[string][][]any{"items": {{"i1", s, "v1"}}},
		}
	}
	d := newCompositeDriver([]Relation{itemsRelation}, compositeTable, decompose)
	err := d.saveWithChildren(context.Background(), db, decompose("o1"))
	if err == nil {
		t.Error("expected error")
	}
}
