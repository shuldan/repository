package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Repository[T any] struct {
	db      *sql.DB
	table   Table
	dialect Dialect
	driver  driver[T]
}

func New[T any](db *sql.DB, dialect Dialect, mapping Mapping[T]) *Repository[T] {
	m := mapping.configure(dialect)
	return &Repository[T]{
		db:      db,
		table:   m.table,
		dialect: dialect,
		driver:  m.driver,
	}
}

func (r *Repository[T]) Find(ctx context.Context, ids ...any) (T, error) {
	var zero T
	if len(ids) != len(r.table.PrimaryKey) {
		return zero, fmt.Errorf("expected %d primary key value(s), got %d",
			len(r.table.PrimaryKey), len(ids))
	}

	pkSpec := r.buildPKSpec(ids)
	spec := r.withSoftDelete(pkSpec)
	condition, args, _ := spec.ToSQL(r.dialect, 1)
	query := r.table.selectWhere(condition)

	agg, err := r.driver.findOne(ctx, r.db, query, args)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return agg, fmt.Errorf("%w: %v", ErrNotFound, err)
		}
		return agg, err
	}
	return agg, nil
}

func (r *Repository[T]) FindBy(ctx context.Context, s Spec) ([]T, error) {
	s = r.withSoftDelete(s)

	var query string
	var args []any

	if s != nil {
		condition, a, _ := s.ToSQL(r.dialect, 1)
		args = a
		query = r.table.selectWhere(condition)
	} else {
		query = r.table.selectFrom()
	}

	return r.driver.findMany(ctx, r.db, query, args)
}

func (r *Repository[T]) ExistsBy(ctx context.Context, s Spec) (bool, error) {
	s = r.withSoftDelete(s)

	var query string
	var args []any

	if s != nil {
		condition, a, _ := s.ToSQL(r.dialect, 1)
		args = a
		query = fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s)", r.table.Name, condition)
	} else {
		query = fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s)", r.table.Name)
	}

	var exists bool
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&exists)
	return exists, err
}

func (r *Repository[T]) CountBy(ctx context.Context, s Spec) (int64, error) {
	s = r.withSoftDelete(s)

	var query string
	var args []any

	if s != nil {
		condition, a, _ := s.ToSQL(r.dialect, 1)
		args = a
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", r.table.Name, condition)
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s", r.table.Name)
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *Repository[T]) Save(ctx context.Context, aggregate T) error {
	return r.driver.save(ctx, r.db, r.db, aggregate)
}

func (r *Repository[T]) SaveTx(ctx context.Context, tx *sql.Tx, aggregate T) error {
	return r.driver.save(ctx, nil, tx, aggregate)
}

func (r *Repository[T]) Delete(ctx context.Context, ids ...any) error {
	if len(ids) != len(r.table.PrimaryKey) {
		return fmt.Errorf("expected %d primary key value(s), got %d",
			len(r.table.PrimaryKey), len(ids))
	}
	return r.driver.delete(ctx, r.db, r.db, ids)
}

func (r *Repository[T]) DeleteTx(ctx context.Context, tx *sql.Tx, ids ...any) error {
	if len(ids) != len(r.table.PrimaryKey) {
		return fmt.Errorf("expected %d primary key value(s), got %d",
			len(r.table.PrimaryKey), len(ids))
	}
	return r.driver.delete(ctx, nil, tx, ids)
}

func (r *Repository[T]) Query(ctx context.Context) *Query[T] {
	return &Query[T]{
		repo:    r,
		ctx:     ctx,
		forward: true,
	}
}

func (r *Repository[T]) withSoftDelete(s Spec) Spec {
	if r.table.SoftDelete == "" {
		return s
	}
	sd := IsNull(r.table.SoftDelete)
	if s == nil {
		return sd
	}
	return And(sd, s)
}

func (r *Repository[T]) buildPKSpec(ids []any) Spec {
	if len(ids) == 1 {
		return Eq(r.table.PrimaryKey[0], ids[0])
	}
	specs := make([]Spec, len(ids))
	for i, pk := range r.table.PrimaryKey {
		specs[i] = Eq(pk, ids[i])
	}
	return And(specs...)
}
