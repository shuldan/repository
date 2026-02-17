package repository

type Dialect interface {
	Placeholder(n int) string
	Now() string
	ILikeOp() string
	QuoteIdent(name string) string
	UpsertSQL(table string, pks []string, columns []string, opts UpsertOptions) string
	BatchInsertSQL(table string, columns []string, rowCount int) string
}

type UpsertOptions struct {
	VersionColumn string
	CreatedAt     string
	UpdatedAt     string
}
