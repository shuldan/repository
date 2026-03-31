package repository

import (
	"context"
	"database/sql"
	sqlDriver "database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestLoggingExecutor_QueryContext_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	rows, err := exec.QueryContext(context.Background(), "SELECT id FROM t", "arg1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = rows.Close()

	if lg.debugCount() != 1 {
		t.Errorf("expected 1 debug log, got %d", lg.debugCount())
	}
	if lg.errorCount() != 0 {
		t.Errorf("expected 0 error logs, got %d", lg.errorCount())
	}
}

func TestLoggingExecutor_QueryContext_Error(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("query fail")}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, err := exec.QueryContext(context.Background(), "SELECT 1")
	if err == nil {
		t.Fatal("expected error")
	}
	if lg.errorCount() != 1 {
		t.Errorf("expected 1 error log, got %d", lg.errorCount())
	}
	if lg.debugCount() != 0 {
		t.Errorf("expected 0 debug logs, got %d", lg.debugCount())
	}
}

func TestLoggingExecutor_QueryRowContext(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"x"}}},
	}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	row := exec.QueryRowContext(context.Background(), "SELECT id FROM t WHERE id = ?", "x")
	var val string
	if err := row.Scan(&val); err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if lg.debugCount() != 1 {
		t.Errorf("expected 1 debug log, got %d", lg.debugCount())
	}
}

func TestLoggingExecutor_ExecContext_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	result, err := exec.ExecContext(context.Background(), "INSERT INTO t VALUES (?)", "v")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		t.Errorf("expected 1 row affected, got %d", affected)
	}
	if lg.debugCount() != 1 {
		t.Errorf("expected 1 debug log, got %d", lg.debugCount())
	}
}

func TestLoggingExecutor_ExecContext_Error(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{err: fmt.Errorf("exec fail")}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, err := exec.ExecContext(context.Background(), "DELETE FROM t")
	if err == nil {
		t.Fatal("expected error")
	}
	if lg.errorCount() != 1 {
		t.Errorf("expected 1 error log, got %d", lg.errorCount())
	}
}

func TestLoggingTxBeginner_Success(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	beginner := &loggingTxBeginner{inner: db, logger: lg}

	tx, err := beginner.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = tx.Rollback()
	if lg.debugCount() != 1 {
		t.Errorf("expected 1 debug log, got %d", lg.debugCount())
	}
	if lg.errorCount() != 0 {
		t.Errorf("expected 0 error logs, got %d", lg.errorCount())
	}
}

func TestLoggingTxBeginner_Error(t *testing.T) {
	t.Parallel()
	conn := &testConn{beginErr: fmt.Errorf("begin fail")}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	beginner := &loggingTxBeginner{inner: db, logger: lg}

	_, err := beginner.BeginTx(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if lg.errorCount() != 1 {
		t.Errorf("expected 1 error log, got %d", lg.errorCount())
	}
	if lg.debugCount() != 0 {
		t.Errorf("expected 0 debug logs, got %d", lg.debugCount())
	}
}

func TestFormatArgs_Empty(t *testing.T) {
	t.Parallel()
	result := formatArgs(nil)
	if result != "[]" {
		t.Errorf("expected '[]', got %q", result)
	}
}

func TestFormatArgs_WithValues(t *testing.T) {
	t.Parallel()
	result := formatArgs([]any{1, "hello", true})
	if result != "[1, hello, true]" {
		t.Errorf("got %q", result)
	}
}

func TestFormatArgs_SingleValue(t *testing.T) {
	t.Parallel()
	result := formatArgs([]any{42})
	if result != "[42]" {
		t.Errorf("got %q", result)
	}
}

func TestWithLogger_ReturnsNewInstance(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg))
	lg := &mockLogger{}

	repoWithLogger := repo.WithLogger(lg)
	if repoWithLogger == repo {
		t.Error("expected different instance")
	}
	if repoWithLogger.logger != lg {
		t.Error("logger not set")
	}
	if repo.logger != nil {
		t.Error("original should not have logger")
	}
}

func TestWithLogger_Find_LogsQuery(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"abc"}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.Find(context.Background(), "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected at least 1 debug log")
	}
}

func TestWithLogger_FindBy_LogsQuery(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.FindBy(context.Background(), Eq("id", "a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log")
	}
}

func TestWithLogger_Save_LogsExec(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	if err := repo.Save(context.Background(), "val"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log for exec")
	}
}

func TestWithLogger_Delete_LogsExec(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	if err := repo.Delete(context.Background(), "id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log")
	}
}

func TestWithLogger_ExistsBy_LogsQuery(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"exists"}, rows: [][]sqlDriver.Value{{true}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.ExistsBy(context.Background(), Eq("id", "a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log")
	}
}

func TestWithLogger_CountBy_LogsQuery(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"count"}, rows: [][]sqlDriver.Value{{int64(5)}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.CountBy(context.Background(), Eq("id", "a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log")
	}
}

func TestWithLogger_SaveTx_LogsExec(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg)).WithLogger(lg)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := repo.SaveTx(context.Background(), tx, "val"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log for SaveTx")
	}
}

func TestWithLogger_DeleteTx_LogsExec(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg)).WithLogger(lg)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := repo.DeleteTx(context.Background(), tx, "id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log for DeleteTx")
	}
}

func TestExec_WithoutLogger(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg))

	exec := repo.exec()
	if _, ok := exec.(*loggingExecutor); ok {
		t.Error("should not wrap without logger")
	}
}

func TestExec_WithLogger(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg)).WithLogger(&mockLogger{})

	exec := repo.exec()
	if _, ok := exec.(*loggingExecutor); !ok {
		t.Error("should wrap with logger")
	}
}

func TestTxBeginner_WithoutLogger(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg))

	beginner := repo.txBeginner()
	if _, ok := beginner.(*loggingTxBeginner); ok {
		t.Error("should not wrap without logger")
	}
}

func TestTxBeginner_WithLogger(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg)).WithLogger(&mockLogger{})

	beginner := repo.txBeginner()
	if _, ok := beginner.(*loggingTxBeginner); !ok {
		t.Error("should wrap with logger")
	}
}

func TestWithLogger_QueryAll_LogsQuery(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}, {"b"}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	items, err := repo.Query(context.Background()).All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2, got %d", len(items))
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log")
	}
}

func TestWithLogger_QueryFirst_LogsQuery(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"first"}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	item, err := repo.Query(context.Background()).First()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item != "first" {
		t.Errorf("expected 'first', got %q", item)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log")
	}
}

func TestWithLogger_QueryCount_LogsQuery(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"count"}, rows: [][]sqlDriver.Value{{int64(42)}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	count, err := repo.Query(context.Background()).Count()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Errorf("expected 42, got %d", count)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log")
	}
}

func TestWithLogger_QueryExists_LogsQuery(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"exists"}, rows: [][]sqlDriver.Value{{true}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	exists, err := repo.Query(context.Background()).Exists()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true")
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log")
	}
}

func TestWithLogger_QueryPage_LogsQuery(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}, {"b"}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	ext := func(s string) map[string]any { return map[string]any{"id": s} }
	page, err := repo.Query(context.Background()).PageSize(5).Page(ext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 2 {
		t.Errorf("expected 2, got %d", len(page.Items))
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log")
	}
}

func TestLoggingExecutor_QueryContext_LogsArgsInMessage(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"x"}}},
	}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	rows, err := exec.QueryContext(context.Background(), "SELECT 1", "arg1", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = rows.Close()

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if len(lg.debugs) == 0 {
		t.Fatal("no debug logs")
	}
	found := false
	for _, arg := range lg.debugs[0].args {
		if s, ok := arg.(string); ok && strings.Contains(s, "arg1") {
			found = true
		}
	}
	if !found {
		t.Error("expected args to be logged")
	}
}

func TestLoggingExecutor_ExecContext_LogsDuration(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, err := exec.ExecContext(context.Background(), "INSERT INTO t VALUES (?)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if len(lg.debugs) == 0 {
		t.Fatal("no debug logs")
	}
	hasDuration := false
	for i, arg := range lg.debugs[0].args {
		if s, ok := arg.(string); ok && s == "duration" && i+1 < len(lg.debugs[0].args) {
			hasDuration = true
		}
	}
	if !hasDuration {
		t.Error("expected duration in log args")
	}
}

func TestLoggingExecutor_ErrorLog_ContainsErrorMessage(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("specific db error")}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, _ = exec.QueryContext(context.Background(), "SELECT 1")

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if len(lg.errors) == 0 {
		t.Fatal("no error logs")
	}
	found := false
	for _, arg := range lg.errors[0].args {
		if s, ok := arg.(string); ok && strings.Contains(s, "specific db error") {
			found = true
		}
	}
	if !found {
		t.Error("expected error message in log")
	}
}

func TestLoggingExecutor_ExecError_ContainsErrorMessage(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{err: fmt.Errorf("exec specific error")}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, _ = exec.ExecContext(context.Background(), "DELETE FROM t")

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if len(lg.errors) == 0 {
		t.Fatal("no error logs")
	}
	found := false
	for _, arg := range lg.errors[0].args {
		if s, ok := arg.(string); ok && strings.Contains(s, "exec specific error") {
			found = true
		}
	}
	if !found {
		t.Error("expected exec error message in log")
	}
}

func TestWithLogger_FindNotFound_LogsDebug(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: nil},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.Find(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log for QueryRowContext even on not found")
	}
}

func TestWithLogger_FindError_LogsDebugForQueryRow(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("db error")}}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.Find(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if lg.debugCount() < 1 {
		t.Error("expected at least a debug log from QueryRowContext")
	}
}

func TestWithLogger_SaveError_LogsError(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{err: fmt.Errorf("save fail")}}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	err := repo.Save(context.Background(), "val")
	if err == nil {
		t.Fatal("expected error")
	}
	if lg.errorCount() < 1 {
		t.Error("expected error log")
	}
}

func TestWithLogger_DeleteError_LogsError(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{err: fmt.Errorf("delete fail")}}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	err := repo.Delete(context.Background(), "id")
	if err == nil {
		t.Fatal("expected error")
	}
	if lg.errorCount() < 1 {
		t.Error("expected error log")
	}
}

func TestSaveTx_WithoutLogger_NoWrap(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg))

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := repo.SaveTx(context.Background(), tx, "val"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteTx_WithoutLogger_NoWrap(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg))

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := repo.DeleteTx(context.Background(), tx, "id"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWithLogger_Find_WrongPKCount(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.Find(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("expected error for wrong PK count")
	}
}

func TestWithLogger_Delete_WrongPKCount(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	err := repo.Delete(context.Background(), "a", "b")
	if err == nil {
		t.Fatal("expected error for wrong PK count")
	}
}

func TestWithLogger_DeleteTx_WrongPKCount(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg)).WithLogger(lg)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin failed: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	err = repo.DeleteTx(context.Background(), tx, "a", "b")
	if err == nil {
		t.Fatal("expected error for wrong PK count")
	}
}

func TestWithLogger_FindBy_NilSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	items, err := repo.FindBy(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1, got %d", len(items))
	}
}

func TestWithLogger_ExistsBy_NilSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"exists"}, rows: [][]sqlDriver.Value{{true}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	exists, err := repo.ExistsBy(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true")
	}
}

func TestWithLogger_CountBy_NilSpec(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"count"}, rows: [][]sqlDriver.Value{{int64(7)}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	count, err := repo.CountBy(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 7 {
		t.Errorf("expected 7, got %d", count)
	}
}

func TestLoggingExecutor_ImplementsExecutor(t *testing.T) {
	t.Parallel()
	var _ Executor = (*loggingExecutor)(nil)
}

func TestLoggingTxBeginner_ImplementsTxBeginner(t *testing.T) {
	t.Parallel()
	var _ TxBeginner = (*loggingTxBeginner)(nil)
}

func TestWithLogger_SoftDelete_Find(t *testing.T) {
	t.Parallel()
	tbl := Table{
		Name: "t", PrimaryKey: []string{"id"},
		Columns: []string{"id"}, SoftDelete: "del",
	}
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, tbl, lg)

	result, err := repo.Find(context.Background(), "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "a" {
		t.Errorf("expected 'a', got %q", result)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log")
	}
}

func TestWithLogger_MultipleCalls_LogsMultiple(t *testing.T) {
	t.Parallel()
	conn := &testConn{
		queries: []testQueryResult{
			{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
			{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"b"}}},
		},
	}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, _ = repo.Find(context.Background(), "a")
	_, _ = repo.Find(context.Background(), "b")

	if lg.debugCount() < 2 {
		t.Errorf("expected at least 2 debug logs, got %d", lg.debugCount())
	}
}

func TestWithLogger_QueryAll_Error(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("query all fail")}}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.Query(context.Background()).All()
	if err == nil {
		t.Fatal("expected error")
	}
	if lg.errorCount() < 1 {
		t.Error("expected error log")
	}
}

func TestWithLogger_MockLogger_AllMethods(t *testing.T) {
	t.Parallel()
	lg := &mockLogger{}
	lg.Info("info msg", "key", "val")
	lg.Warn("warn msg", "key", "val")
	lg.Error("error msg", "key", "val")
	lg.Debug("debug msg", "key", "val")

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if len(lg.infos) != 1 || lg.infos[0].msg != "info msg" {
		t.Error("info not recorded")
	}
	if len(lg.warns) != 1 || lg.warns[0].msg != "warn msg" {
		t.Error("warn not recorded")
	}
	if len(lg.errors) != 1 || lg.errors[0].msg != "error msg" {
		t.Error("error not recorded")
	}
	if len(lg.debugs) != 1 || lg.debugs[0].msg != "debug msg" {
		t.Error("debug not recorded")
	}
}

func TestMockLogger_Reset(t *testing.T) {
	t.Parallel()
	lg := &mockLogger{}
	lg.Info("a")
	lg.Debug("b")
	lg.Error("c")
	lg.Warn("d")
	lg.reset()

	lg.mu.Lock()
	defer lg.mu.Unlock()
	total := len(lg.infos) + len(lg.warns) + len(lg.errors) + len(lg.debugs)
	if total != 0 {
		t.Errorf("expected 0 after reset, got %d", total)
	}
}

func TestLoggingExecutor_QueryRowContext_NoArgs(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"v"}, rows: [][]sqlDriver.Value{{int64(1)}}},
	}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	row := exec.QueryRowContext(context.Background(), "SELECT 1")
	var v int64
	if err := row.Scan(&v); err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if v != 1 {
		t.Errorf("expected 1, got %d", v)
	}

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if len(lg.debugs) != 1 {
		t.Errorf("expected 1 debug, got %d", len(lg.debugs))
	}
	if lg.debugs[0].msg != "SQL query row" {
		t.Errorf("unexpected msg: %q", lg.debugs[0].msg)
	}
}

func TestLoggingExecutor_QueryContext_EmptyResult(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: nil},
	}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	rows, err := exec.QueryContext(context.Background(), "SELECT id FROM t WHERE 1=0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = rows.Close()
	if lg.debugCount() != 1 {
		t.Errorf("expected 1 debug log, got %d", lg.debugCount())
	}
}

func TestWithLogger_SaveTx_WithLoggerWrapsExecutor(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg)).WithLogger(lg)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	err = repo.SaveTx(context.Background(), tx, "v")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log from wrapped tx executor")
	}
}

func TestWithLogger_DeleteTx_WithLoggerWrapsExecutor(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg)).WithLogger(lg)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	err = repo.DeleteTx(context.Background(), tx, "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lg.debugCount() < 1 {
		t.Error("expected debug log from wrapped tx executor")
	}
}

func TestLoggingExecutor_QueryContext_LogMessageFormat(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	rows, _ := exec.QueryContext(context.Background(), "SELECT id FROM t")
	_ = rows.Close()

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if lg.debugs[0].msg != "SQL query" {
		t.Errorf("expected 'SQL query', got %q", lg.debugs[0].msg)
	}
}

func TestLoggingExecutor_ExecContext_LogMessageFormat(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, _ = exec.ExecContext(context.Background(), "INSERT INTO t VALUES (?)")

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if lg.debugs[0].msg != "SQL exec" {
		t.Errorf("expected 'SQL exec', got %q", lg.debugs[0].msg)
	}
}

func TestLoggingExecutor_QueryError_LogMessageFormat(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("e")}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, _ = exec.QueryContext(context.Background(), "SELECT 1")

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if lg.errors[0].msg != "SQL query failed" {
		t.Errorf("expected 'SQL query failed', got %q", lg.errors[0].msg)
	}
}

func TestLoggingExecutor_ExecError_LogMessageFormat(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{err: fmt.Errorf("e")}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, _ = exec.ExecContext(context.Background(), "DELETE FROM t")

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if lg.errors[0].msg != "SQL exec failed" {
		t.Errorf("expected 'SQL exec failed', got %q", lg.errors[0].msg)
	}
}

func TestLoggingTxBeginner_SuccessLogMessage(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	beginner := &loggingTxBeginner{inner: db, logger: lg}

	tx, _ := beginner.BeginTx(context.Background(), nil)
	_ = tx.Rollback()

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if lg.debugs[0].msg != "SQL begin tx" {
		t.Errorf("expected 'SQL begin tx', got %q", lg.debugs[0].msg)
	}
}

func TestLoggingTxBeginner_ErrorLogMessage(t *testing.T) {
	t.Parallel()
	conn := &testConn{beginErr: fmt.Errorf("fail")}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	beginner := &loggingTxBeginner{inner: db, logger: lg}

	_, _ = beginner.BeginTx(context.Background(), nil)

	lg.mu.Lock()
	defer lg.mu.Unlock()
	if lg.errors[0].msg != "SQL begin tx failed" {
		t.Errorf("expected 'SQL begin tx failed', got %q", lg.errors[0].msg)
	}
}

func TestLoggingExecutor_QueryContext_LogKeyOrder(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"x"}}},
	}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	rows, _ := exec.QueryContext(context.Background(), "SELECT 1")
	_ = rows.Close()

	lg.mu.Lock()
	defer lg.mu.Unlock()
	args := lg.debugs[0].args
	expectedKeys := []string{"query", "args", "duration"}
	for i, key := range expectedKeys {
		idx := i * 2
		if idx >= len(args) {
			t.Fatalf("not enough args at index %d", idx)
		}
		if args[idx] != key {
			t.Errorf("expected key %q at position %d, got %v", key, idx, args[idx])
		}
	}
}

func TestLoggingExecutor_ExecContext_LogKeyOrder(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{rowsAffected: 1}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, _ = exec.ExecContext(context.Background(), "INSERT 1")

	lg.mu.Lock()
	defer lg.mu.Unlock()
	args := lg.debugs[0].args
	expectedKeys := []string{"query", "args", "duration"}
	for i, key := range expectedKeys {
		idx := i * 2
		if idx >= len(args) {
			t.Fatalf("not enough args at index %d", idx)
		}
		if args[idx] != key {
			t.Errorf("expected key %q at position %d, got %v", key, idx, args[idx])
		}
	}
}

func TestLoggingExecutor_ErrorLog_KeyOrder(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("err")}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, _ = exec.QueryContext(context.Background(), "SELECT 1")

	lg.mu.Lock()
	defer lg.mu.Unlock()
	args := lg.errors[0].args
	expectedKeys := []string{"query", "args", "duration", "error"}
	for i, key := range expectedKeys {
		idx := i * 2
		if idx >= len(args) {
			t.Fatalf("not enough args at index %d", idx)
		}
		if args[idx] != key {
			t.Errorf("expected key %q at position %d, got %v", key, idx, args[idx])
		}
	}
}

func TestWithLogger_DoesNotAffectOriginal(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	original := New(db, Postgres(), Simple(cfg))
	lg := &mockLogger{}
	withLog := original.WithLogger(lg)

	if original.logger != nil {
		t.Error("original should have nil logger")
	}
	if withLog.logger == nil {
		t.Error("copy should have logger")
	}
	if original.table.Name != withLog.table.Name {
		t.Error("table should be same")
	}
	if original.dialect != withLog.dialect {
		t.Error("dialect should be same")
	}
}

func TestWithLogger_ChainedWithLogger(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	cfg := SimpleConfig[string]{Table: simpleTable, Scan: simpleScan, Values: simpleValues}
	repo := New(db, Postgres(), Simple(cfg))

	lg1 := &mockLogger{}
	lg2 := &mockLogger{}
	r1 := repo.WithLogger(lg1)
	r2 := r1.WithLogger(lg2)

	if r1.logger != lg1 {
		t.Error("r1 should use lg1")
	}
	if r2.logger != lg2 {
		t.Error("r2 should use lg2")
	}
	if repo.logger != nil {
		t.Error("original should have no logger")
	}
}

func TestWithLogger_QueryBuilder_UsesLogger(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	q := repo.Query(context.Background())
	if q.repo.logger != lg {
		t.Error("query should reference repo with logger")
	}
	_, _ = q.All()
	if lg.debugCount() < 1 {
		t.Error("expected debug log from query")
	}
}

func TestWithLogger_SoftDelete_FindBy(t *testing.T) {
	t.Parallel()
	tbl := Table{
		Name: "t", PrimaryKey: []string{"id"},
		Columns: []string{"id"}, SoftDelete: "del",
	}
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"a"}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, tbl, lg)

	items, err := repo.FindBy(context.Background(), Eq("id", "a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1, got %d", len(items))
	}
}

func TestNewSimpleTestRepoWithLogger_Helper(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{
		{columns: []string{"id"}, rows: [][]sqlDriver.Value{{"test"}}},
	}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)
	if repo == nil {
		t.Fatal("expected non-nil repo")
	}
	if repo.logger != lg {
		t.Error("logger not set")
	}
}

func TestFailingExecutor_QueryContext(t *testing.T) {
	t.Parallel()
	fe := &failingExecutor{err: fmt.Errorf("query fail")}
	_, err := fe.QueryContext(context.Background(), "SELECT 1")
	if err == nil || err.Error() != "query fail" {
		t.Errorf("expected 'query fail', got %v", err)
	}
}

func TestFailingExecutor_ExecContext(t *testing.T) {
	t.Parallel()
	fe := &failingExecutor{err: fmt.Errorf("exec fail")}
	_, err := fe.ExecContext(context.Background(), "INSERT 1")
	if err == nil || err.Error() != "exec fail" {
		t.Errorf("expected 'exec fail', got %v", err)
	}
}

func TestFailingExecutor_ImplementsExecutor(t *testing.T) {
	t.Parallel()
	var _ Executor = (*failingExecutor)(nil)
}

func TestFakeResult_Implements_SqlResult(t *testing.T) {
	t.Parallel()
	var _ sql.Result = (*fakeResult)(nil)
}

func TestLoggingExecutor_ExecError_KeyOrder(t *testing.T) {
	t.Parallel()
	conn := &testConn{execs: []testExecResult{{err: fmt.Errorf("err")}}}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	exec := &loggingExecutor{inner: db, logger: lg}

	_, _ = exec.ExecContext(context.Background(), "DELETE FROM t")

	lg.mu.Lock()
	defer lg.mu.Unlock()
	args := lg.errors[0].args
	expectedKeys := []string{"query", "args", "duration", "error"}
	for i, key := range expectedKeys {
		idx := i * 2
		if idx >= len(args) {
			t.Fatalf("not enough args at index %d", idx)
		}
		if args[idx] != key {
			t.Errorf("expected key %q at position %d, got %v", key, idx, args[idx])
		}
	}
}

func TestLoggingTxBeginner_ErrorLog_KeyOrder(t *testing.T) {
	t.Parallel()
	conn := &testConn{beginErr: fmt.Errorf("fail")}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	beginner := &loggingTxBeginner{inner: db, logger: lg}

	_, _ = beginner.BeginTx(context.Background(), nil)

	lg.mu.Lock()
	defer lg.mu.Unlock()
	args := lg.errors[0].args
	expectedKeys := []string{"duration", "error"}
	for i, key := range expectedKeys {
		idx := i * 2
		if idx >= len(args) {
			t.Fatalf("not enough args at index %d", idx)
		}
		if args[idx] != key {
			t.Errorf("expected key %q at position %d, got %v", key, idx, args[idx])
		}
	}
}

func TestLoggingTxBeginner_SuccessLog_KeyOrder(t *testing.T) {
	t.Parallel()
	conn := &testConn{}
	db := newTestDB(t, conn)
	lg := &mockLogger{}
	beginner := &loggingTxBeginner{inner: db, logger: lg}

	tx, _ := beginner.BeginTx(context.Background(), nil)
	_ = tx.Rollback()

	lg.mu.Lock()
	defer lg.mu.Unlock()
	args := lg.debugs[0].args
	if len(args) < 2 {
		t.Fatalf("not enough args: %v", args)
	}
	if args[0] != "duration" {
		t.Errorf("expected 'duration' key, got %v", args[0])
	}
}

func TestWithLogger_FindBy_Error(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("findby fail")}}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.FindBy(context.Background(), Eq("id", "a"))
	if err == nil {
		t.Fatal("expected error")
	}
	if lg.errorCount() < 1 {
		t.Error("expected error log")
	}
}

func TestWithLogger_ExistsBy_Error(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("exists fail")}}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.ExistsBy(context.Background(), Eq("id", "a"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWithLogger_CountBy_Error(t *testing.T) {
	t.Parallel()
	conn := &testConn{queries: []testQueryResult{{err: fmt.Errorf("count fail")}}}
	lg := &mockLogger{}
	repo := newSimpleTestRepoWithLogger(t, conn, simpleTable, lg)

	_, err := repo.CountBy(context.Background(), Eq("id", "a"))
	if err == nil {
		t.Fatal("expected error")
	}
}
