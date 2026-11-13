package session

import (
	"context"
	"crypto/subtle"
	"mindflow/internal/account/storage"
	"strings"
	"time"
)

// Service owns persisted login sessions and their secret verifiers.
type Service struct{ repository *storage.Repository }

// NewService constructs the session lifetime manager.
func NewService(repository *storage.Repository) *Service { return &Service{repository: repository} }

// Create issues a revocable session for an authenticated account.
func (a *Service) Create(ctx context.Context, userID string) (Credential, error) {
	id, secret, csrf := newID(), newID(), newID()
	_, err := a.repository.SessionCreateSession(ctx, storage.SessionCreateSessionParams{ID: id, UserID: userID, VerifierHash: secretDigest(secret), Csrf: csrf, ExpiresAt: time.Now().Unix() + sessionLifetimeSeconds})

	return Credential{Cookie: id + "." + secret, CSRF: csrf}, err
}

// Verify compares the cookie verifier with the unexpired stored session.
func (a *Service) Verify(ctx context.Context, cookie string) (Identity, error) {
	parts := strings.SplitN(cookie, ".", cookieParts)
	if len(parts) != cookieParts {
		return Identity{}, failure("session_expired", "Please sign in.")
	}
	row, err := a.repository.SessionFindSession(ctx, storage.SessionFindSessionParams{ID: parts[0], ExpiresAt: time.Now().Unix()})
	if err != nil && !storage.IsNotFound(err) {
		return Identity{}, err
	}
	if err != nil || subtle.ConstantTimeCompare([]byte(row.VerifierHash), []byte(secretDigest(parts[1]))) != 1 {
		return Identity{}, failure("session_expired", "Please sign in.")
	}

	return Identity{UserID: row.UserID, SessionID: row.ID, CSRF: row.Csrf}, nil
}

// Logout revokes the current session.
func (a *Service) Logout(ctx context.Context, actor Principal) error {
	_, err := a.repository.SessionDeleteSession(ctx, actor.SessionID)

	return err
}

// Cleanup removes expired sessions.
func (a *Service) Cleanup(ctx context.Context) error {
	return a.repository.SessionDeleteExpiredSessions(ctx)
}

const (
	cookieParts            = 2
	sessionLifetimeSeconds = 30 * 86400
)

// ActiveWithin verifies an account and its session are still eligible during control transfer.
func (a *Service) ActiveWithin(ctx context.Context, transaction *storage.Transaction, sessionID, userID string) (bool, error) {
	count, err := a.repository.Within(transaction).SessionActiveWithin(ctx, storage.SessionActiveWithinParams{ID: sessionID, UserID: userID})

	return count > 0, err
}
