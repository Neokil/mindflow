package email

import (
	"mindflow/internal/account"
	"net/http"
)

// HTTP is the request boundary needed by this domain's routes.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	CurrentUser(request *http.Request) account.User
}
type httpHandler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP registers routes owned by this domain.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &httpHandler{service: a, routes: routes}
	routes.Endpoint("POST /api/v1/admin/settings/test-email", true, true, handler.testDelivery)
}

func (a *httpHandler) testDelivery(_ http.ResponseWriter, request *http.Request) (any, error) {
	err := a.service.Test(request.Context(), a.routes.CurrentUser(request))
	if err != nil {
		return nil, err
	}

	return map[string]bool{"ok": true}, nil
}
