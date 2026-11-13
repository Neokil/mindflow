package template

import (
	"mindflow/internal/account/session"
	"net/http"
)

// HTTP supplies common authentication, CSRF, origin and JSON handling.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	Decode(writer http.ResponseWriter, request *http.Request, value any) error
	CurrentSession(request *http.Request) session.Principal
}
type handler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP registers workspace-scoped template operations.
func (a *Service) RegisterHTTP(routes HTTP) {
	h := &handler{service: a, routes: routes}
	routes.Endpoint("GET /api/v1/workspaces/{id}/templates", true, false, h.list)
	routes.Endpoint("POST /api/v1/workspaces/{id}/templates", true, false, h.create)
	routes.Endpoint("GET /api/v1/workspaces/{id}/templates/{template}", true, false, h.get)
	routes.Endpoint("DELETE /api/v1/workspaces/{id}/templates/{template}", true, false, h.remove)
}

func (h *handler) list(_ http.ResponseWriter, request *http.Request) (any, error) {
	return h.service.List(request.Context(), h.routes.CurrentSession(request), request.PathValue("id"))
}

func (h *handler) get(_ http.ResponseWriter, request *http.Request) (any, error) {
	return h.service.Get(request.Context(), h.routes.CurrentSession(request), request.PathValue("id"), request.PathValue("template"))
}

func (h *handler) create(writer http.ResponseWriter, request *http.Request) (any, error) {
	var command CreateCommand
	err := h.routes.Decode(writer, request, &command)
	if err != nil {
		return nil, err
	}

	return h.service.Create(request.Context(), h.routes.CurrentSession(request), request.PathValue("id"), command)
}

func (h *handler) remove(_ http.ResponseWriter, request *http.Request) (any, error) {
	return h.service.Remove(request.Context(), h.routes.CurrentSession(request), request.PathValue("id"), request.PathValue("template"))
}
