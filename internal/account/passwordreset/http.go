package passwordreset

import (
	"mindflow/internal/account"
	"net/http"
)

// HTTP is the request boundary needed by this domain's routes.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	Decode(writer http.ResponseWriter, request *http.Request, value any) error
	Limited(request *http.Request, key string) bool
	CurrentUser(request *http.Request) account.User
}
type httpHandler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP registers routes owned by this domain.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &httpHandler{service: a, routes: routes}
	routes.Endpoint("POST /api/v1/auth/forgot-password", false, false, handler.forgot)
	routes.Endpoint("POST /api/v1/auth/reset-password", false, false, handler.redeem)
	routes.Endpoint("POST /api/v1/admin/users/{id}/reset", true, true, handler.adminReset)
}

func (a *httpHandler) forgot(w http.ResponseWriter, request *http.Request) (any, error) {
	if a.routes.Limited(request, "forgot") {
		return nil, failure("rate_limit", "Please retry in a minute.")
	}
	var value ForgotCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.Forgot(request.Context(), value)
}

func (a *httpHandler) redeem(writer http.ResponseWriter, request *http.Request) (any, error) {
	if a.routes.Limited(request, "redeem") {
		return nil, failure("rate_limit", "Please retry in a minute.")
	}
	var command RedeemCommand
	err := a.routes.Decode(writer, request, &command)
	if err != nil {
		return nil, err
	}

	return a.service.Redeem(request.Context(), command)
}

func (a *httpHandler) adminReset(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.service.AdminReset(r.Context(), a.routes.CurrentUser(r), r.PathValue("id"))
}
