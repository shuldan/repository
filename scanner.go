package repository

import "fmt"

type Scanner interface {
	Scan(dest ...any) error
}

type valuesScanner struct {
	values []any
}

func (s *valuesScanner) Scan(dest ...any) error {
	if len(dest) != len(s.values) {
		return fmt.Errorf("scan: expected %d destinations, got %d", len(s.values), len(dest))
	}
	for i, src := range s.values {
		if err := convertAssign(dest[i], src); err != nil {
			return fmt.Errorf("scan column %d: %w", i, err)
		}
	}
	return nil
}
