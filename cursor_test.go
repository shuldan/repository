package repository

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestEncodeCursor_Basic(t *testing.T) {
	t.Parallel()
	c := Cursor{Values: map[string]any{"id": "abc"}}
	encoded := EncodeCursor(c)
	if encoded == "" {
		t.Error("expected non-empty encoded cursor")
	}
}

func TestDecodeCursor_Valid(t *testing.T) {
	t.Parallel()
	original := Cursor{Values: map[string]any{"id": "abc"}}
	encoded := EncodeCursor(original)
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded.Values["id"] != "abc" {
		t.Errorf("expected id='abc', got %v", decoded.Values["id"])
	}
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	t.Parallel()
	_, err := DecodeCursor("!!!invalid!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
	if !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestDecodeCursor_InvalidJSON(t *testing.T) {
	t.Parallel()
	encoded := base64.URLEncoding.EncodeToString([]byte("{bad json"))
	_, err := DecodeCursor(encoded)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestBuildKeysetSpec_Empty(t *testing.T) {
	t.Parallel()
	result := buildKeysetSpec(nil, nil, true)
	if result != nil {
		t.Error("expected nil for empty orders")
	}
}

func TestBuildKeysetSpec_SingleAscForward(t *testing.T) {
	t.Parallel()
	orders := []orderClause{{column: "id", dir: Asc}}
	vals := map[string]any{"id": "100"}
	spec := buildKeysetSpec(orders, vals, true)
	if spec == nil {
		t.Fatal("expected non-nil spec")
	}
	sql, args, _ := spec.ToSQL(Postgres(), 1)
	if sql != "id > $1" {
		t.Errorf("expected 'id > $1', got %q", sql)
	}
	if len(args) != 1 || args[0] != "100" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestBuildKeysetSpec_SingleDescForward(t *testing.T) {
	t.Parallel()
	orders := []orderClause{{column: "id", dir: Desc}}
	vals := map[string]any{"id": "100"}
	spec := buildKeysetSpec(orders, vals, true)
	sql, _, _ := spec.ToSQL(Postgres(), 1)
	if sql != "id < $1" {
		t.Errorf("expected 'id < $1', got %q", sql)
	}
}

func TestBuildKeysetSpec_SingleAscBackward(t *testing.T) {
	t.Parallel()
	orders := []orderClause{{column: "id", dir: Asc}}
	vals := map[string]any{"id": "100"}
	spec := buildKeysetSpec(orders, vals, false)
	sql, _, _ := spec.ToSQL(Postgres(), 1)
	if sql != "id < $1" {
		t.Errorf("expected 'id < $1', got %q", sql)
	}
}

func TestBuildKeysetSpec_SingleDescBackward(t *testing.T) {
	t.Parallel()
	orders := []orderClause{{column: "id", dir: Desc}}
	vals := map[string]any{"id": "100"}
	spec := buildKeysetSpec(orders, vals, false)
	sql, _, _ := spec.ToSQL(Postgres(), 1)
	if sql != "id > $1" {
		t.Errorf("expected 'id > $1', got %q", sql)
	}
}

func TestBuildKeysetSpec_MultipleOrders(t *testing.T) {
	t.Parallel()
	orders := []orderClause{
		{column: "name", dir: Asc},
		{column: "id", dir: Asc},
	}
	vals := map[string]any{"name": "a", "id": "1"}
	spec := buildKeysetSpec(orders, vals, true)
	if spec == nil {
		t.Fatal("expected non-nil spec")
	}
	sql, args, _ := spec.ToSQL(Postgres(), 1)
	if len(args) != 3 {
		t.Errorf("expected 3 args, got %d", len(args))
	}
	if sql == "" {
		t.Error("expected non-empty SQL")
	}
}
