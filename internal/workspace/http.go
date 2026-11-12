package workspace

import (
	"mindflow/internal/account/session"
	"net/http"
)

// HTTP is the request boundary needed by this domain's routes.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	Decode(writer http.ResponseWriter, request *http.Request, value any) error
	CurrentSession(request *http.Request) session.Principal
}
type httpHandler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP registers routes owned by this domain.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &httpHandler{service: a, routes: routes}
	routes.Endpoint("GET /api/v1/workspaces", true, false, handler.listWorkspaces)
	routes.Endpoint("POST /api/v1/workspaces", true, false, handler.createWorkspace)
	routes.Endpoint("PATCH /api/v1/workspaces/{id}", true, false, handler.renameWorkspace)
	routes.Endpoint("DELETE /api/v1/workspaces/{id}", true, false, handler.deleteWorkspace)
	routes.Endpoint("POST /api/v1/workspaces/{id}/duplicate", true, false, handler.duplicateWorkspace)
	routes.Endpoint("PUT /api/v1/workspaces/{id}/page-order", true, false, handler.reorderPages)
}

func (a *httpHandler) listWorkspaces(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.service.ListWorkspaces(r.Context(), a.routes.CurrentSession(r))
}

func (a *httpHandler) createWorkspace(w http.ResponseWriter, request *http.Request) (any, error) {
	var value CreateWorkspaceCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.CreateWorkspace(request.Context(), a.routes.CurrentSession(request), value)
}

func (a *httpHandler) renameWorkspace(w http.ResponseWriter, request *http.Request) (any, error) {
	var value RenameWorkspaceCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.RenameWorkspace(request.Context(), a.routes.CurrentSession(request), request.PathValue("id"), value)
}

func (a *httpHandler) deleteWorkspace(w http.ResponseWriter, request *http.Request) (any, error) {
	var value DeleteWorkspaceCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.DeleteWorkspace(request.Context(), a.routes.CurrentSession(request), request.PathValue("id"), value)
}

func (a *httpHandler) duplicateWorkspace(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.service.DuplicateWorkspace(r.Context(), a.routes.CurrentSession(r), r.PathValue("id"))
}

func (a *httpHandler) reorderPages(w http.ResponseWriter, request *http.Request) (any, error) {
	var value ReorderPagesCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.ReorderPages(request.Context(), a.routes.CurrentSession(request), request.PathValue("id"), value)
}
