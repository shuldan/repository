package repository

import (
	"testing"
)

func pgDialect() Dialect { return Postgres() }

func TestEq_ToSQL(t *testing.T) {
	t.Parallel()
	sql, args, next := Eq("name", "alice").ToSQL(pgDialect(), 1)
	if sql != "name = $1" {
		t.Errorf("expected 'name = $1', got %q", sql)
	}
	if len(args) != 1 || args[0] != "alice" {
		t.Errorf("unexpected args: %v", args)
	}
	if next != 2 {
		t.Errorf("expected next=2, got %d", next)
	}
}

func TestNotEq_ToSQL(t *testing.T) {
	t.Parallel()
	sql, _, _ := NotEq("x", 1).ToSQL(pgDialect(), 3)
	if sql != "x != $3" {
		t.Errorf("expected 'x != $3', got %q", sql)
	}
}

func TestGt_ToSQL(t *testing.T) {
	t.Parallel()
	sql, _, _ := Gt("x", 5).ToSQL(pgDialect(), 1)
	if sql != "x > $1" {
		t.Errorf("got %q", sql)
	}
}

func TestGte_ToSQL(t *testing.T) {
	t.Parallel()
	sql, _, _ := Gte("x", 5).ToSQL(pgDialect(), 1)
	if sql != "x >= $1" {
		t.Errorf("got %q", sql)
	}
}

func TestLt_ToSQL(t *testing.T) {
	t.Parallel()
	sql, _, _ := Lt("x", 5).ToSQL(pgDialect(), 1)
	if sql != "x < $1" {
		t.Errorf("got %q", sql)
	}
}

func TestLte_ToSQL(t *testing.T) {
	t.Parallel()
	sql, _, _ := Lte("x", 5).ToSQL(pgDialect(), 1)
	if sql != "x <= $1" {
		t.Errorf("got %q", sql)
	}
}

func TestIn_ToSQL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    Spec
		wantSQL string
		wantN   int
	}{
		{"with values", In("id", 1, 2, 3), "id IN ($1, $2, $3)", 3},
		{"empty", In("id"), "FALSE", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sql, args, _ := tt.spec.ToSQL(pgDialect(), 1)
			if sql != tt.wantSQL {
				t.Errorf("expected %q, got %q", tt.wantSQL, sql)
			}
			if len(args) != tt.wantN {
				t.Errorf("expected %d args, got %d", tt.wantN, len(args))
			}
		})
	}
}

func TestNotIn_ToSQL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		spec    Spec
		wantSQL string
	}{
		{"with values", NotIn("id", 1, 2), "id NOT IN ($1, $2)"},
		{"empty", NotIn("id"), "TRUE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sql, _, _ := tt.spec.ToSQL(pgDialect(), 1)
			if sql != tt.wantSQL {
				t.Errorf("expected %q, got %q", tt.wantSQL, sql)
			}
		})
	}
}

func TestLike_ToSQL(t *testing.T) {
	t.Parallel()
	sql, args, _ := Like("name", "%test%").ToSQL(pgDialect(), 1)
	if sql != "name LIKE $1" {
		t.Errorf("got %q", sql)
	}
	if len(args) != 1 || args[0] != "%test%" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestILike_ToSQL_Postgres(t *testing.T) {
	t.Parallel()
	sql, _, _ := ILike("name", "%test%").ToSQL(Postgres(), 1)
	if sql != "name ILIKE $1" {
		t.Errorf("got %q", sql)
	}
}

func TestILike_ToSQL_MySQL(t *testing.T) {
	t.Parallel()
	sql, _, _ := ILike("name", "%test%").ToSQL(MySQL(), 1)
	if sql != "name LIKE ?" {
		t.Errorf("got %q", sql)
	}
}

func TestBetween_ToSQL(t *testing.T) {
	t.Parallel()
	sql, args, next := Between("age", 18, 65).ToSQL(pgDialect(), 1)
	if sql != "age BETWEEN $1 AND $2" {
		t.Errorf("got %q", sql)
	}
	if len(args) != 2 {
		t.Errorf("expected 2 args, got %d", len(args))
	}
	if next != 3 {
		t.Errorf("expected next=3, got %d", next)
	}
}

func TestIsNull_ToSQL(t *testing.T) {
	t.Parallel()
	sql, args, next := IsNull("deleted_at").ToSQL(pgDialect(), 5)
	if sql != "deleted_at IS NULL" {
		t.Errorf("got %q", sql)
	}
	if len(args) != 0 {
		t.Errorf("expected no args")
	}
	if next != 5 {
		t.Errorf("expected next=5, got %d", next)
	}
}

func TestIsNotNull_ToSQL(t *testing.T) {
	t.Parallel()
	sql, _, _ := IsNotNull("deleted_at").ToSQL(pgDialect(), 1)
	if sql != "deleted_at IS NOT NULL" {
		t.Errorf("got %q", sql)
	}
}

func TestAnd_ToSQL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		specs   []Spec
		wantSQL string
	}{
		{"empty", nil, "TRUE"},
		{"single", []Spec{Eq("a", 1)}, "a = $1"},
		{"multiple", []Spec{Eq("a", 1), Eq("b", 2)}, "(a = $1) AND (b = $2)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sql, _, _ := And(tt.specs...).ToSQL(pgDialect(), 1)
			if sql != tt.wantSQL {
				t.Errorf("expected %q, got %q", tt.wantSQL, sql)
			}
		})
	}
}

func TestOr_ToSQL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		specs   []Spec
		wantSQL string
	}{
		{"empty", nil, "FALSE"},
		{"single", []Spec{Eq("a", 1)}, "a = $1"},
		{"multiple", []Spec{Eq("a", 1), Eq("b", 2)}, "(a = $1) OR (b = $2)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sql, _, _ := Or(tt.specs...).ToSQL(pgDialect(), 1)
			if sql != tt.wantSQL {
				t.Errorf("expected %q, got %q", tt.wantSQL, sql)
			}
		})
	}
}

func TestNot_ToSQL(t *testing.T) {
	t.Parallel()
	sql, args, next := Not(Eq("a", 1)).ToSQL(pgDialect(), 1)
	if sql != "NOT (a = $1)" {
		t.Errorf("got %q", sql)
	}
	if len(args) != 1 {
		t.Errorf("expected 1 arg")
	}
	if next != 2 {
		t.Errorf("expected next=2, got %d", next)
	}
}

func TestRaw_ToSQL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		rawSQL  string
		args    []any
		wantSQL string
		wantN   int
	}{
		{"no args", "1=1", nil, "1=1", 0},
		{"one arg", "x > $1", []any{5}, "x > $1", 1},
		{"two args", "x BETWEEN $1 AND $2", []any{1, 10}, "x BETWEEN $1 AND $2", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sql, args, _ := Raw(tt.rawSQL, tt.args...).ToSQL(pgDialect(), 1)
			if sql != tt.wantSQL {
				t.Errorf("expected %q, got %q", tt.wantSQL, sql)
			}
			if len(args) != tt.wantN {
				t.Errorf("expected %d args, got %d", tt.wantN, len(args))
			}
		})
	}
}

func TestRaw_ToSQL_WithOffset(t *testing.T) {
	t.Parallel()
	sql, _, next := Raw("x > $1", 5).ToSQL(pgDialect(), 3)
	if sql != "x > $3" {
		t.Errorf("expected 'x > $3', got %q", sql)
	}
	if next != 4 {
		t.Errorf("expected next=4, got %d", next)
	}
}

func TestRaw_ToSQL_MySQL(t *testing.T) {
	t.Parallel()
	sql, _, _ := Raw("x > $1 AND y < $2", 5, 10).ToSQL(MySQL(), 1)
	if sql != "x > ? AND y < ?" {
		t.Errorf("expected mysql placeholders, got %q", sql)
	}
}
