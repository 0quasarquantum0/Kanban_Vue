package kanban

import (
	"errors"
	"fmt"
)

// Errors that callers can test for with errors.Is.
var (
	// ErrNotFound means the board, column or card an operation named does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict means a write was based on a state version that is no longer current.
	ErrConflict = errors.New("version conflict")
	// ErrInvalid means the input does not describe a valid state or change.
	ErrInvalid = errors.New("invalid input")
)

// invalidError carries a human-readable message and also matches ErrInvalid, so callers can
// both show the message and branch on errors.Is(err, ErrInvalid).
type invalidError struct{ msg string }

func (e invalidError) Error() string { return e.msg }

func (e invalidError) Is(target error) bool { return target == ErrInvalid }

// invalidf builds an error that matches ErrInvalid. The ...any parameter is fmt's own
// signature: Printf-style functions cannot be typed more tightly.
func invalidf(format string, args ...any) error {
	return invalidError{msg: fmt.Sprintf(format, args...)}
}
