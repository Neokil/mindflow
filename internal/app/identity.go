package app

import (
	"net/http"
	"strings"

	"mindflow/internal/domain"
)

func (a *App) identityRoutes() {
	a.endpoint("GET /api/v1/setup/status", false, false, func(w http.ResponseWriter, r *http.Request) (any, error) {
		initialized, err := a.service.Initialized(r.Context())
		return map[string]bool{"initialized": initialized}, err
	})
	a.endpoint("POST /api/v1/setup/complete", false, false, a.setup)
	a.endpoint("POST /api/v1/setup/test-email", false, false, a.setupTestEmail)
	a.endpoint("POST /api/v1/auth/login", false, false, a.login)
	a.endpoint("POST /api/v1/auth/logout", true, false, func(w http.ResponseWriter, r *http.Request) (any, error) {
		err := a.service.Logout(r.Context(), who(r))
		http.SetCookie(w, &http.Cookie{Name: "mindflow_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: !a.Config.Dev, SameSite: http.SameSiteLaxMode})
		return map[string]bool{"ok": true}, err
	})
	a.endpoint("GET /api/v1/auth/me", true, false, func(w http.ResponseWriter, r *http.Request) (any, error) {
		p := who(r)
		return map[string]any{"user": p.User, "csrf": p.CSRF}, nil
	})
	a.endpoint("PATCH /api/v1/auth/preferences", true, false, func(w http.ResponseWriter, r *http.Request) (any, error) {
		var v struct {
			Appearance string `json:"appearance"`
		}
		if err := decode(w, r, &v); err != nil {
			return nil, err
		}
		if v.Appearance != "system" && v.Appearance != "light" && v.Appearance != "dark" {
			return nil, bad("Invalid appearance.")
		}
		err := a.service.SetAppearance(r.Context(), who(r), v.Appearance)
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
	a.endpoint("GET /api/v1/admin/settings", true, true, func(w http.ResponseWriter, r *http.Request) (any, error) {
		s, err := a.service.Settings(r.Context())
		configured := s.SMTP.Password != ""
		s.SMTP.Password = ""
		return map[string]any{"settings": s, "passwordConfigured": configured}, err
	})
	a.endpoint("PUT /api/v1/admin/settings", true, true, a.updateSettings)
	a.endpoint("POST /api/v1/admin/settings/test-email", true, true, func(w http.ResponseWriter, r *http.Request) (any, error) {
		if err := a.service.SendTestEmail(r.Context(), who(r)); err != nil {
			return nil, err
		}
		return map[string]bool{"ok": true}, nil
	})
}
func (a *App) setup(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.limited(r, "setup") {
		return nil, problem(429, "rate_limit", "Please retry in a minute.")
	}
	var v domain.SetupCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.Setup(r.Context(), who(r), v)
}
func (a *App) setupTestEmail(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.limited(r, "setup-mail") {
		return nil, problem(429, "rate_limit", "Please retry in a minute.")
	}
	var v domain.SetupTestEmailCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.SetupTestEmail(r.Context(), who(r), v)
}
func (a *App) login(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.limited(r, "login") {
		return nil, problem(429, "rate_limit", "Too many attempts. Please retry in a minute.")
	}
	var v domain.LoginCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	result, err := a.service.Login(r.Context(), who(r), v)
	if err != nil {
		return nil, err
	}
	login := result
	http.SetCookie(w, &http.Cookie{Name: "mindflow_session", Value: login.Cookie, Path: "/", HttpOnly: true, Secure: !a.Config.Dev, SameSite: http.SameSiteLaxMode, MaxAge: 30 * 86400})
	return map[string]any{"user": login.User, "csrf": login.CSRF}, nil
}
func (a *App) listUsers(w http.ResponseWriter, r *http.Request) (any, error) {

	return a.service.ListUsers(r.Context(), who(r))
}
func (a *App) invite(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.InviteCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.Invite(r.Context(), who(r), v)
}
func (a *App) resendInvite(w http.ResponseWriter, r *http.Request) (any, error) {

	return a.service.ResendInvite(r.Context(), who(r), r.PathValue("id"))
}
func (a *App) adminReset(w http.ResponseWriter, r *http.Request) (any, error) {

	return a.service.AdminReset(r.Context(), who(r), r.PathValue("id"))
}
func (a *App) forgot(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.limited(r, "forgot") {
		return nil, problem(429, "rate_limit", "Please retry in a minute.")
	}
	var v domain.ForgotCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.Forgot(r.Context(), who(r), v)
}
func (a *App) redeem(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.limited(r, "redeem") {
		return nil, problem(429, "rate_limit", "Please retry in a minute.")
	}
	var v domain.RedeemCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	purpose := "reset"
	if strings.HasSuffix(r.URL.Path, "accept-invite") {
		purpose = "invite"
	}
	return a.service.Redeem(r.Context(), who(r), purpose, v)
}
func (a *App) updateUser(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.UpdateUserCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.UpdateUser(r.Context(), who(r), r.PathValue("id"), v)
}
func (a *App) deleteUser(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.DeleteUserCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.DeleteUser(r.Context(), who(r), r.PathValue("id"), v)
}
func (a *App) updateSettings(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.UpdateSettingsCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.UpdateSettings(r.Context(), who(r), v)
}
