package repository

import (
	"context"
	"database/sql"
	"errors"
)

type Finder[T Aggregate, I ID] interface {
	Find(ctx context.Context, id I) (T, error)
	FindAll(ctx context.Context, limit, offset int) ([]T, error)
	FindBy(ctx context.Context, conditions string, args []any) ([]T, error)
	ExistsBy(ctx context.Context, conditions string, args []any) (bool, error)
	CountBy(ctx context.Context, conditions string, args []any) (int64, error)
}

type Writer[T Aggregate, I ID] interface {
	Save(ctx context.Context, aggregate T) error
	Delete(ctx context.Context, id I) error
}

type Repository[T Aggregate, I ID] interface {
	Finder[T, I]
	Writer[T, I]
}

type repository[T Aggregate, I ID] struct {
	db     *sql.DB
	mapper Mapper[T]
}

func NewRepository[T Aggregate, I ID](
	db *sql.DB,
	mapper Mapper[T],
) Repository[T, I] {
	return &repository[T, I]{
		db:     db,
		mapper: mapper,
	}
}

func (r repository[T, I]) Find(ctx context.Context, id I) (T, error) {
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

func (r repository[T, I]) FindAll(ctx context.Context, limit, offset int) ([]T, error) {
	rows, err := r.mapper.FindAll(ctx, r.db, limit, offset)
	if err != nil {
		return nil, err
	}

	return r.mapper.FromRows(rows)
}

func (r repository[T, I]) FindBy(ctx context.Context, conditions string, args []any) ([]T, error) {
	rows, err := r.mapper.FindBy(ctx, r.db, conditions, args)
	if err != nil {
		return nil, err
	}
	return r.mapper.FromRows(rows)
}

func (r repository[T, I]) ExistsBy(ctx context.Context, conditions string, args []any) (bool, error) {
	exists, err := r.mapper.ExistsBy(ctx, r.db, conditions, args)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r repository[T, I]) CountBy(ctx context.Context, conditions string, args []any) (int64, error) {
	count, err := r.mapper.CountBy(ctx, r.db, conditions, args)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r repository[T, I]) Save(ctx context.Context, aggregate T) error {
	return r.mapper.Save(ctx, r.db, aggregate)
}

func (r repository[T, I]) Delete(ctx context.Context, id I) error {
	return r.mapper.Delete(ctx, r.db, id)
}
