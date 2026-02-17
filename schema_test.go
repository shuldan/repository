package repository

import (
	"strings"
	"testing"
)

func newTestTable() Table {
	return Table{
		Name:       "users",
		PrimaryKey: []string{"id"},
		Columns:    []string{"id", "name", "email"},
	}
}

func TestTable_SelectFrom(t *testing.T) {
	t.Parallel()
	tbl := newTestTable()
	sql := tbl.selectFrom()
	expected := "SELECT id, name, email FROM users"
	if sql != expected {
		t.Errorf("expected %q, got %q", expected, sql)
	}
}

func TestTable_SelectWhere(t *testing.T) {
	t.Parallel()
	tbl := newTestTable()
	sql := tbl.selectWhere("id = $1")
	if !strings.HasSuffix(sql, " WHERE id = $1") {
		t.Errorf("expected WHERE clause, got %q", sql)
	}
}

func TestTable_UpsertSQL(t *testing.T) {
	t.Parallel()
	tbl := newTestTable()
	sql := tbl.upsertSQL(Postgres())
	if !strings.Contains(sql, "INSERT INTO users") {
		t.Errorf("expected INSERT, got %q", sql)
	}
}

func TestTable_DeleteSQL_Hard(t *testing.T) {
	t.Parallel()
	tbl := newTestTable()
	sql := tbl.deleteSQL(Postgres())
	if !strings.HasPrefix(sql, "DELETE FROM users") {
		t.Errorf("expected DELETE, got %q", sql)
	}
}

func TestTable_DeleteSQL_Soft(t *testing.T) {
	t.Parallel()
	tbl := newTestTable()
	tbl.SoftDelete = "deleted_at"
	sql := tbl.deleteSQL(Postgres())
	if !strings.HasPrefix(sql, "UPDATE users SET deleted_at") {
		t.Errorf("expected soft delete UPDATE, got %q", sql)
	}
	if !strings.Contains(sql, "IS NULL") {
		t.Errorf("expected IS NULL check, got %q", sql)
	}
}

func TestRelation_SelectByFK(t *testing.T) {
	t.Parallel()
	r := Relation{Table: "items", ForeignKey: "user_id", Columns: []string{"id", "user_id", "val"}}
	sql := r.selectByFK(Postgres())
	if !strings.Contains(sql, "WHERE user_id = $1") {
		t.Errorf("expected FK condition, got %q", sql)
	}
}

func TestRelation_DeleteByFK(t *testing.T) {
	t.Parallel()
	r := Relation{Table: "items", ForeignKey: "user_id"}
	sql := r.deleteByFK(Postgres())
	if !strings.Contains(sql, "DELETE FROM items WHERE user_id") {
		t.Errorf("expected delete, got %q", sql)
	}
}

func TestRelation_BatchSelectByFKs(t *testing.T) {
	t.Parallel()
	r := Relation{Table: "items", ForeignKey: "uid", Columns: []string{"id", "uid"}}
	sql := r.batchSelectByFKs(Postgres(), 3)
	if !strings.Contains(sql, "IN ($1, $2, $3)") {
		t.Errorf("expected IN clause, got %q", sql)
	}
}

func TestRelation_InsertSQL(t *testing.T) {
	t.Parallel()
	r := Relation{Table: "items", Columns: []string{"id", "val"}}
	sql := r.insertSQL(Postgres())
	if !strings.Contains(sql, "INSERT INTO items") {
		t.Errorf("expected INSERT, got %q", sql)
	}
}

func TestRelation_UpsertSQL(t *testing.T) {
	t.Parallel()
	r := Relation{Table: "items", PrimaryKey: "id", Columns: []string{"id", "val"}}
	sql := r.upsertSQL(Postgres())
	if !strings.Contains(sql, "ON CONFLICT") {
		t.Errorf("expected upsert, got %q", sql)
	}
}

func TestRelation_BatchInsertSQL(t *testing.T) {
	t.Parallel()
	r := Relation{Table: "items", Columns: []string{"a", "b"}}
	sql := r.batchInsertSQL(Postgres(), 2)
	if !strings.Contains(sql, "VALUES") {
		t.Errorf("expected VALUES, got %q", sql)
	}
}

func TestRelation_FkColumnIndex_Found(t *testing.T) {
	t.Parallel()
	r := Relation{ForeignKey: "uid", Columns: []string{"id", "uid", "val"}}
	if idx := r.fkColumnIndex(); idx != 1 {
		t.Errorf("expected 1, got %d", idx)
	}
}

func TestRelation_FkColumnIndex_NotFound(t *testing.T) {
	t.Parallel()
	r := Relation{ForeignKey: "missing", Columns: []string{"id", "val"}}
	if idx := r.fkColumnIndex(); idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}
