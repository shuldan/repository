package repository

import (
	"context"
)

type driver[T any] interface {
	findOne(ctx context.Context, exec Executor, query string, args []any) (T, error)
	findMany(ctx context.Context, exec Executor, query string, args []any) ([]T, error)
	save(ctx context.Context, db TxBeginner, exec Executor, aggregate T) error
	delete(ctx context.Context, db TxBeginner, exec Executor, ids []any) error
}
