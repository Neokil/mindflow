package authentication

import (
	"context"
	"errors"
	"mindflow/internal/account"
	"mindflow/internal/account/session"
)

// LoginCommand contains the credentials supplied at sign-in.
type LoginCommand struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResult returns public sign-in data while hiding the session cookie.
type LoginResult struct {
	User   account.User `json:"user"`
	CSRF   string       `json:"csrf"`
	Cookie string       `json:"-"`
}

// Service authenticates requests using account and session collaborators.
type Service struct {
	accounts *account.Service
	sessions *session.Service
	dev      bool
}

// NewService constructs authentication without its own storage.
func NewService(accounts *account.Service, sessions *session.Service, dev bool) *Service {
	return &Service{accounts: accounts, sessions: sessions, dev: dev}
}

// Login verifies account credentials before creating a separate session.
func (a *Service) Login(ctx context.Context, command LoginCommand) (LoginResult, error) {
	user, err := a.accounts.VerifyCredentials(ctx, command.Email, command.Password)
	if err != nil {
		return LoginResult{}, err
	}
	credential, err := a.sessions.Create(ctx, user.ID)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{User: user, CSRF: credential.CSRF, Cookie: credential.Cookie}, nil
}

// Authenticate verifies the session and checks the current account state.
func (a *Service) Authenticate(ctx context.Context, cookie string) (session.Principal, error) {
	identity, err := a.sessions.Verify(ctx, cookie)
	if err != nil {
		return session.Principal{}, err
	}
	user, err := a.accounts.Find(ctx, identity.UserID)
	problem, ok := errors.AsType[*account.Error](err)
	if ok && problem.Code == "not_found" {
		return session.Principal{}, failure("session_expired", "Please sign in.")
	}
	if err != nil {
		return session.Principal{}, err
	}
	if user.Status != account.StatusActive {
		return session.Principal{}, failure("session_expired", "Please sign in.")
	}

	return session.Principal{User: user, SessionID: identity.SessionID, CSRF: identity.CSRF}, nil
}
