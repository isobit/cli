package opts

import (
	"encoding"
	"errors"
	"fmt"
	"time"
)

// setters

func tryGetSetter(i interface{}) setter {
	switch v := i.(type) {
	case setter:
		return v
	case encoding.TextUnmarshaler:
		return textSetter{v}
	case encoding.BinaryUnmarshaler:
		return binarySetter{v}
	case *time.Duration:
		return durationSetter{v}
	case *string:
		return stringSetter{v}
	case
		*bool,
		*int, *int8, *int16, *int32, *int64,
		*uint, *uint8, *uint16, *uint32, *uint64,
		*float32, *float64:
		return scanfSetter{v}
	default:
		return nil
	}
}

type stringSetter struct {
	v *string
}

func (ss stringSetter) Set(s string) error {
	*ss.v = s
	return nil
}

type textSetter struct {
	encoding.TextUnmarshaler
}

func (ts textSetter) Set(s string) error {
	return ts.UnmarshalText([]byte(s))
}

type binarySetter struct {
	encoding.BinaryUnmarshaler
}

func (bs binarySetter) Set(s string) error {
	return bs.UnmarshalBinary([]byte(s))
}

type scanfSetter struct {
	v interface{}
}

func (ss scanfSetter) Set(s string) error {
	n, err := fmt.Sscanf(s, "%v", ss.v)
	if err != nil {
		return err
	} else if n == 0 {
		return errors.New("scanf did not scan any items")
	}
	return nil
}

type durationSetter struct {
	duration *time.Duration
}

func (ds durationSetter) Set(s string) error {
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*ds.duration = v
	return nil
}

// stringers

func tryGetStringer(i interface{}) stringer {
	switch v := i.(type) {
	case stringer:
		return v
	default:
		return nil
	}
}

type staticStringer string

func (ss staticStringer) String() string {
	return string(ss)
}

type sprintfStringer struct {
	v interface{}
}

func (ss sprintfStringer) String() string {
	return fmt.Sprintf("%v", ss.v)
}
