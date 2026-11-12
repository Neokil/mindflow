package fault

import "fmt"

// Error describes a business failure independently from its HTTP representation.
type Error struct{ Code, Message string }

// Error returns the safe business-failure message.
func (e *Error) Error() string { return e.Message }

// Fault constructs a business failure with a stable machine-readable code.
func Fault(code, message string) error { return &Error{Code: code, Message: message} }

// Invalid reports a validation failure.
func Invalid(message string) error { return Fault("validation_failed", message) }

// NotFound hides unavailable and unowned resources behind the same failure.
func NotFound() error { return Fault("not_found", "This item is unavailable.") }

// Wrap adds operation context to infrastructure failures while preserving nil.
func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", operation, err)
}
