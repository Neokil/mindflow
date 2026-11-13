package avatar

import (
	"io"
	"mindflow/internal/account"
	"net/http"
)

// HTTP supplies authentication and CSRF-protected endpoints.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	CurrentUser(request *http.Request) account.User
}
type handler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP exposes authenticated avatar viewing and self-only changes.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &handler{service: a, routes: routes}
	routes.Endpoint("GET /api/v1/accounts/{id}/avatar", true, false, handler.get)
	routes.Endpoint("POST /api/v1/auth/avatar", true, false, handler.set)
	routes.Endpoint("DELETE /api/v1/auth/avatar", true, false, handler.remove)
}

func (a *handler) get(writer http.ResponseWriter, request *http.Request) (any, error) {
	content, err := a.service.Get(request.Context(), request.PathValue("id"))
	if err != nil {
		return nil, err
	}
	writer.Header().Set("Content-Type", "image/png")
	_, _ = writer.Write(content) //nolint:gosec // G705: fixed image/png content type; bytes are normalized by our PNG encoder.

	return responseWritten{}, nil
}

func (a *handler) set(writer http.ResponseWriter, request *http.Request) (any, error) {
	request.Body = http.MaxBytesReader(writer, request.Body, multipartLimit)
	err := request.ParseMultipartForm(maxUploadBytes) //nolint:gosec // G120: MaxBytesReader above bounds the complete body.
	if request.MultipartForm != nil {
		defer func() { _ = request.MultipartForm.RemoveAll() }()
	}
	if err != nil {
		return nil, &account.Error{Code: validationCode, Message: "Choose an avatar up to 2 MB."}
	}
	file, _, err := request.FormFile("file")
	if err != nil {
		return nil, &account.Error{Code: validationCode, Message: "Choose an avatar image."}
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil || len(raw) == 0 {
		return nil, &account.Error{Code: validationCode, Message: "Could not read avatar."}
	}

	return a.service.Set(request.Context(), a.routes.CurrentUser(request).ID, raw)
}

func (a *handler) remove(_ http.ResponseWriter, request *http.Request) (any, error) {
	return a.service.Set(request.Context(), a.routes.CurrentUser(request).ID, nil)
}

type responseWritten struct{}

// ResponseWritten marks a directly written image response.
func (responseWritten) ResponseWritten() {}

const multipartLimit = 3 << 20
