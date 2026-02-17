package repository

import (
	"fmt"
	"strings"
)

type mysqlDialect struct{}

func MySQL() Dialect { return &mysqlDialect{} }

func (d *mysqlDialect) Placeholder(_ int) string      { return "?" }
func (d *mysqlDialect) Now() string                   { return "NOW()" }
func (d *mysqlDialect) ILikeOp() string               { return "LIKE" }
func (d *mysqlDialect) QuoteIdent(name string) string { return "`" + name + "`" }

func (d *mysqlDialect) UpsertSQL(table string, pks []string, columns []string, opts UpsertOptions) string {
	pkSet := makeSet(pks)

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
		if pkSet[col] {
			continue
		}
		if col == opts.VersionColumn && opts.VersionColumn != "" {
			setClauses = append(setClauses,
				fmt.Sprintf("%s = %s + 1", col, col))
			continue
		}
		setClauses = append(setClauses,
			fmt.Sprintf("%s = VALUES(%s)", col, col))
	}
	if opts.UpdatedAt != "" {
		setClauses = append(setClauses,
			fmt.Sprintf("%s = %s", opts.UpdatedAt, d.Now()))
	}

	if len(setClauses) == 0 {
		return strings.Replace(insert, "INSERT INTO", "INSERT IGNORE INTO", 1)
	}

	return insert + " ON DUPLICATE KEY UPDATE " + strings.Join(setClauses, ", ")
}

func (d *mysqlDialect) BatchInsertSQL(table string, columns []string, rowCount int) string {
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
