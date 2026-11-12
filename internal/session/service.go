package session

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// Service owns persisted login sessions and their secret verifiers.
type Service struct{ db *sql.DB }

// NewService constructs the session lifetime manager.
func NewService(connection *sql.DB) *Service { return &Service{db: connection} }

// Create issues a revocable session for an authenticated account.
func (a *Service) Create(ctx context.Context, userID string) (Credential, error) {
	id, secret, csrf := newID(), newID(), newID()
	_, err := New(a.db).CreateSession(ctx, CreateSessionParams{ID: id, UserID: userID, VerifierHash: secretDigest(secret), Csrf: csrf, ExpiresAt: time.Now().Unix() + sessionLifetimeSeconds})

	return Credential{Cookie: id + "." + secret, CSRF: csrf}, err
}

// Verify compares the cookie verifier with the unexpired stored session.
func (a *Service) Verify(ctx context.Context, cookie string) (Identity, error) {
	parts := strings.SplitN(cookie, ".", cookieParts)
	if len(parts) != cookieParts {
		return Identity{}, failure("session_expired", "Please sign in.")
	}
	row, err := New(a.db).FindSession(ctx, FindSessionParams{ID: parts[0], ExpiresAt: time.Now().Unix()})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Identity{}, err
	}
	if err != nil || subtle.ConstantTimeCompare([]byte(row.VerifierHash), []byte(secretDigest(parts[1]))) != 1 {
		return Identity{}, failure("session_expired", "Please sign in.")
	}

	return Identity{UserID: row.UserID, SessionID: row.ID, CSRF: row.Csrf}, nil
}

// Logout revokes the current session.
func (a *Service) Logout(ctx context.Context, actor Principal) error {
	_, err := New(a.db).DeleteSession(ctx, actor.SessionID)

	return err
}

// RevokeUserWithin removes all sessions during an account credential change.
func RevokeUserWithin(ctx context.Context, transaction *sql.Tx, userID string) error {
	_, err := New(transaction).DeleteUserSessions(ctx, userID)

	return err
}

// RevokeAllWithin is used by stopped-instance restore maintenance.
func RevokeAllWithin(ctx context.Context, transaction *sql.Tx) error {
	return New(transaction).RevokeSessions(ctx)
}

// Cleanup removes expired sessions.
func (a *Service) Cleanup(ctx context.Context) error { return New(a.db).DeleteExpiredSessions(ctx) }

const (
	cookieParts            = 2
	sessionLifetimeSeconds = 30 * 86400
)
