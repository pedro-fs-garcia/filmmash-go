package freezeframe

import (
	"errors"
	"strings"
)

var (
	ErrRequired   = errors.New("required")
	ErrOutOfRange = errors.New("out of range")
	ErrWrongCount = errors.New("wrong count")
	ErrOutOfOrder = errors.New("out of order")
	ErrDuplicate  = errors.New("duplicate")
	ErrMismatch   = errors.New("mismatch")
)

type Violation struct {
	Field string
	Err   error
	Msg   string
}

func (i *Violation) Error() string { return i.Field + ": " + i.Msg }

func (i *Violation) Unwrap() error { return i.Err }

type ValidationError struct {
	Where string
	Errs  []error
}

func (ve *ValidationError) Error() string {
	return strings.Join(ve.lines(""), "; ")
}

func (ve *ValidationError) lines(prefix string) []string {
	path := joinField(prefix, ve.Where)

	var out []string
	for _, e := range ve.Errs {
		switch x := e.(type) {
		case *ValidationError:
			out = append(out, x.lines(path)...)
		case *Violation:
			out = append(out, joinField(path, x.Field)+": "+x.Msg)
		default:
			out = append(out, e.Error())
		}
	}
	return out
}

func (ve *ValidationError) Unwrap() []error { return ve.Errs }

func joinField(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + "." + b
	}
}

func group(where string, errs ...error) error {
	kept := make([]error, 0, len(errs))
	for _, e := range errs {
		if e != nil {
			kept = append(kept, e)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return &ValidationError{Where: where, Errs: kept}
}
