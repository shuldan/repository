package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type compositeDriver[T any, S any] struct {
	table     Table
	relations []Relation //nolint:unused
	dialect   Dialect    //nolint:unused

	scanRoot  func(Scanner) (S, error)                     //nolint:unused
	scanChild func(table string, sc Scanner, snap S) error //nolint:unused
	build     func(S) (T, error)                           //nolint:unused
	decompose func(T) CompositeValues                      //nolint:unused
	extractPK func(S) string                               //nolint:unused
}

//nolint:unused
func (d *compositeDriver[T, S]) findOne(ctx context.Context, exec Executor, query string, args []any) (T, error) {
	var zero T

	row := exec.QueryRowContext(ctx, query, args...)
	snap, err := d.scanRoot(row)
	if err != nil {
		return zero, err
	}

	pk := d.extractPK(snap)
	for _, rel := range d.relations {
		if err := d.loadChildren(ctx, exec, rel, pk, snap); err != nil {
			return zero, err
		}
	}

	return d.build(snap)
}

//nolint:unused
func (d *compositeDriver[T, S]) findMany(ctx context.Context, exec Executor, query string, args []any) ([]T, error) {
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if len(d.relations) == 0 {
		return d.scanAndBuildAll(rows)
	}

	type entry struct {
		id   string
		snap S
	}

	var entries []entry
	snapByID := make(map[string]S)

	for rows.Next() {
		snap, err := d.scanRoot(rows)
		if err != nil {
			return nil, err
		}
		id := d.extractPK(snap)
		entries = append(entries, entry{id: id, snap: snap})
		snapByID[id] = snap
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}

	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.id
	}

	for _, rel := range d.relations {
		if err := d.batchLoadChildren(ctx, exec, rel, ids, snapByID); err != nil {
			return nil, err
		}
	}

	result := make([]T, 0, len(entries))
	for _, e := range entries {
		agg, err := d.build(snapByID[e.id])
		if err != nil {
			return nil, err
		}
		result = append(result, agg)
	}
	return result, nil
}

//nolint:unused
func (d *compositeDriver[T, S]) scanAndBuildAll(rows *sql.Rows) ([]T, error) {
	var result []T
	for rows.Next() {
		snap, err := d.scanRoot(rows)
		if err != nil {
			return nil, err
		}
		agg, err := d.build(snap)
		if err != nil {
			return nil, err
		}
		result = append(result, agg)
	}
	return result, rows.Err()
}

//nolint:unused
func (d *compositeDriver[T, S]) save(
	ctx context.Context, db TxBeginner, exec Executor, aggregate T,
) error {
	cv := d.decompose(aggregate)

	if len(d.relations) == 0 {
		query := d.table.upsertSQL(d.dialect)
		result, err := exec.ExecContext(ctx, query, cv.Root...)
		if err != nil {
			return err
		}
		return d.checkVersion(result)
	}

	if db != nil {
		return inTx(ctx, db, func(tx *sql.Tx) error {
			return d.saveWithChildren(ctx, tx, cv)
		})
	}

	return d.saveWithChildren(ctx, exec, cv)
}

//nolint:unused
func (d *compositeDriver[T, S]) saveWithChildren(
	ctx context.Context, exec Executor, cv CompositeValues,
) error {
	query := d.table.upsertSQL(d.dialect)
	result, err := exec.ExecContext(ctx, query, cv.Root...)
	if err != nil {
		return err
	}
	if err := d.checkVersion(result); err != nil {
		return err
	}

	rootPK := cv.Root[0]

	for _, rel := range d.relations {
		childRows, ok := cv.Children[rel.Table]
		if !ok {
			childRows = nil
		}

		switch rel.OnSave {
		case DeleteAndReinsert:
			delQuery := rel.deleteByFK(d.dialect)
			if _, err := exec.ExecContext(ctx, delQuery, rootPK); err != nil {
				return fmt.Errorf("delete children %s: %w", rel.Table, err)
			}
			if len(childRows) > 0 {
				if err := d.batchInsert(ctx, exec, rel, childRows); err != nil {
					return fmt.Errorf("insert children %s: %w", rel.Table, err)
				}
			}

		case Upsert:
			upsertQuery := rel.upsertSQL(d.dialect)
			for _, row := range childRows {
				if _, err := exec.ExecContext(ctx, upsertQuery, row...); err != nil {
					return fmt.Errorf("upsert child %s: %w", rel.Table, err)
				}
			}
		}
	}

	return nil
}

//nolint:unused
func (d *compositeDriver[T, S]) delete(
	ctx context.Context, db TxBeginner, exec Executor, ids []any,
) error {
	if d.table.SoftDelete != "" || len(d.relations) == 0 {
		query := d.table.deleteSQL(d.dialect)
		_, err := exec.ExecContext(ctx, query, ids...)
		return err
	}

	if db != nil {
		return inTx(ctx, db, func(tx *sql.Tx) error {
			return d.deleteWithChildren(ctx, tx, ids)
		})
	}

	return d.deleteWithChildren(ctx, exec, ids)
}

//nolint:unused
func (d *compositeDriver[T, S]) deleteWithChildren(
	ctx context.Context, exec Executor, ids []any,
) error {
	fkValue := ids[0]

	for i := len(d.relations) - 1; i >= 0; i-- {
		rel := d.relations[i]
		delQuery := rel.deleteByFK(d.dialect)
		if _, err := exec.ExecContext(ctx, delQuery, fkValue); err != nil {
			return fmt.Errorf("delete children %s: %w", rel.Table, err)
		}
	}
	rootQuery := d.table.deleteSQL(d.dialect)
	_, err := exec.ExecContext(ctx, rootQuery, ids...)
	return err
}

//nolint:unused
func (d *compositeDriver[T, S]) checkVersion(result sql.Result) error {
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

//nolint:unused
func (d *compositeDriver[T, S]) loadChildren(
	ctx context.Context, exec Executor, rel Relation, parentID string, snap S,
) error {
	query := rel.selectByFK(d.dialect)
	rows, err := exec.QueryContext(ctx, query, parentID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		if err := d.scanChild(rel.Table, rows, snap); err != nil {
			return err
		}
	}
	return rows.Err()
}

//nolint:unused
func (d *compositeDriver[T, S]) batchLoadChildren(
	ctx context.Context, exec Executor, rel Relation,
	ids []string, snapByID map[string]S,
) error {
	if len(ids) == 0 {
		return nil
	}

	fkIdx := rel.fkColumnIndex()
	if fkIdx == -1 {
		return fmt.Errorf("foreign key %s not found in columns of %s", rel.ForeignKey, rel.Table)
	}

	query := rel.batchSelectByFKs(d.dialect, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	nCols := len(rel.Columns)

	for rows.Next() {
		rawValues := make([]any, nCols)
		scanDest := make([]any, nCols)
		for i := range rawValues {
			scanDest[i] = &rawValues[i]
		}
		if err := rows.Scan(scanDest...); err != nil {
			return err
		}

		parentID := fmt.Sprint(rawValues[fkIdx])
		snap, ok := snapByID[parentID]
		if !ok {
			continue
		}

		if err := d.scanChild(rel.Table, &valuesScanner{values: rawValues}, snap); err != nil {
			return err
		}
	}
	return rows.Err()
}

//nolint:unused
func (d *compositeDriver[T, S]) batchInsert(
	ctx context.Context, exec Executor, rel Relation, childRows [][]any,
) error {
	query := rel.batchInsertSQL(d.dialect, len(childRows))
	allArgs := make([]any, 0, len(childRows)*len(rel.Columns))
	for _, row := range childRows {
		allArgs = append(allArgs, row...)
	}
	_, err := exec.ExecContext(ctx, query, allArgs...)
	return err
}
