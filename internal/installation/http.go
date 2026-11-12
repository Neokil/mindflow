package installation

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
	routes.Endpoint("GET /api/v1/setup/status", false, false, handler.status)
	routes.Endpoint("POST /api/v1/setup/complete", false, false, handler.setup)
	routes.Endpoint("POST /api/v1/setup/test-email", false, false, handler.setupTestEmail)
	routes.Endpoint("GET /api/v1/admin/settings", true, true, handler.getSettings)
	routes.Endpoint("PUT /api/v1/admin/settings", true, true, handler.updateSettings)
}

func (a *httpHandler) status(_ http.ResponseWriter, request *http.Request) (any, error) {
	initialized, err := a.service.Initialized(request.Context())

	return map[string]bool{"initialized": initialized}, err
}

func (a *httpHandler) setup(w http.ResponseWriter, request *http.Request) (any, error) {
	if a.routes.Limited(request, "setup") {
		return nil, failure("rate_limit", "Please retry in a minute.")
	}
	var value SetupCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.Setup(request.Context(), a.routes.CurrentUser(request), value)
}

func (a *httpHandler) setupTestEmail(w http.ResponseWriter, request *http.Request) (any, error) {
	if a.routes.Limited(request, "setup-mail") {
		return nil, failure("rate_limit", "Please retry in a minute.")
	}
	var value SetupTestEmailCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.SetupTestEmail(request.Context(), a.routes.CurrentUser(request), value)
}

func (a *httpHandler) getSettings(_ http.ResponseWriter, request *http.Request) (any, error) {
	settings, err := a.service.Settings(request.Context())
	configured := settings.SMTP.Password != ""
	settings.SMTP.Password = ""

	return map[string]any{"settings": settings, "passwordConfigured": configured}, err
}

func (a *httpHandler) updateSettings(w http.ResponseWriter, request *http.Request) (any, error) {
	var value UpdateSettingsCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.UpdateSettings(request.Context(), a.routes.CurrentUser(request), value)
}
