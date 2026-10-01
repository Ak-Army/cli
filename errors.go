package cli

import "fmt"

// UsageError is an error in how a command was called: a bad flag, a missing
// or invalid argument, an unknown command. Run prints it with the help of
// the command. A command may return one from Run or Parse for its own checks.
type UsageError struct {
	Err error
}

func (e *UsageError) Error() string { return e.Err.Error() }
func (e *UsageError) Unwrap() error { return e.Err }

// Usagef returns a *UsageError with a formatted message; %w wraps as in
// fmt.Errorf.
func Usagef(format string, a ...any) error {
	return &UsageError{Err: fmt.Errorf(format, a...)}
}
