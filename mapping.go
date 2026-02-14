package repository

type Mapping[T any] interface {
	configure(dialect Dialect) mappingResult[T]
}

type mappingResult[T any] struct {
	driver driver[T]
	table  Table
}

type SimpleConfig[T any] struct {
	Table  Table
	Scan   func(Scanner) (T, error)
	Values func(T) []any
}

type simpleMapping[T any] struct {
	cfg SimpleConfig[T]
}

func Simple[T any](cfg SimpleConfig[T]) Mapping[T] {
	return &simpleMapping[T]{cfg: cfg}
}

//nolint:unused
func (m *simpleMapping[T]) configure(dialect Dialect) mappingResult[T] {
	return mappingResult[T]{
		driver: &simpleDriver[T]{
			table:   m.cfg.Table,
			dialect: dialect,
			scan:    m.cfg.Scan,
			values:  m.cfg.Values,
		},
		table: m.cfg.Table,
	}
}

type CompositeConfig[T any, S any] struct {
	Table     Table
	Relations []Relation

	ScanRoot  func(Scanner) (S, error)
	ScanChild func(table string, sc Scanner, snap S) error
	Build     func(S) (T, error)
	Decompose func(T) CompositeValues
	ExtractPK func(S) string
}

type compositeMapping[T any, S any] struct {
	cfg CompositeConfig[T, S]
}

func Composite[T any, S any](cfg CompositeConfig[T, S]) Mapping[T] {
	return &compositeMapping[T, S]{cfg: cfg}
}

//nolint:unused
func (m *compositeMapping[T, S]) configure(dialect Dialect) mappingResult[T] {
	return mappingResult[T]{
		driver: &compositeDriver[T, S]{
			table:     m.cfg.Table,
			relations: m.cfg.Relations,
			dialect:   dialect,
			scanRoot:  m.cfg.ScanRoot,
			scanChild: m.cfg.ScanChild,
			build:     m.cfg.Build,
			decompose: m.cfg.Decompose,
			extractPK: m.cfg.ExtractPK,
		},
		table: m.cfg.Table,
	}
}
