package app

import (
	"mindflow/internal/authentication"
	"net/http"
	"strings"
)

func (a *App) identityRoutes() {
	a.endpoint("GET /api/v1/setup/status", false, false, func(_ http.ResponseWriter, r *http.Request) (any, error) {
		initialized, err := a.authentication.Initialized(r.Context())

		return map[string]bool{"initialized": initialized}, err
	})
	a.endpoint("POST /api/v1/setup/complete", false, false, a.setup)
	a.endpoint("POST /api/v1/setup/test-email", false, false, a.setupTestEmail)
	a.endpoint("POST /api/v1/auth/login", false, false, a.login)
	a.endpoint("POST /api/v1/auth/logout", true, false, func(w http.ResponseWriter, r *http.Request) (any, error) {
		err := a.authentication.Logout(r.Context(), who(r))
		a.setSessionCookie(w, &http.Cookie{Name: "mindflow_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})

		return map[string]bool{"ok": true}, err
	})
	a.endpoint("GET /api/v1/auth/me", true, false, func(_ http.ResponseWriter, r *http.Request) (any, error) {
		p := who(r)

		return map[string]any{"user": p.User, csrfKey: p.CSRF}, nil
	})
	a.endpoint("PATCH /api/v1/auth/preferences", true, false, func(w http.ResponseWriter, request *http.Request) (any, error) {
		var value struct {
			Appearance string `json:"appearance"`
		}
		decodeErr := decode(w, request, &value)
		if decodeErr != nil {
			return nil, decodeErr
		}
		if value.Appearance != "system" && value.Appearance != "light" && value.Appearance != "dark" {
			return nil, bad("Invalid appearance.")
		}
		err := a.authentication.SetAppearance(request.Context(), who(request), value.Appearance)

		return map[string]bool{"ok": true}, err
	})
	a.endpoint("POST /api/v1/auth/forgot-password", false, false, a.forgot)
	a.endpoint("POST /api/v1/auth/reset-password", false, false, a.redeem)
	a.endpoint("POST /api/v1/auth/accept-invite", false, false, a.redeem)
	a.endpoint("GET /api/v1/admin/users", true, true, a.listUsers)
	a.endpoint("POST /api/v1/admin/users", true, true, a.invite)
	a.endpoint("PATCH /api/v1/admin/users/{id}", true, true, a.updateUser)
	a.endpoint("DELETE /api/v1/admin/users/{id}", true, true, a.deleteUser)
	a.endpoint("POST /api/v1/admin/users/{id}/invite", true, true, a.resendInvite)
	a.endpoint("POST /api/v1/admin/users/{id}/reset", true, true, a.adminReset)
	a.endpoint("GET /api/v1/admin/settings", true, true, func(_ http.ResponseWriter, r *http.Request) (any, error) {
		s, err := a.authentication.Settings(r.Context())
		configured := s.SMTP.Password != ""
		s.SMTP.Password = ""

		return map[string]any{"settings": s, "passwordConfigured": configured}, err
	})
	a.endpoint("PUT /api/v1/admin/settings", true, true, a.updateSettings)
	a.endpoint("POST /api/v1/admin/settings/test-email", true, true, func(_ http.ResponseWriter, r *http.Request) (any, error) {
		err := a.authentication.SendTestEmail(r.Context(), who(r))
		if err != nil {
			return nil, err
		}

		return map[string]bool{"ok": true}, nil
	})
}

func (a *App) setup(w http.ResponseWriter, request *http.Request) (any, error) {
	if a.limited(request, "setup") {
		return nil, problem(http.StatusTooManyRequests, "rate_limit", "Please retry in a minute.")
	}
	var value authentication.SetupCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.authentication.Setup(request.Context(), who(request), value)
}

func (a *App) setupTestEmail(w http.ResponseWriter, request *http.Request) (any, error) {
	if a.limited(request, "setup-mail") {
		return nil, problem(http.StatusTooManyRequests, "rate_limit", "Please retry in a minute.")
	}
	var value authentication.SetupTestEmailCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.authentication.SetupTestEmail(request.Context(), who(request), value)
}

func (a *App) login(writer http.ResponseWriter, request *http.Request) (any, error) {
	if a.limited(request, "login") {
		return nil, problem(http.StatusTooManyRequests, "rate_limit", "Too many attempts. Please retry in a minute.")
	}
	var v authentication.LoginCommand
	decodeErr2 := decode(writer, request, &v)
	if decodeErr2 != nil {
		return nil, decodeErr2
	}
	result, err := a.authentication.Login(request.Context(), who(request), v)
	if err != nil {
		return nil, err
	}
	login := result
	http.SetCookie(writer, &http.Cookie{Name: "mindflow_session", Value: login.Cookie, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: sessionLifetimeSeconds})

	return map[string]any{"user": login.User, csrfKey: login.CSRF}, nil
}

func (a *App) listUsers(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.authentication.ListUsers(r.Context(), who(r))
}

func (a *App) invite(w http.ResponseWriter, request *http.Request) (any, error) {
	var value authentication.InviteCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.authentication.Invite(request.Context(), who(request), value)
}

func (a *App) resendInvite(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.authentication.ResendInvite(r.Context(), who(r), r.PathValue("id"))
}

func (a *App) adminReset(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.authentication.AdminReset(r.Context(), who(r), r.PathValue("id"))
}

func (a *App) forgot(w http.ResponseWriter, request *http.Request) (any, error) {
	if a.limited(request, "forgot") {
		return nil, problem(http.StatusTooManyRequests, "rate_limit", "Please retry in a minute.")
	}
	var value authentication.ForgotCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.authentication.Forgot(request.Context(), who(request), value)
}

func (a *App) redeem(w http.ResponseWriter, request *http.Request) (any, error) {
	if a.limited(request, "redeem") {
		return nil, problem(http.StatusTooManyRequests, "rate_limit", "Please retry in a minute.")
	}
	var value authentication.RedeemCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}
	purpose := "reset"
	if strings.HasSuffix(request.URL.Path, "accept-invite") {
		purpose = "invite"
	}

	return a.authentication.Redeem(request.Context(), who(request), purpose, value)
}

func (a *App) updateUser(w http.ResponseWriter, request *http.Request) (any, error) {
	var value authentication.UpdateUserCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.authentication.UpdateUser(request.Context(), who(request), request.PathValue("id"), value)
}

func (a *App) deleteUser(w http.ResponseWriter, request *http.Request) (any, error) {
	var value authentication.DeleteUserCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.authentication.DeleteUser(request.Context(), who(request), request.PathValue("id"), value)
}

func (a *App) updateSettings(w http.ResponseWriter, request *http.Request) (any, error) {
	var value authentication.UpdateSettingsCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.authentication.UpdateSettings(request.Context(), who(request), value)
}

//nolint:gosec // G124: callers set secure attributes; explicit local development alone permits HTTP.
func (a *App) setSessionCookie(writer http.ResponseWriter, cookie *http.Cookie) {
	if a.Config.Dev {
		cookie.Secure = false
	}
	http.SetCookie(writer, cookie)
}
