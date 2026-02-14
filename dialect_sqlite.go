package repository

import (
	"fmt"
	"strings"
)

type sqliteDialect struct{}

func SQLite() Dialect { return &sqliteDialect{} }

func (d *sqliteDialect) Placeholder(_ int) string      { return "?" }
func (d *sqliteDialect) Now() string                   { return "datetime('now')" }
func (d *sqliteDialect) ILikeOp() string               { return "LIKE" }
func (d *sqliteDialect) QuoteIdent(name string) string { return `"` + name + `"` }

func (d *sqliteDialect) UpsertSQL(table, pk string, columns []string, opts UpsertOptions) string {
	insertCols := make([]string, 0, len(columns)+2)
	insertCols = append(insertCols, columns...)

	valuePh := make([]string, len(columns))
	for i := range columns {
		valuePh[i] = "?"
	}

	if opts.CreatedAt != "" {
		insertCols = append(insertCols, opts.CreatedAt)
		valuePh = append(valuePh, d.Now())
	}
	if opts.UpdatedAt != "" {
		insertCols = append(insertCols, opts.UpdatedAt)
		valuePh = append(valuePh, d.Now())
	}

	insert := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(insertCols, ", "),
		strings.Join(valuePh, ", "),
	)

	setClauses := make([]string, 0, len(columns)+1)
	for _, col := range columns {
		if col == pk {
			continue
		}
		if col == opts.VersionColumn && opts.VersionColumn != "" {
			setClauses = append(setClauses,
				fmt.Sprintf("%s = %s + 1", col, col))
			continue
		}
		setClauses = append(setClauses,
			fmt.Sprintf("%s = excluded.%s", col, col))
	}
	if opts.UpdatedAt != "" {
		setClauses = append(setClauses,
			fmt.Sprintf("%s = %s", opts.UpdatedAt, d.Now()))
	}

	conflict := fmt.Sprintf(" ON CONFLICT(%s) DO UPDATE SET %s",
		pk, strings.Join(setClauses, ", "))

	if opts.VersionColumn != "" {
		conflict += fmt.Sprintf(" WHERE %s = excluded.%s",
			opts.VersionColumn, opts.VersionColumn)
	}

	return insert + conflict
}

func (d *sqliteDialect) BatchInsertSQL(table string, columns []string, rowCount int) string {
	colCount := len(columns)
	singleRow := make([]string, colCount)
	for i := range singleRow {
		singleRow[i] = "?"
	}
	rowPh := "(" + strings.Join(singleRow, ", ") + ")"

	allRows := make([]string, rowCount)
	for i := range allRows {
		allRows[i] = rowPh
	}

	return fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		table,
		strings.Join(columns, ", "),
		strings.Join(allRows, ", "),
	)
}
