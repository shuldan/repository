package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type loggingExecutor struct {
	inner  Executor
	logger Logger
}

func (e *loggingExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := e.inner.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		e.logger.Error("SQL query failed",
			"query", query,
			"args", formatArgs(args),
			"duration", duration.String(),
			"error", err.Error(),
		)
	} else {
		e.logger.Debug("SQL query",
			"query", query,
			"args", formatArgs(args),
			"duration", duration.String(),
		)
	}

	return rows, err
}

func (e *loggingExecutor) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := e.inner.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)

	e.logger.Debug("SQL query row",
		"query", query,
		"args", formatArgs(args),
		"duration", duration.String(),
	)

	return row
}

func (e *loggingExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := e.inner.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		e.logger.Error("SQL exec failed",
			"query", query,
			"args", formatArgs(args),
			"duration", duration.String(),
			"error", err.Error(),
		)
	} else {
		e.logger.Debug("SQL exec",
			"query", query,
			"args", formatArgs(args),
			"duration", duration.String(),
		)
	}

	return result, err
}

type loggingTxBeginner struct {
	inner  TxBeginner
	logger Logger
}

func (b *loggingTxBeginner) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	start := time.Now()
	tx, err := b.inner.BeginTx(ctx, opts)
	duration := time.Since(start)

	if err != nil {
		b.logger.Error("SQL begin tx failed",
			"duration", duration.String(),
			"error", err.Error(),
		)
	} else {
		b.logger.Debug("SQL begin tx",
			"duration", duration.String(),
		)
	}

	return tx, err
}

func formatArgs(args []any) string {
	if len(args) == 0 {
		return "[]"
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprintf("%v", a)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
