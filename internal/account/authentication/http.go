package authentication

import (
	"net/http"
)

// HTTP is the request boundary needed by this domain's routes.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	Decode(writer http.ResponseWriter, request *http.Request, value any) error
	Limited(request *http.Request, key string) bool
}
type httpHandler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP registers routes owned by this domain.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &httpHandler{service: a, routes: routes}
	routes.Endpoint("POST /api/v1/auth/login", false, false, handler.login)
	routes.Endpoint("POST /api/v1/auth/logout", true, false, handler.logout)
	routes.Endpoint("GET /api/v1/auth/me", true, false, handler.me)
}

func (a *httpHandler) login(writer http.ResponseWriter, request *http.Request) (any, error) {
	if a.routes.Limited(request, "login") {
		return nil, failure("rate_limit", "Too many attempts. Please retry in a minute.")
	}
	var v LoginCommand
	decodeErr2 := a.routes.Decode(writer, request, &v)
	if decodeErr2 != nil {
		return nil, decodeErr2
	}
	result, err := a.service.Login(request.Context(), v)
	if err != nil {
		return nil, err
	}
	login := result
	a.service.setSessionCookie(writer, &http.Cookie{Name: "mindflow_session", Value: login.Cookie, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: sessionLifetimeSeconds})

	return map[string]any{"user": login.User, csrfKey: login.CSRF}, nil
}

func (a *httpHandler) logout(writer http.ResponseWriter, request *http.Request) (any, error) {
	err := a.service.sessions.Logout(request.Context(), Current(request.Context()))
	a.service.setSessionCookie(writer, &http.Cookie{Name: "mindflow_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})

	return map[string]bool{"ok": true}, err
}

func (a *httpHandler) me(_ http.ResponseWriter, request *http.Request) (any, error) {
	actor := Current(request.Context())

	return map[string]any{"user": actor.User, "csrf": actor.CSRF}, nil
}

//nolint:gosec // G124: callers set secure attributes; explicit local development alone permits HTTP.
func (a *Service) setSessionCookie(writer http.ResponseWriter, cookie *http.Cookie) {
	if a.dev {
		cookie.Secure = false
	}
	http.SetCookie(writer, cookie)
}

const (
	sessionLifetimeSeconds = 30 * 86400
	csrfKey                = "csrf"
)
