package repository

import (
	"strings"
	"testing"
)

func TestPostgresDialect_Placeholder(t *testing.T) {
	t.Parallel()
	d := Postgres()
	tests := []struct {
		n    int
		want string
	}{
		{1, "$1"},
		{5, "$5"},
		{100, "$100"},
	}
	for _, tt := range tests {
		if got := d.Placeholder(tt.n); got != tt.want {
			t.Errorf("Placeholder(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestPostgresDialect_Now(t *testing.T) {
	t.Parallel()
	if got := Postgres().Now(); got != "NOW()" {
		t.Errorf("expected 'NOW()', got %q", got)
	}
}

func TestPostgresDialect_ILikeOp(t *testing.T) {
	t.Parallel()
	if got := Postgres().ILikeOp(); got != "ILIKE" {
		t.Errorf("expected 'ILIKE', got %q", got)
	}
}

func TestPostgresDialect_QuoteIdent(t *testing.T) {
	t.Parallel()
	if got := Postgres().QuoteIdent("col"); got != `"col"` {
		t.Errorf("expected quoted col, got %q", got)
	}
}

func TestPostgresDialect_UpsertSQL_Basic(t *testing.T) {
	t.Parallel()
	d := Postgres()
	sql := d.UpsertSQL("users", "id", []string{"id", "name"}, UpsertOptions{})
	if !strings.Contains(sql, "ON CONFLICT (id) DO UPDATE SET") {
		t.Errorf("expected ON CONFLICT, got %q", sql)
	}
	if !strings.Contains(sql, "name = EXCLUDED.name") {
		t.Errorf("expected EXCLUDED reference, got %q", sql)
	}
}

func TestPostgresDialect_UpsertSQL_WithVersion(t *testing.T) {
	t.Parallel()
	d := Postgres()
	opts := UpsertOptions{
		VersionColumn: "version",
		CreatedAt:     "created_at",
		UpdatedAt:     "updated_at",
	}
	sql := d.UpsertSQL("users", "id", []string{"id", "name", "version"}, opts)
	if !strings.Contains(sql, "version = users.version + 1") {
		t.Errorf("expected version increment, got %q", sql)
	}
	if !strings.Contains(sql, "WHERE users.version = EXCLUDED.version") {
		t.Errorf("expected version WHERE clause, got %q", sql)
	}
}

func TestPostgresDialect_UpsertSQL_NoVersion(t *testing.T) {
	t.Parallel()
	d := Postgres()
	opts := UpsertOptions{CreatedAt: "created_at", UpdatedAt: "updated_at"}
	sql := d.UpsertSQL("t", "id", []string{"id", "name"}, opts)
	if strings.Contains(sql, "WHERE") {
		t.Errorf("unexpected WHERE for no version, got %q", sql)
	}
}

func TestPostgresDialect_BatchInsertSQL(t *testing.T) {
	t.Parallel()
	d := Postgres()
	sql := d.BatchInsertSQL("items", []string{"a", "b"}, 2)
	if !strings.Contains(sql, "($1, $2)") {
		t.Errorf("expected first row ($1, $2), got %q", sql)
	}
	if !strings.Contains(sql, "($3, $4)") {
		t.Errorf("expected second row ($3, $4), got %q", sql)
	}
}
