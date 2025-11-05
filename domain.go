package repository

type ID interface {
	String() string
}

type Aggregate interface {
	ID() ID
	CreateMemento() (Memento, error)
}

type Memento interface {
	ID() ID
	RestoreAggregate() (Aggregate, error)
}
