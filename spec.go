package repository

import (
	"fmt"
	"strings"
)

const likeOp = "LIKE"

type Spec interface {
	ToSQL(d Dialect, offset int) (sql string, args []any, nextOffset int)
}

type comparisonSpec struct {
	column string
	op     string
	value  any
}

func Eq(column string, value any) Spec    { return &comparisonSpec{column, "=", value} }
func NotEq(column string, value any) Spec { return &comparisonSpec{column, "!=", value} }
func Gt(column string, value any) Spec    { return &comparisonSpec{column, ">", value} }
func Gte(column string, value any) Spec   { return &comparisonSpec{column, ">=", value} }
func Lt(column string, value any) Spec    { return &comparisonSpec{column, "<", value} }
func Lte(column string, value any) Spec   { return &comparisonSpec{column, "<=", value} }

func (s *comparisonSpec) ToSQL(d Dialect, offset int) (string, []any, int) {
	return fmt.Sprintf("%s %s %s", s.column, s.op, d.Placeholder(offset)),
		[]any{s.value}, offset + 1
}

type inSpec struct {
	column string
	values []any
	negate bool
}

func In(column string, values ...any) Spec {
	return &inSpec{column: column, values: values}
}

func NotIn(column string, values ...any) Spec {
	return &inSpec{column: column, values: values, negate: true}
}

func (s *inSpec) ToSQL(d Dialect, offset int) (string, []any, int) {
	if len(s.values) == 0 {
		if s.negate {
			return "TRUE", nil, offset
		}
		return "FALSE", nil, offset
	}
	placeholders := make([]string, len(s.values))
	for i := range s.values {
		placeholders[i] = d.Placeholder(offset + i)
	}
	op := "IN"
	if s.negate {
		op = "NOT IN"
	}
	sql := fmt.Sprintf("%s %s (%s)", s.column, op, strings.Join(placeholders, ", "))
	return sql, s.values, offset + len(s.values)
}

type likeSpec struct {
	column  string
	pattern string
	ilike   bool
}

func Like(column, pattern string) Spec {
	return &likeSpec{column: column, pattern: pattern}
}

func ILike(column, pattern string) Spec {
	return &likeSpec{column: column, pattern: pattern, ilike: true}
}

func (s *likeSpec) ToSQL(d Dialect, offset int) (string, []any, int) {
	op := likeOp
	if s.ilike {
		op = d.ILikeOp()
	}
	return fmt.Sprintf("%s %s %s", s.column, op, d.Placeholder(offset)),
		[]any{s.pattern}, offset + 1
}

type betweenSpec struct {
	column string
	from   any
	to     any
}

func Between(column string, from, to any) Spec {
	return &betweenSpec{column: column, from: from, to: to}
}

func (s *betweenSpec) ToSQL(d Dialect, offset int) (string, []any, int) {
	sql := fmt.Sprintf("%s BETWEEN %s AND %s",
		s.column, d.Placeholder(offset), d.Placeholder(offset+1))
	return sql, []any{s.from, s.to}, offset + 2
}

type nullSpec struct {
	column string
	not    bool
}

func IsNull(column string) Spec    { return &nullSpec{column: column} }
func IsNotNull(column string) Spec { return &nullSpec{column: column, not: true} }

func (s *nullSpec) ToSQL(_ Dialect, offset int) (string, []any, int) {
	if s.not {
		return s.column + " IS NOT NULL", nil, offset
	}
	return s.column + " IS NULL", nil, offset
}

type andSpec struct{ specs []Spec }
type orSpec struct{ specs []Spec }
type notSpec struct{ spec Spec }

func And(specs ...Spec) Spec { return &andSpec{specs: specs} }
func Or(specs ...Spec) Spec  { return &orSpec{specs: specs} }
func Not(spec Spec) Spec     { return &notSpec{spec: spec} }

func (s *andSpec) ToSQL(d Dialect, offset int) (string, []any, int) {
	return joinSpecs(s.specs, " AND ", "TRUE", d, offset)
}

func (s *orSpec) ToSQL(d Dialect, offset int) (string, []any, int) {
	return joinSpecs(s.specs, " OR ", "FALSE", d, offset)
}

func (s *notSpec) ToSQL(d Dialect, offset int) (string, []any, int) {
	sql, args, next := s.spec.ToSQL(d, offset)
	return "NOT (" + sql + ")", args, next
}

func joinSpecs(specs []Spec, sep, empty string, d Dialect, offset int) (string, []any, int) {
	if len(specs) == 0 {
		return empty, nil, offset
	}
	if len(specs) == 1 {
		return specs[0].ToSQL(d, offset)
	}
	parts := make([]string, 0, len(specs))
	var allArgs []any
	current := offset
	for _, spec := range specs {
		sql, args, next := spec.ToSQL(d, current)
		parts = append(parts, "("+sql+")")
		allArgs = append(allArgs, args...)
		current = next
	}
	return strings.Join(parts, sep), allArgs, current
}

type rawSpec struct {
	sql  string
	args []any
}

func Raw(sql string, args ...any) Spec {
	return &rawSpec{sql: sql, args: args}
}

func (s *rawSpec) ToSQL(d Dialect, offset int) (string, []any, int) {
	sql := s.sql
	for i := len(s.args); i >= 1; i-- {
		old := fmt.Sprintf("$%d", i)
		placeholder := fmt.Sprintf("__RAW_%d__", i)
		sql = strings.ReplaceAll(sql, old, placeholder)
	}
	for i := 1; i <= len(s.args); i++ {
		placeholder := fmt.Sprintf("__RAW_%d__", i)
		actual := d.Placeholder(offset + i - 1)
		sql = strings.ReplaceAll(sql, placeholder, actual)
	}
	return sql, s.args, offset + len(s.args)
}
