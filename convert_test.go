package repository

import (
	"database/sql"
	"fmt"
	"math"
	"testing"
	"time"
)

type testScanner struct {
	val any
	err error
}

func (s *testScanner) Scan(src any) error {
	if s.err != nil {
		return s.err
	}
	s.val = src
	return nil
}

var _ sql.Scanner = (*testScanner)(nil)

func TestConvertAssign_NilSource(t *testing.T) {
	t.Parallel()
	var s string
	if err := convertAssign(&s, nil); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if s != "" {
		t.Errorf("expected empty string, got %q", s)
	}
}

func TestConvertAssign_NilSourcePointer(t *testing.T) {
	t.Parallel()
	var p *int
	if err := convertAssign(&p, nil); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if p != nil {
		t.Errorf("expected nil pointer, got %v", p)
	}
}

func TestConvertAssign_NilSourceNonPointer(t *testing.T) {
	t.Parallel()
	err := convertAssign(42, nil)
	if err == nil {
		t.Error("expected error for non-pointer dest with nil src")
	}
}

func TestConvertAssign_SqlScanner(t *testing.T) {
	t.Parallel()
	sc := &testScanner{}
	if err := convertAssign(sc, "hello"); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if sc.val != "hello" {
		t.Errorf("expected 'hello', got %v", sc.val)
	}
}

func TestConvertAssign_SqlScannerError(t *testing.T) {
	t.Parallel()
	sc := &testScanner{err: fmt.Errorf("scan error")}
	if err := convertAssign(sc, "hello"); err == nil {
		t.Error("expected error from scanner")
	}
}

func TestAssignString_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		src  any
		want string
	}{
		{"from string", "abc", "abc"},
		{"from bytes", []byte("def"), "def"},
		{"from int", 42, "42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var d string
			if err := convertAssign(&d, tt.src); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if d != tt.want {
				t.Errorf("expected %q, got %q", tt.want, d)
			}
		})
	}
}

func TestAssignBytes_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		src     any
		want    string
		wantErr bool
	}{
		{"from bytes", []byte("abc"), "abc", false},
		{"from string", "def", "def", false},
		{"from int", 42, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var d []byte
			err := convertAssign(&d, tt.src)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && string(d) != tt.want {
				t.Errorf("expected %q, got %q", tt.want, string(d))
			}
		})
	}
}

func TestAssignInt64_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		src     any
		want    int64
		wantErr bool
	}{
		{"from int64", int64(10), 10, false},
		{"from int", int(20), 20, false},
		{"from int32", int32(30), 30, false},
		{"from float64", float64(40.0), 40, false},
		{"from string", "x", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var d int64
			err := convertAssign(&d, tt.src)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && d != tt.want {
				t.Errorf("expected %d, got %d", tt.want, d)
			}
		})
	}
}

func TestConvertAssign_Int(t *testing.T) {
	t.Parallel()
	var d int
	if err := convertAssign(&d, int64(99)); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if d != 99 {
		t.Errorf("expected 99, got %d", d)
	}
}

func TestConvertAssign_IntError(t *testing.T) {
	t.Parallel()
	var d int
	if err := convertAssign(&d, "bad"); err == nil {
		t.Error("expected error")
	}
}

func TestConvertAssign_Int32(t *testing.T) {
	t.Parallel()
	var d int32
	if err := convertAssign(&d, int64(55)); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if d != 55 {
		t.Errorf("expected 55, got %d", d)
	}
}

func TestConvertAssign_Int32Error(t *testing.T) {
	t.Parallel()
	var d int32
	if err := convertAssign(&d, "bad"); err == nil {
		t.Error("expected error")
	}
}

func TestConvertAssign_Int32Overflow(t *testing.T) {
	t.Parallel()
	var d int32
	err := convertAssign(&d, int64(math.MaxInt32+1))
	if err == nil {
		t.Error("expected overflow error")
	}
}

func TestConvertAssign_Int32Underflow(t *testing.T) {
	t.Parallel()
	var d int32
	err := convertAssign(&d, int64(math.MinInt32-1))
	if err == nil {
		t.Error("expected underflow error")
	}
}

func TestAssignFloat64_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		src     any
		want    float64
		wantErr bool
	}{
		{"from float64", float64(1.5), 1.5, false},
		{"from int64", int64(2), 2.0, false},
		{"from float32", float32(3.5), float64(float32(3.5)), false},
		{"from string", "x", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var d float64
			err := convertAssign(&d, tt.src)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && d != tt.want {
				t.Errorf("expected %f, got %f", tt.want, d)
			}
		})
	}
}

func TestAssignFloat32_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		src     any
		want    float32
		wantErr bool
	}{
		{"from float32", float32(1.5), 1.5, false},
		{"from float64", float64(2.5), 2.5, false},
		{"from string", "x", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var d float32
			err := convertAssign(&d, tt.src)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && d != tt.want {
				t.Errorf("expected %f, got %f", tt.want, d)
			}
		})
	}
}

func TestAssignBool_Cases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		src     any
		want    bool
		wantErr bool
	}{
		{"from bool true", true, true, false},
		{"from bool false", false, false, false},
		{"from int64 nonzero", int64(1), true, false},
		{"from int64 zero", int64(0), false, false},
		{"from string", "x", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var d bool
			err := convertAssign(&d, tt.src)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && d != tt.want {
				t.Errorf("expected %v, got %v", tt.want, d)
			}
		})
	}
}

func TestAssignTime_Cases(t *testing.T) {
	t.Parallel()
	now := time.Now()
	tests := []struct {
		name    string
		src     any
		wantErr bool
	}{
		{"from time.Time", now, false},
		{"from RFC3339Nano", now.Format(time.RFC3339Nano), false},
		{"from datetime", "2024-01-02 15:04:05", false},
		{"from bad string", "not-a-time", true},
		{"from int", 42, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var d time.Time
			err := convertAssign(&d, tt.src)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConvertAssign_AnyDest(t *testing.T) {
	t.Parallel()
	var d any
	if err := convertAssign(&d, "hello"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if d != "hello" {
		t.Errorf("expected 'hello', got %v", d)
	}
}

func TestConvertAssign_ReflectAssignable(t *testing.T) {
	t.Parallel()
	type myStr string
	var d myStr
	if err := convertAssign(&d, myStr("abc")); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if d != "abc" {
		t.Errorf("expected 'abc', got %v", d)
	}
}

func TestConvertAssign_ReflectConvertible(t *testing.T) {
	t.Parallel()
	type myInt int
	var d myInt
	if err := convertAssign(&d, int(42)); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if d != 42 {
		t.Errorf("expected 42, got %v", d)
	}
}

func TestReflectAssign_NonPointerDest(t *testing.T) {
	t.Parallel()
	err := reflectAssign(42, "hello")
	if err == nil {
		t.Error("expected error for non-pointer dest")
	}
}

func TestReflectAssign_Inconvertible(t *testing.T) {
	t.Parallel()
	var d struct{ X int }
	err := reflectAssign(&d, "hello")
	if err == nil {
		t.Error("expected error for inconvertible types")
	}
}

func TestSetNil_NonPointerDest(t *testing.T) {
	t.Parallel()
	err := setNil(42)
	if err == nil {
		t.Error("expected error for non-pointer dest")
	}
}
