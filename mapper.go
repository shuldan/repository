package repository

import (
	"context"
	"database/sql"
)

type Mapper[T Aggregate, M Memento] interface {
	Find(ctx context.Context, db *sql.DB, id ID) *sql.Row
	FindAll(ctx context.Context, db *sql.DB, limit, offset int) (*sql.Rows, error)
	FindBy(ctx context.Context, db *sql.DB, conditions string, args []any) (*sql.Rows, error)
	ExistsBy(ctx context.Context, db *sql.DB, conditions string, args []any) (bool, error)
	CountBy(ctx context.Context, db *sql.DB, conditions string, args []any) (int64, error)
	Save(ctx context.Context, db *sql.DB, memento M) error
	Delete(ctx context.Context, db *sql.DB, id ID) error
	FromRow(row *sql.Row) (M, error)
	FromRows(rows *sql.Rows) ([]M, error)
}
