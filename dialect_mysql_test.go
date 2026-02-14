package repository

import (
	"strings"
	"testing"
)

func TestMysqlDialect_Placeholder(t *testing.T) {
	t.Parallel()
	d := MySQL()
	if got := d.Placeholder(1); got != "?" {
		t.Errorf("expected '?', got %q", got)
	}
	if got := d.Placeholder(99); got != "?" {
		t.Errorf("expected '?', got %q", got)
	}
}

func TestMysqlDialect_Now(t *testing.T) {
	t.Parallel()
	if got := MySQL().Now(); got != "NOW()" {
		t.Errorf("expected 'NOW()', got %q", got)
	}
}

func TestMysqlDialect_ILikeOp(t *testing.T) {
	t.Parallel()
	if got := MySQL().ILikeOp(); got != "LIKE" {
		t.Errorf("expected 'LIKE', got %q", got)
	}
}

func TestMysqlDialect_QuoteIdent(t *testing.T) {
	t.Parallel()
	if got := MySQL().QuoteIdent("col"); got != "`col`" {
		t.Errorf("expected '`col`', got %q", got)
	}
}

func TestMysqlDialect_UpsertSQL_Basic(t *testing.T) {
	t.Parallel()
	d := MySQL()
	sql := d.UpsertSQL("users", "id", []string{"id", "name"}, UpsertOptions{})
	if !strings.Contains(sql, "INSERT INTO users") {
		t.Errorf("expected INSERT INTO, got %q", sql)
	}
	if !strings.Contains(sql, "ON DUPLICATE KEY UPDATE") {
		t.Errorf("expected ON DUPLICATE KEY UPDATE, got %q", sql)
	}
	if strings.Contains(sql, "id = VALUES(id)") {
		t.Errorf("pk should be skipped in update, got %q", sql)
	}
}

func TestMysqlDialect_UpsertSQL_WithOptions(t *testing.T) {
	t.Parallel()
	d := MySQL()
	opts := UpsertOptions{
		VersionColumn: "version",
		CreatedAt:     "created_at",
		UpdatedAt:     "updated_at",
	}
	sql := d.UpsertSQL("users", "id", []string{"id", "name", "version"}, opts)
	if !strings.Contains(sql, "created_at") {
		t.Error("expected created_at in SQL")
	}
	if !strings.Contains(sql, "updated_at") {
		t.Error("expected updated_at in SQL")
	}
	if !strings.Contains(sql, "version = version + 1") {
		t.Errorf("expected version increment, got %q", sql)
	}
}

func TestMysqlDialect_BatchInsertSQL(t *testing.T) {
	t.Parallel()
	d := MySQL()
	sql := d.BatchInsertSQL("items", []string{"a", "b"}, 3)
	if !strings.Contains(sql, "INSERT INTO items") {
		t.Errorf("expected INSERT INTO, got %q", sql)
	}
	if strings.Count(sql, "(?, ?)") != 3 {
		t.Errorf("expected 3 row placeholders, got %q", sql)
	}
}
