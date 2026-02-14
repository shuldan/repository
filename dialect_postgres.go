package repository

import (
	"fmt"
	"strings"
)

type postgresDialect struct{}

// Postgres возвращает диалект PostgreSQL.
func Postgres() Dialect { return &postgresDialect{} }

func (d *postgresDialect) Placeholder(n int) string      { return fmt.Sprintf("$%d", n) }
func (d *postgresDialect) Now() string                   { return "NOW()" }
func (d *postgresDialect) ILikeOp() string               { return "ILIKE" }
func (d *postgresDialect) QuoteIdent(name string) string { return `"` + name + `"` }

func (d *postgresDialect) UpsertSQL(table, pk string, columns []string, opts UpsertOptions) string {
	insertCols := make([]string, 0, len(columns)+2)
	insertCols = append(insertCols, columns...)

	valuePh := make([]string, 0, len(columns)+2)
	for i := range columns {
		valuePh = append(valuePh, d.Placeholder(i+1))
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
				fmt.Sprintf("%s = %s.%s + 1", col, table, col))
			continue
		}
		setClauses = append(setClauses,
			fmt.Sprintf("%s = EXCLUDED.%s", col, col))
	}
	if opts.UpdatedAt != "" {
		setClauses = append(setClauses,
			fmt.Sprintf("%s = %s", opts.UpdatedAt, d.Now()))
	}

	conflict := fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET %s",
		pk, strings.Join(setClauses, ", "))

	if opts.VersionColumn != "" {
		conflict += fmt.Sprintf(" WHERE %s.%s = EXCLUDED.%s",
			table, opts.VersionColumn, opts.VersionColumn)
	}

	return insert + conflict
}

func (d *postgresDialect) BatchInsertSQL(table string, columns []string, rowCount int) string {
	colCount := len(columns)
	rowPh := make([]string, rowCount)
	for i := 0; i < rowCount; i++ {
		ph := make([]string, colCount)
		for j := range ph {
			ph[j] = d.Placeholder(i*colCount + j + 1)
		}
		rowPh[i] = "(" + strings.Join(ph, ", ") + ")"
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		table,
		strings.Join(columns, ", "),
		strings.Join(rowPh, ", "),
	)
}
