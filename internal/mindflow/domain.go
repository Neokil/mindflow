package mindflow

import "fmt"

// Error is a safe domain failure with a stable API code.
type Error struct{ Code, Message string }

// Error returns the safe domain failure message.
func (e *Error) Error() string { return e.Message }

// FailureCode identifies the domain failure to the HTTP boundary.
func (e *Error) FailureCode() string { return e.Code }

// UserMessage returns the failure text that may be shown to the caller.
func (e *Error) UserMessage() string     { return e.Message }
func failure(code, message string) error { return &Error{Code: code, Message: message} }
func invalid(message string) error       { return failure("validation_failed", message) }

func wrap(operation string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", operation, err)
}
