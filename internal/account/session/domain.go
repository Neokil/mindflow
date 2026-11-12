package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// Error is a safe domain failure with a stable API code.
type Error struct{ Code, Message string }

// Error implements error.
func (e *Error) Error() string { return e.Message }

// FailureCode identifies the domain failure to the HTTP boundary.
func (e *Error) FailureCode() string { return e.Code }

// UserMessage returns the failure text that may be shown to the caller.
func (e *Error) UserMessage() string     { return e.Message }
func failure(code, message string) error { return &Error{Code: code, Message: message} }

func newID() string {
	value := make([]byte, randomIdentifierBytes)
	_, err := rand.Read(value)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(value)
}

const randomIdentifierBytes = 32

func secretDigest(value string) string {
	hash := sha256.Sum256([]byte(value))

	return hex.EncodeToString(hash[:])
}
