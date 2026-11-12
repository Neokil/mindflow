package store

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"strings"

	"mindflow/internal/domain"
)

func (a *Repositories) Authenticate(ctx context.Context, cookie string) (principal, error) {
	var p principal
	parts := strings.SplitN(cookie, ".", 2)
	if len(parts) != 2 {
		return p, domain.Fault("session_expired", "Please sign in.")
	}
	var hash string
	err := a.DB.QueryRowContext(ctx, `SELECT u.id,u.email,u.role,u.status,u.appearance,s.id,s.csrf,s.verifier_hash FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.id=? AND s.expires_at>? AND u.status='active'`, parts[0], now()).Scan(&p.ID, &p.Email, &p.Role, &p.Status, &p.Appearance, &p.SessionID, &p.CSRF, &hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return p, err
	}
	if err != nil || subtle.ConstantTimeCompare([]byte(hash), []byte(digest(parts[1]))) != 1 {
		return p, domain.Fault("session_expired", "Please sign in.")
	}
	return p, nil
}
