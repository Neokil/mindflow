package domain

// Error describes a business failure independently from its HTTP representation.
type Error struct{ Code, Message string }

func (e *Error) Error() string         { return e.Message }
func Fault(code, message string) error { return &Error{Code: code, Message: message} }
func Invalid(message string) error     { return Fault("validation_failed", message) }
func NotFound() error                  { return Fault("not_found", "This item is unavailable.") }
