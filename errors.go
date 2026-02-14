package repository

import "errors"

var (
	ErrNotFound               = errors.New("entity not found")
	ErrConcurrentModification = errors.New("concurrent modification")
	ErrInvalidCursor          = errors.New("invalid cursor")
)
