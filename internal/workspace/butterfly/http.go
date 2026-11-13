package butterfly

import (
	"mindflow/internal/account/session"
	"net/http"
)

// HTTP provides the same authentication, origin and CSRF middleware as other mutations.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	CurrentSession(request *http.Request) session.Principal
}
type handler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP exposes authenticated workspace import.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &handler{service: a, routes: routes}
	routes.Endpoint("POST /api/v1/workspaces/import/butterfly", true, false, handler.importWorkspace)
}

func (a *handler) importWorkspace(writer http.ResponseWriter, request *http.Request) (any, error) {
	request.Body = http.MaxBytesReader(writer, request.Body, maxMultipartBytes)
	err := request.ParseMultipartForm(multipartMemoryBytes) //nolint:gosec // G120: MaxBytesReader bounds the complete multipart request above.
	if request.MultipartForm != nil {
		defer func() { _ = request.MultipartForm.RemoveAll() }()
	}
	if err != nil {
		return nil, invalid("Choose a Butterfly export up to 64 MiB.")
	}
	file, header, err := request.FormFile("file")
	if err != nil {
		return nil, invalid("Choose a Butterfly .bfly export.")
	}
	defer func() { _ = file.Close() }()

	return a.service.Import(request.Context(), a.routes.CurrentSession(request), file, request.FormValue("name"), header.Filename)
}

const (
	maxMultipartBytes    = 65 << 20
	multipartMemoryBytes = 1 << 20
)
