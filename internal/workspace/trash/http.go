package trash

import (
	"mindflow/internal/account/session"
	"net/http"
)

// HTTP supplies authenticated, CSRF-protected routes.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	CurrentSession(request *http.Request) session.Principal
}

// RegisterHTTP exposes the current user's trash and restoration operations.
func (a *Service) RegisterHTTP(routes HTTP) {
	routes.Endpoint("GET /api/v1/trash", true, false, func(_ http.ResponseWriter, r *http.Request) (any, error) {
		return a.List(r.Context(), routes.CurrentSession(r))
	})
	for _, kind := range []string{kindWorkspace, "page"} {
		routes.Endpoint("POST /api/v1/trash/"+kind+"s/{id}/restore", true, false, func(_ http.ResponseWriter, r *http.Request) (any, error) {
			return a.Restore(r.Context(), routes.CurrentSession(r), kind, r.PathValue("id"))
		})
	}
}
