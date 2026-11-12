package session

import (
	"mindflow/internal/account"
)

// Principal identifies the authenticated user and server-side session.
type Principal struct {
	account.User

	SessionID string `json:"-"`
	CSRF      string `json:"-"`
}

// Identity is the verified server-side session identity.
type Identity struct{ UserID, SessionID, CSRF string }

// Credential contains the secret cookie and CSRF value for a newly created session.
type Credential struct {
	Cookie string `json:"-"`
	CSRF   string `json:"csrf"`
}
