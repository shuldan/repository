package repository

import (
	"context"
	"fmt"
	"strings"
)

type Direction string

const (
	Asc  Direction = "ASC"
	Desc Direction = "DESC"
)

type orderClause struct {
	column string
	dir    Direction
}

type Query[T any] struct {
	repo      *Repository[T]
	ctx       context.Context
	specs     []Spec
	orderCols []orderClause
	limit     *int64
	offset    *int64
	pageSize  *int64
	cursor    string
	forward   bool
}

func (q *Query[T]) Where(s Spec) *Query[T] {
	q.specs = append(q.specs, s)
	return q
}

func (q *Query[T]) OrderBy(column string, dir Direction) *Query[T] {
	q.orderCols = append(q.orderCols, orderClause{column: column, dir: dir})
	return q
}

func (q *Query[T]) Limit(n int64) *Query[T]    { q.limit = &n; return q }
func (q *Query[T]) Offset(n int64) *Query[T]   { q.offset = &n; return q }
func (q *Query[T]) PageSize(n int64) *Query[T] { q.pageSize = &n; return q }

func (q *Query[T]) After(cursor string) *Query[T] {
	q.cursor = cursor
	q.forward = true
	return q
}

func (q *Query[T]) Before(cursor string) *Query[T] {
	q.cursor = cursor
	q.forward = false
	return q
}

func (q *Query[T]) All() ([]T, error) {
	query, args := q.buildSQL()
	return q.repo.driver.findMany(q.ctx, q.repo.db, query, args)
}

func (q *Query[T]) First() (T, error) {
	one := int64(1)
	q.limit = &one
	query, args := q.buildSQL()

	items, err := q.repo.driver.findMany(q.ctx, q.repo.db, query, args)
	if err != nil {
		var zero T
		return zero, err
	}
	if len(items) == 0 {
		var zero T
		return zero, ErrNotFound
	}
	return items[0], nil
}

func (q *Query[T]) Count() (int64, error) {
	spec := q.combinedSpec()
	spec = q.repo.withSoftDelete(spec)

	d := q.repo.dialect
	var query string
	var args []any

	if spec != nil {
		condition, a, _ := spec.ToSQL(d, 1)
		args = a
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", q.repo.table.Name, condition)
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s", q.repo.table.Name)
	}

	var count int64
	err := q.repo.db.QueryRowContext(q.ctx, query, args...).Scan(&count)
	return count, err
}

func (q *Query[T]) Exists() (bool, error) {
	spec := q.combinedSpec()
	spec = q.repo.withSoftDelete(spec)

	d := q.repo.dialect
	var query string
	var args []any

	if spec != nil {
		condition, a, _ := spec.ToSQL(d, 1)
		args = a
		query = fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s)", q.repo.table.Name, condition)
	} else {
		query = fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s)", q.repo.table.Name)
	}

	var exists bool
	err := q.repo.db.QueryRowContext(q.ctx, query, args...).Scan(&exists)
	return exists, err
}

func (q *Query[T]) Page(extract CursorExtractor[T]) (*Page[T], error) {
	if q.pageSize == nil {
		size := int64(20)
		q.pageSize = &size
	}

	d := q.repo.dialect
	orders := q.ensurePKOrder()
	spec := q.combinedSpec()
	spec = q.repo.withSoftDelete(spec)

	if q.cursor != "" {
		cur, err := DecodeCursor(q.cursor)
		if err != nil {
			return nil, err
		}
		keysetSpec := buildKeysetSpec(orders, cur.Values, q.forward)
		if keysetSpec != nil {
			if spec != nil {
				spec = And(spec, keysetSpec)
			} else {
				spec = keysetSpec
			}
		}
	}

	fetchSize := *q.pageSize + 1
	var query string
	var args []any
	nextParam := 1

	if spec != nil {
		condition, specArgs, np := spec.ToSQL(d, 1)
		args = specArgs
		nextParam = np
		query = q.repo.table.selectWhere(condition)
	} else {
		query = q.repo.table.selectFrom()
	}

	query += buildOrderSQL(orders)
	query += fmt.Sprintf(" LIMIT %s", d.Placeholder(nextParam))
	args = append(args, fetchSize)

	items, err := q.repo.driver.findMany(q.ctx, q.repo.db, query, args)
	if err != nil {
		return nil, err
	}

	hasMore := int64(len(items)) > *q.pageSize
	if hasMore {
		items = items[:len(items)-1]
	}

	var nextCursor string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = EncodeCursor(Cursor{Values: extract(last)})
	}

	return &Page[T]{Items: items, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (q *Query[T]) combinedSpec() Spec {
	if len(q.specs) == 0 {
		return nil
	}
	if len(q.specs) == 1 {
		return q.specs[0]
	}
	return And(q.specs...)
}

func (q *Query[T]) ensurePKOrder() []orderClause {
	orders := make([]orderClause, len(q.orderCols))
	copy(orders, q.orderCols)

	existing := make(map[string]bool, len(orders))
	for _, o := range orders {
		existing[o.column] = true
	}

	for _, pk := range q.repo.table.PrimaryKey {
		if !existing[pk] {
			orders = append(orders, orderClause{column: pk, dir: Asc})
		}
	}
	return orders
}

func (q *Query[T]) buildSQL() (string, []any) {
	d := q.repo.dialect
	spec := q.combinedSpec()
	spec = q.repo.withSoftDelete(spec)

	var query string
	var args []any
	nextParam := 1

	if spec != nil {
		condition, specArgs, np := spec.ToSQL(d, 1)
		args = specArgs
		nextParam = np
		query = q.repo.table.selectWhere(condition)
	} else {
		query = q.repo.table.selectFrom()
	}

	if len(q.orderCols) > 0 {
		query += buildOrderSQL(q.orderCols)
	}

	if q.limit != nil {
		query += fmt.Sprintf(" LIMIT %s", d.Placeholder(nextParam))
		args = append(args, *q.limit)
		nextParam++
	}
	if q.offset != nil {
		query += fmt.Sprintf(" OFFSET %s", d.Placeholder(nextParam))
		args = append(args, *q.offset)
	}

	return query, args
}

func buildOrderSQL(orders []orderClause) string {
	if len(orders) == 0 {
		return ""
	}
	parts := make([]string, len(orders))
	for i, o := range orders {
		parts[i] = o.column + " " + string(o.dir)
	}
	return " ORDER BY " + strings.Join(parts, ", ")
}
