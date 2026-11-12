package page

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Error is a safe domain failure with a stable machine-readable code.
type Error struct{ Code, Message string }

// Error implements error.
func (e *Error) Error() string { return e.Message }

// FailureCode identifies the domain failure to the HTTP boundary.
func (e *Error) FailureCode() string { return e.Code }

// UserMessage returns the failure text that may be shown to the caller.
func (e *Error) UserMessage() string     { return e.Message }
func failure(code, message string) error { return &Error{Code: code, Message: message} }
func invalid(message string) error       { return failure("validation_failed", message) }
func notFound() error                    { return failure("not_found", "This item is unavailable.") }

func wrap(operation string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", operation, err)
}

func newID() string {
	value := make([]byte, randomIdentifierBytes)
	_, err := rand.Read(value)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(value)
}

const randomIdentifierBytes = 32

// Completed acknowledges a successful domain operation.
type Completed struct {
	OK bool `json:"ok"`
}
