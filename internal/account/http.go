package account

import (
	"net/http"
)

// HTTP is the request boundary needed by this domain's routes.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	Decode(writer http.ResponseWriter, request *http.Request, value any) error
	CurrentUser(request *http.Request) User
}
type httpHandler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP registers routes owned by this domain.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &httpHandler{service: a, routes: routes}
	routes.Endpoint("GET /api/v1/admin/users", true, true, handler.listUsers)
	routes.Endpoint("PATCH /api/v1/admin/users/{id}", true, true, handler.updateUser)
	routes.Endpoint("DELETE /api/v1/admin/users/{id}", true, true, handler.deleteUser)
	routes.Endpoint("PATCH /api/v1/auth/preferences", true, false, handler.preferences)
	routes.Endpoint("PATCH /api/v1/auth/profile", true, false, handler.profile)
	routes.Endpoint("GET /api/v1/accounts/directory", true, false, handler.directory)
}

func (a *httpHandler) listUsers(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.service.ListUsers(r.Context(), a.routes.CurrentUser(r))
}

func (a *httpHandler) updateUser(w http.ResponseWriter, request *http.Request) (any, error) {
	var value UpdateUserCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.UpdateUser(request.Context(), a.routes.CurrentUser(request), request.PathValue("id"), value)
}

func (a *httpHandler) deleteUser(w http.ResponseWriter, request *http.Request) (any, error) {
	var value DeleteUserCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.DeleteUser(request.Context(), a.routes.CurrentUser(request), request.PathValue("id"), value)
}

func (a *httpHandler) preferences(writer http.ResponseWriter, request *http.Request) (any, error) {
	var value struct {
		Appearance string `json:"appearance"`
	}
	err := a.routes.Decode(writer, request, &value)
	if err != nil {
		return nil, err
	}
	err = a.service.SetAppearance(request.Context(), a.routes.CurrentUser(request), value.Appearance)

	return map[string]bool{"ok": true}, err
}

func (a *httpHandler) profile(writer http.ResponseWriter, request *http.Request) (any, error) {
	var command ProfileCommand
	err := a.routes.Decode(writer, request, &command)
	if err != nil {
		return nil, err
	}

	return a.service.UpdateProfile(request.Context(), a.routes.CurrentUser(request), command)
}

func (a *httpHandler) directory(_ http.ResponseWriter, request *http.Request) (any, error) {
	return a.service.Directory(request.Context())
}
