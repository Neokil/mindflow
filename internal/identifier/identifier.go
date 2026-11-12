package identifier

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// New returns a cryptographically random opaque identifier.
func New() string {
	buffer := make([]byte, identifierBytes)
	_, readErr := rand.Read(buffer)
	if readErr != nil {
		panic(readErr)
	}

	return hex.EncodeToString(buffer)
}

// Digest returns the SHA-256 representation used for secret verifiers.
func Digest(s string) string {
	h := sha256.Sum256([]byte(s))

	return hex.EncodeToString(h[:])
}

// Now returns the current Unix time in seconds.
func Now() int64 { return time.Now().Unix() }
