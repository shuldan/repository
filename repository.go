package repository

import (
	"context"
	"database/sql"
	"errors"
)

type Finder[T any] interface {
	Find(ctx context.Context, id string) (T, error)
	FindAll(ctx context.Context, limit, offset int) ([]T, error)
	FindBy(ctx context.Context, conditions string, args []any) ([]T, error)
	ExistsBy(ctx context.Context, conditions string, args []any) (bool, error)
	CountBy(ctx context.Context, conditions string, args []any) (int64, error)
}

type Writer[T any] interface {
	Save(ctx context.Context, aggregate T) error
	Delete(ctx context.Context, id string) error
}

type Repository[T any] interface {
	Finder[T]
	Writer[T]
}

type repository[T any] struct {
	db     *sql.DB
	mapper Mapper[T]
}

func NewRepository[T any](
	db *sql.DB,
	mapper Mapper[T],
) Repository[T] {
	return &repository[T]{
		db:     db,
		mapper: mapper,
	}
}

func (r repository[T]) Find(ctx context.Context, id string) (T, error) {
	row := r.mapper.Find(ctx, r.db, id)
	aggregate, err := r.mapper.FromRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return aggregate, errors.Join(ErrEntityNotFound, err)
		}
		return aggregate, err
	}

	return aggregate, nil
}

func (r repository[T]) FindAll(ctx context.Context, limit, offset int) ([]T, error) {
	rows, err := r.mapper.FindAll(ctx, r.db, limit, offset)
	if err != nil {
		return nil, err
	}

	return r.mapper.FromRows(rows)
}

func (r repository[T]) FindBy(ctx context.Context, conditions string, args []any) ([]T, error) {
	rows, err := r.mapper.FindBy(ctx, r.db, conditions, args)
	if err != nil {
		return nil, err
	}
	return r.mapper.FromRows(rows)
}

func (r repository[T]) ExistsBy(ctx context.Context, conditions string, args []any) (bool, error) {
	exists, err := r.mapper.ExistsBy(ctx, r.db, conditions, args)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r repository[T]) CountBy(ctx context.Context, conditions string, args []any) (int64, error) {
	count, err := r.mapper.CountBy(ctx, r.db, conditions, args)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r repository[T]) Save(ctx context.Context, aggregate T) error {
	return r.mapper.Save(ctx, r.db, aggregate)
}

func (r repository[T]) Delete(ctx context.Context, id string) error {
	return r.mapper.Delete(ctx, r.db, id)
}
