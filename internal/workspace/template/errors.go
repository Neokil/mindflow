package template

import (
	"fmt"
	"mindflow/internal/workspace/storage"
)

// Error is a safe template validation failure.
type Error struct{ message string }

func (e *Error) Error() string { return e.message }

// FailureCode identifies invalid requests to the HTTP boundary.
func (e *Error) FailureCode() string { return "validation_failed" }

// UserMessage supplies the safe user-facing message.
func (e *Error) UserMessage() string { return e.message }
func invalid(message string) error   { return &Error{message: message} }
func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return &missingError{}
	}

	return err
}

type missingError struct{}

func (*missingError) Error() string         { return "This template or workspace is unavailable." }
func (*missingError) FailureCode() string   { return "not_found" }
func (e *missingError) UserMessage() string { return e.Error() }

func wrap(operation string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", operation, err)
}
