package repository

import (
	"context"
	"database/sql"
)

type Mapper[T any] interface {
	Find(ctx context.Context, db *sql.DB, id string) *sql.Row
	FindAll(ctx context.Context, db *sql.DB, limit, offset int64) (*sql.Rows, error)
	FindBy(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error)
	ExistsBy(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error)
	CountBy(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error)
	Save(ctx context.Context, db *sql.DB, aggregate T) error
	Delete(ctx context.Context, db *sql.DB, id string) error
	FromRow(row *sql.Row) (T, error)
	FromRows(rows *sql.Rows) ([]T, error)
}
