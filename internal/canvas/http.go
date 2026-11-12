package canvas

import (
	"mindflow/internal/session"
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
	routes.Endpoint("GET /api/v1/pages/{id}/document", true, false, handler.getDocument)
	routes.Endpoint("PUT /api/v1/pages/{id}/document", true, false, handler.saveDocument)
}

func (a *httpHandler) getDocument(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.service.GetDocument(r.Context(), a.routes.CurrentSession(r), r.PathValue("id"))
}

func (a *httpHandler) saveDocument(w http.ResponseWriter, request *http.Request) (any, error) {
	var value SaveDocumentCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.SaveDocument(request.Context(), a.routes.CurrentSession(request), request.PathValue("id"), value)
}
