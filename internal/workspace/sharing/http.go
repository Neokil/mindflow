package sharing

import (
	"mindflow/internal/account/session"
	"net/http"
)

// HTTP supplies authenticated protocol mechanics.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	Decode(writer http.ResponseWriter, request *http.Request, value any) error
	CurrentSession(request *http.Request) session.Principal
}
type handler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP registers owner-managed sharing routes.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &handler{service: a, routes: routes}
	routes.Endpoint("GET /api/v1/workspaces/{id}/members", true, false, handler.list)
	routes.Endpoint("PUT /api/v1/workspaces/{id}/members", true, false, handler.set)
	routes.Endpoint("DELETE /api/v1/workspaces/{id}/members/{user}", true, false, handler.remove)
}

func (handler *handler) list(_ http.ResponseWriter, request *http.Request) (any, error) {
	return handler.service.List(request.Context(), handler.routes.CurrentSession(request), request.PathValue("id"))
}

func (handler *handler) set(writer http.ResponseWriter, request *http.Request) (any, error) {
	var command Command
	err := handler.routes.Decode(writer, request, &command)
	if err != nil {
		return nil, err
	}

	return handler.service.Set(request.Context(), handler.routes.CurrentSession(request), request.PathValue("id"), command)
}

func (handler *handler) remove(_ http.ResponseWriter, request *http.Request) (any, error) {
	return handler.service.Remove(request.Context(), handler.routes.CurrentSession(request), request.PathValue("id"), request.PathValue("user"))
}
