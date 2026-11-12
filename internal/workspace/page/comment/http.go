package comment

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

// RegisterHTTP registers comment thread operations.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &handler{service: a, routes: routes}
	routes.Endpoint("GET /api/v1/pages/{id}/comments", true, false, handler.list)
	routes.Endpoint("POST /api/v1/pages/{id}/comments", true, false, handler.create)
	routes.Endpoint("POST /api/v1/comments/{id}/replies", true, false, handler.reply)
	routes.Endpoint("PATCH /api/v1/comments/{id}", true, false, handler.resolve)
	routes.Endpoint("DELETE /api/v1/comments/{id}", true, false, handler.remove)
}

func (handler *handler) list(_ http.ResponseWriter, request *http.Request) (any, error) {
	return handler.service.List(request.Context(), handler.routes.CurrentSession(request), request.PathValue("id"))
}

func (handler *handler) create(writer http.ResponseWriter, request *http.Request) (any, error) {
	var command CreateCommand
	err := handler.routes.Decode(writer, request, &command)
	if err != nil {
		return nil, err
	}

	return handler.service.Create(request.Context(), handler.routes.CurrentSession(request), request.PathValue("id"), command)
}

func (handler *handler) reply(writer http.ResponseWriter, request *http.Request) (any, error) {
	var command ReplyCommand
	err := handler.routes.Decode(writer, request, &command)
	if err != nil {
		return nil, err
	}

	return handler.service.Reply(request.Context(), handler.routes.CurrentSession(request), request.PathValue("id"), command)
}

func (handler *handler) resolve(writer http.ResponseWriter, request *http.Request) (any, error) {
	var command ResolveCommand
	err := handler.routes.Decode(writer, request, &command)
	if err != nil {
		return nil, err
	}

	return handler.service.Manage(request.Context(), handler.routes.CurrentSession(request), request.PathValue("id"), command.Resolved, false)
}

func (handler *handler) remove(_ http.ResponseWriter, request *http.Request) (any, error) {
	return handler.service.Manage(request.Context(), handler.routes.CurrentSession(request), request.PathValue("id"), false, true)
}
