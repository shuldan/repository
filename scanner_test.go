package repository

import (
	"strings"
	"testing"
)

func TestValuesScanner_Success(t *testing.T) {
	t.Parallel()
	vs := &valuesScanner{values: []any{"hello", int64(42)}}
	var s string
	var i int64
	if err := vs.Scan(&s, &i); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "hello" {
		t.Errorf("expected 'hello', got %q", s)
	}
	if i != 42 {
		t.Errorf("expected 42, got %d", i)
	}
}

func TestValuesScanner_MismatchCount(t *testing.T) {
	t.Parallel()
	vs := &valuesScanner{values: []any{"a", "b"}}
	var s string
	err := vs.Scan(&s)
	if err == nil {
		t.Error("expected error for mismatched count")
	}
	if !strings.Contains(err.Error(), "expected 2 destinations, got 1") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValuesScanner_ConvertError(t *testing.T) {
	t.Parallel()
	vs := &valuesScanner{values: []any{42}}
	var d []byte
	err := vs.Scan(&d)
	if err == nil {
		t.Error("expected conversion error")
	}
	if !strings.Contains(err.Error(), "scan column 0") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValuesScanner_Empty(t *testing.T) {
	t.Parallel()
	vs := &valuesScanner{values: []any{}}
	err := vs.Scan()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
