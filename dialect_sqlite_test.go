package repository

import (
	"strings"
	"testing"
)

func TestSqliteDialect_Placeholder(t *testing.T) {
	t.Parallel()
	if got := SQLite().Placeholder(5); got != "?" {
		t.Errorf("expected '?', got %q", got)
	}
}

func TestSqliteDialect_Now(t *testing.T) {
	t.Parallel()
	if got := SQLite().Now(); got != "datetime('now')" {
		t.Errorf("expected datetime('now'), got %q", got)
	}
}

func TestSqliteDialect_ILikeOp(t *testing.T) {
	t.Parallel()
	if got := SQLite().ILikeOp(); got != "LIKE" {
		t.Errorf("expected 'LIKE', got %q", got)
	}
}

func TestSqliteDialect_QuoteIdent(t *testing.T) {
	t.Parallel()
	if got := SQLite().QuoteIdent("col"); got != `"col"` {
		t.Errorf("expected quoted, got %q", got)
	}
}

func TestSqliteDialect_UpsertSQL_Basic(t *testing.T) {
	t.Parallel()
	d := SQLite()
	sql := d.UpsertSQL("users", []string{"id"}, []string{"id", "name"}, UpsertOptions{})
	if !strings.Contains(sql, "ON CONFLICT(id) DO UPDATE SET") {
		t.Errorf("expected ON CONFLICT, got %q", sql)
	}
	if !strings.Contains(sql, "name = excluded.name") {
		t.Errorf("expected excluded ref, got %q", sql)
	}
}

func TestSqliteDialect_UpsertSQL_WithOptions(t *testing.T) {
	t.Parallel()
	d := SQLite()
	opts := UpsertOptions{
		VersionColumn: "version",
		CreatedAt:     "created_at",
		UpdatedAt:     "updated_at",
	}
	sql := d.UpsertSQL("t", []string{"id"}, []string{"id", "name", "version"}, opts)
	if !strings.Contains(sql, "version = version + 1") {
		t.Errorf("expected version inc, got %q", sql)
	}
	if !strings.Contains(sql, "WHERE version = excluded.version") {
		t.Errorf("expected WHERE clause, got %q", sql)
	}
}

func TestSqliteDialect_UpsertSQL_NoVersion(t *testing.T) {
	t.Parallel()
	d := SQLite()
	opts := UpsertOptions{CreatedAt: "ca", UpdatedAt: "ua"}
	sql := d.UpsertSQL("t", []string{"id"}, []string{"id", "name"}, opts)
	if strings.Contains(sql, "WHERE") {
		t.Errorf("unexpected WHERE, got %q", sql)
	}
}

func TestSqliteDialect_BatchInsertSQL(t *testing.T) {
	t.Parallel()
	d := SQLite()
	sql := d.BatchInsertSQL("t", []string{"a", "b"}, 2)
	if strings.Count(sql, "(?, ?)") != 2 {
		t.Errorf("expected 2 row placeholders, got %q", sql)
	}
}
