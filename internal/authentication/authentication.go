package authentication

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"mindflow/internal/authentication/dbquery"
	"mindflow/internal/fault"
	"mindflow/internal/identifier"
	"strings"
)

// Authenticate verifies a session verifier against the active account.
func (a *Service) Authenticate(ctx context.Context, cookie string) (Principal, error) {
	var page Principal
	parts := strings.SplitN(cookie, ".", cookieParts)
	if len(parts) != cookieParts {
		return page, fault.Fault("session_expired", "Please sign in.")
	}
	var hash string
	authenticateSessionRow, err := dbquery.New(a.db).AuthenticateSession(ctx, dbquery.AuthenticateSessionParams{ID: parts[0], ExpiresAt: identifier.Now()})
	page.ID = authenticateSessionRow.ID
	page.Email = authenticateSessionRow.Email
	page.Role = authenticateSessionRow.Role
	page.Status = authenticateSessionRow.Status
	page.Appearance = authenticateSessionRow.Appearance
	page.SessionID = authenticateSessionRow.SessionID
	page.CSRF = authenticateSessionRow.Csrf
	hash = authenticateSessionRow.VerifierHash

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return page, err
	}
	if err != nil || subtle.ConstantTimeCompare([]byte(hash), []byte(identifier.Digest(parts[1]))) != 1 {
		return page, fault.Fault("session_expired", "Please sign in.")
	}

	return page, nil
}
