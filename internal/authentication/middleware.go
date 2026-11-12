package authentication

import (
	"context"
	"crypto/subtle"
	"mindflow/internal/account"
	"mindflow/internal/session"
	"net/http"
)

type principalKey struct{}

// Current returns the authenticated caller installed by Middleware.
func Current(ctx context.Context) session.Principal {
	actor, _ := ctx.Value(principalKey{}).(session.Principal)

	return actor
}

// Middleware verifies the session on every request and checks CSRF on writes.
// Response formatting belongs to the HTTP adapter supplied through reject.
func (a *Service) Middleware(reject func(http.ResponseWriter, *http.Request, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			cookie, err := request.Cookie("mindflow_session")
			if err != nil {
				reject(writer, request, failure("session_expired", "Please sign in."))

				return
			}
			actor, err := a.Authenticate(request.Context(), cookie.Value)
			if err != nil {
				reject(writer, request, err)

				return
			}
			if request.Method != http.MethodGet && request.Method != http.MethodHead && subtle.ConstantTimeCompare([]byte(request.Header.Get("X-CSRF-Token")), []byte(actor.CSRF)) != 1 {
				reject(writer, request, failure("csrf", "Refresh the page and try again."))

				return
			}
			next.ServeHTTP(writer, request.WithContext(context.WithValue(request.Context(), principalKey{}, actor)))
		})
	}
}

// RequireAdministrator rejects callers without an active administrator account.
func RequireAdministrator(reject func(http.ResponseWriter, *http.Request, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			err := account.RequireAdministrator(Current(request.Context()).User)
			if err != nil {
				reject(writer, request, err)

				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}
