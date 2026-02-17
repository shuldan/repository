package repository

import (
	"context"
	"database/sql"
)

type simpleDriver[T any] struct {
	table   Table
	dialect Dialect                  //nolint:unused
	scan    func(Scanner) (T, error) //nolint:unused
	values  func(T) []any            //nolint:unused
}

//nolint:unused
func (d *simpleDriver[T]) findOne(ctx context.Context, exec Executor, query string, args []any) (T, error) {
	row := exec.QueryRowContext(ctx, query, args...)
	return d.scan(row)
}

//nolint:unused
func (d *simpleDriver[T]) findMany(ctx context.Context, exec Executor, query string, args []any) ([]T, error) {
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []T
	for rows.Next() {
		item, err := d.scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

//nolint:unused
func (d *simpleDriver[T]) save(ctx context.Context, _ TxBeginner, exec Executor, aggregate T) error {
	values := d.values(aggregate)
	query := d.table.upsertSQL(d.dialect)
	result, err := exec.ExecContext(ctx, query, values...)
	if err != nil {
		return err
	}
	return d.checkVersion(result)
}

//nolint:unused
func (d *simpleDriver[T]) delete(ctx context.Context, _ TxBeginner, exec Executor, ids []any) error {
	query := d.table.deleteSQL(d.dialect)
	_, err := exec.ExecContext(ctx, query, ids...)
	return err
}

//nolint:unused
func (d *simpleDriver[T]) checkVersion(result sql.Result) error {
	if d.table.VersionColumn == "" {
		return nil
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrConcurrentModification
	}
	return nil
}
