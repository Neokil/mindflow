package page

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
	routes.Endpoint("GET /api/v1/workspaces/{id}/pages", true, false, handler.listPages)
	routes.Endpoint("POST /api/v1/workspaces/{id}/pages", true, false, handler.createPage)
	routes.Endpoint("PATCH /api/v1/pages/{id}", true, false, handler.renamePage)
	routes.Endpoint("DELETE /api/v1/pages/{id}", true, false, handler.deletePage)
	routes.Endpoint("POST /api/v1/pages/{id}/duplicate", true, false, handler.duplicatePage)
	routes.Endpoint("POST /api/v1/pages/recovery-copy", true, false, handler.recoveryCopy)
}

func (a *httpHandler) listPages(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.service.ListPages(r.Context(), a.routes.CurrentSession(r), r.PathValue("id"))
}

func (a *httpHandler) createPage(w http.ResponseWriter, request *http.Request) (any, error) {
	var value CreatePageCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.CreatePage(request.Context(), a.routes.CurrentSession(request), request.PathValue("id"), value)
}

func (a *httpHandler) renamePage(w http.ResponseWriter, request *http.Request) (any, error) {
	var value RenamePageCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.RenamePage(request.Context(), a.routes.CurrentSession(request), request.PathValue("id"), value)
}

func (a *httpHandler) deletePage(w http.ResponseWriter, request *http.Request) (any, error) {
	var value DeletePageCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.DeletePage(request.Context(), a.routes.CurrentSession(request), request.PathValue("id"), value)
}

func (a *httpHandler) duplicatePage(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.service.DuplicatePage(r.Context(), a.routes.CurrentSession(r), r.PathValue("id"))
}

func (a *httpHandler) recoveryCopy(w http.ResponseWriter, request *http.Request) (any, error) {
	var value RecoveryCopyCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.RecoveryCopy(request.Context(), a.routes.CurrentSession(request), value)
}
