package repository

import (
	"testing"
)

func TestSimpleMapping_Configure(t *testing.T) {
	t.Parallel()
	cfg := SimpleConfig[string]{
		Table: Table{
			Name:       "test",
			PrimaryKey: "id",
			Columns:    []string{"id", "val"},
		},
		Scan:   func(sc Scanner) (string, error) { return "", nil },
		Values: func(s string) []any { return []any{s} },
	}
	m := Simple(cfg)
	result := m.configure(Postgres())
	if result.table.Name != "test" {
		t.Errorf("expected table name 'test', got %q", result.table.Name)
	}
	if result.driver == nil {
		t.Error("expected non-nil driver")
	}
}

func TestCompositeMapping_Configure(t *testing.T) {
	t.Parallel()
	cfg := CompositeConfig[string, string]{
		Table: Table{
			Name:       "composite",
			PrimaryKey: "id",
			Columns:    []string{"id"},
		},
		Relations: []Relation{},
		ScanRoot:  func(sc Scanner) (string, error) { return "", nil },
		ScanChild: func(table string, sc Scanner, snap string) error { return nil },
		Build:     func(s string) (string, error) { return s, nil },
		Decompose: func(s string) CompositeValues { return CompositeValues{} },
		ExtractPK: func(s string) string { return s },
	}
	m := Composite(cfg)
	result := m.configure(Postgres())
	if result.table.Name != "composite" {
		t.Errorf("expected 'composite', got %q", result.table.Name)
	}
	if result.driver == nil {
		t.Error("expected non-nil driver")
	}
}
