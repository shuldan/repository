package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type Cursor struct {
	Values map[string]any `json:"v"`
}

type CursorExtractor[T any] func(T) map[string]any

type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

func EncodeCursor(c Cursor) string {
	b, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(b)
}

func DecodeCursor(s string) (Cursor, error) {
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: %v", ErrInvalidCursor, err)
	}
	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil {
		return Cursor{}, fmt.Errorf("%w: %v", ErrInvalidCursor, err)
	}
	return c, nil
}

func buildKeysetSpec(orders []orderClause, values map[string]any, forward bool) Spec {
	n := len(orders)
	if n == 0 {
		return nil
	}

	orParts := make([]Spec, 0, n)
	for i := range n {
		andParts := make([]Spec, 0, i+1)

		for j := range i {
			andParts = append(andParts, Eq(orders[j].column, values[orders[j].column]))
		}

		col := orders[i]
		val := values[col.column]
		useGt := (col.dir == Asc && forward) || (col.dir == Desc && !forward)
		if useGt {
			andParts = append(andParts, Gt(col.column, val))
		} else {
			andParts = append(andParts, Lt(col.column, val))
		}

		if len(andParts) == 1 {
			orParts = append(orParts, andParts[0])
		} else {
			orParts = append(orParts, And(andParts...))
		}
	}

	if len(orParts) == 1 {
		return orParts[0]
	}
	return Or(orParts...)
}
