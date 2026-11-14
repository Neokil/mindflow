package feedback

import "fmt"

// Error represents a safe feedback validation or access failure.
type Error struct{ code, message string }

// Error implements error.
func (e *Error) Error() string { return e.message }

// FailureCode identifies the HTTP failure category.
func (e *Error) FailureCode() string { return e.code }

// UserMessage returns text safe to show to the caller.
func (e *Error) UserMessage() string { return e.message }
func invalid(message string) error   { return &Error{code: "validation_failed", message: message} }
func unavailable() error             { return &Error{code: "not_found", message: "This report is unavailable."} }

func wrap(operation string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", operation, err)
}
