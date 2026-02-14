package repository

import (
	"database/sql"
	"fmt"
	"math"
	"reflect"
	"time"
)

func convertAssign(dest, src any) error {
	if src == nil {
		return setNil(dest)
	}

	if scanner, ok := dest.(sql.Scanner); ok {
		return scanner.Scan(src)
	}

	switch d := dest.(type) {
	case *string:
		return assignString(d, src)
	case *[]byte:
		return assignBytes(d, src)
	case *int64:
		return assignInt64(d, src)
	case *int:
		return assignInt(d, src)
	case *int32:
		return assignInt32(d, src)
	case *float64:
		return assignFloat64(d, src)
	case *float32:
		return assignFloat32(d, src)
	case *bool:
		return assignBool(d, src)
	case *time.Time:
		return assignTime(d, src)
	case *any:
		*d = src
		return nil
	}

	return reflectAssign(dest, src)
}

func assignString(d *string, src any) error {
	switch s := src.(type) {
	case string:
		*d = s
	case []byte:
		*d = string(s)
	default:
		*d = fmt.Sprint(src)
	}
	return nil
}

func assignBytes(d *[]byte, src any) error {
	switch s := src.(type) {
	case []byte:
		c := make([]byte, len(s))
		copy(c, s)
		*d = c
	case string:
		*d = []byte(s)
	default:
		return fmt.Errorf("cannot convert %T to []byte", src)
	}
	return nil
}

func assignInt64(d *int64, src any) error {
	switch s := src.(type) {
	case int64:
		*d = s
	case int:
		*d = int64(s)
	case int32:
		*d = int64(s)
	case float64:
		*d = int64(s)
	default:
		return fmt.Errorf("cannot convert %T to int64", src)
	}
	return nil
}

func assignInt(d *int, src any) error {
	var v int64
	if err := assignInt64(&v, src); err != nil {
		return err
	}
	*d = int(v)
	return nil
}

func assignInt32(d *int32, src any) error {
	var v int64
	if err := assignInt64(&v, src); err != nil {
		return err
	}
	if v < math.MinInt32 || v > math.MaxInt32 {
		return fmt.Errorf("value %d overflows int32", v)
	}
	*d = int32(v)
	return nil
}

func assignFloat64(d *float64, src any) error {
	switch s := src.(type) {
	case float64:
		*d = s
	case int64:
		*d = float64(s)
	case float32:
		*d = float64(s)
	default:
		return fmt.Errorf("cannot convert %T to float64", src)
	}
	return nil
}

func assignFloat32(d *float32, src any) error {
	switch s := src.(type) {
	case float32:
		*d = s
	case float64:
		*d = float32(s)
	default:
		return fmt.Errorf("cannot convert %T to float32", src)
	}
	return nil
}

func assignBool(d *bool, src any) error {
	switch s := src.(type) {
	case bool:
		*d = s
	case int64:
		*d = s != 0
	default:
		return fmt.Errorf("cannot convert %T to bool", src)
	}
	return nil
}

func assignTime(d *time.Time, src any) error {
	switch s := src.(type) {
	case time.Time:
		*d = s
	case string:
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05", s)
		}
		if err != nil {
			return fmt.Errorf("cannot parse %q as time.Time", s)
		}
		*d = t
	default:
		return fmt.Errorf("cannot convert %T to time.Time", src)
	}
	return nil
}

func reflectAssign(dest, src any) error {
	dpv := reflect.ValueOf(dest)
	if dpv.Kind() != reflect.Pointer {
		return fmt.Errorf("destination must be a pointer, got %T", dest)
	}
	dv := dpv.Elem()
	sv := reflect.ValueOf(src)

	if sv.Type().AssignableTo(dv.Type()) {
		dv.Set(sv)
		return nil
	}
	if sv.Type().ConvertibleTo(dv.Type()) {
		dv.Set(sv.Convert(dv.Type()))
		return nil
	}
	return fmt.Errorf("cannot convert %T to %s", src, dv.Type())
}

func setNil(dest any) error {
	dpv := reflect.ValueOf(dest)
	if dpv.Kind() != reflect.Pointer {
		return fmt.Errorf("destination must be a pointer, got %T", dest)
	}
	dpv.Elem().Set(reflect.Zero(dpv.Elem().Type()))
	return nil
}
