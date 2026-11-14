package ownership

import (
	"mindflow/internal/account/session"
	"net/http"
)

// HTTP supplies authenticated protocol mechanics.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	CurrentSession(request *http.Request) session.Principal
	Decode(writer http.ResponseWriter, request *http.Request, value any) error
}

// RegisterHTTP exposes explicit owner-only transfers.
func (a *Service) RegisterHTTP(routes HTTP) {
	routes.Endpoint("POST /api/v1/workspaces/{id}/ownership", true, false, func(w http.ResponseWriter, request *http.Request) (any, error) {
		var command Command
		err := routes.Decode(w, request, &command)
		if err != nil {
			return nil, err
		}

		return a.Transfer(request.Context(), routes.CurrentSession(request), request.PathValue("id"), command)
	})
}
