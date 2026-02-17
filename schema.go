package repository

import (
	"fmt"
	"strings"
)

type SaveStrategy int

const (
	DeleteAndReinsert SaveStrategy = iota
	Upsert
)

type Table struct {
	Name       string
	PrimaryKey []string
	Columns    []string

	VersionColumn string
	SoftDelete    string
	CreatedAt     string
	UpdatedAt     string
}

type Relation struct {
	Table      string
	ForeignKey string
	PrimaryKey string
	Columns    []string
	OnSave     SaveStrategy
}

type CompositeValues struct {
	Root     []any
	Children map[string][][]any
}

func (t Table) selectFrom() string {
	return fmt.Sprintf("SELECT %s FROM %s", strings.Join(t.Columns, ", "), t.Name)
}

func (t Table) selectWhere(condition string) string {
	return t.selectFrom() + " WHERE " + condition
}

func (t Table) upsertSQL(d Dialect) string {
	return d.UpsertSQL(t.Name, t.PrimaryKey, t.Columns, UpsertOptions{
		VersionColumn: t.VersionColumn,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	})
}

func (t Table) deleteSQL(d Dialect) string {
	whereParts := make([]string, len(t.PrimaryKey))
	for i, pk := range t.PrimaryKey {
		whereParts[i] = fmt.Sprintf("%s = %s", pk, d.Placeholder(i+1))
	}
	where := strings.Join(whereParts, " AND ")

	if t.SoftDelete != "" {
		return fmt.Sprintf("UPDATE %s SET %s = %s WHERE %s AND %s IS NULL",
			t.Name, t.SoftDelete, d.Now(), where, t.SoftDelete)
	}
	return fmt.Sprintf("DELETE FROM %s WHERE %s", t.Name, where)
}

func (r Relation) selectByFK(d Dialect) string {
	return fmt.Sprintf("SELECT %s FROM %s WHERE %s = %s",
		strings.Join(r.Columns, ", "),
		r.Table,
		r.ForeignKey,
		d.Placeholder(1))
}

func (r Relation) deleteByFK(d Dialect) string {
	return fmt.Sprintf("DELETE FROM %s WHERE %s = %s",
		r.Table, r.ForeignKey, d.Placeholder(1))
}

func (r Relation) batchSelectByFKs(d Dialect, count int) string {
	placeholders := make([]string, count)
	for i := range placeholders {
		placeholders[i] = d.Placeholder(i + 1)
	}
	return fmt.Sprintf("SELECT %s FROM %s WHERE %s IN (%s)",
		strings.Join(r.Columns, ", "),
		r.Table,
		r.ForeignKey,
		strings.Join(placeholders, ", "))
}

func (r Relation) insertSQL(d Dialect) string {
	placeholders := make([]string, len(r.Columns))
	for i := range r.Columns {
		placeholders[i] = d.Placeholder(i + 1)
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		r.Table,
		strings.Join(r.Columns, ", "),
		strings.Join(placeholders, ", "))
}

func (r Relation) upsertSQL(d Dialect) string {
	return d.UpsertSQL(r.Table, []string{r.PrimaryKey}, r.Columns, UpsertOptions{})
}

func (r Relation) batchInsertSQL(d Dialect, rowCount int) string {
	return d.BatchInsertSQL(r.Table, r.Columns, rowCount)
}

func (r Relation) fkColumnIndex() int {
	for i, col := range r.Columns {
		if col == r.ForeignKey {
			return i
		}
	}
	return -1
}
