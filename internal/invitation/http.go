package invitation

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
	routes.Endpoint("POST /api/v1/admin/users", true, true, handler.invite)
	routes.Endpoint("POST /api/v1/admin/users/{id}/invite", true, true, handler.resendInvite)
	routes.Endpoint("POST /api/v1/auth/accept-invite", false, false, handler.redeem)
}

func (a *httpHandler) invite(w http.ResponseWriter, request *http.Request) (any, error) {
	var value InviteCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.Invite(request.Context(), a.routes.CurrentUser(request), value)
}

func (a *httpHandler) resendInvite(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.service.ResendInvite(r.Context(), a.routes.CurrentUser(r), r.PathValue("id"))
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
