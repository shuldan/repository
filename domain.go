package repository

type ID interface {
	String() string
}

type Aggregate interface {
	ID() ID
}
