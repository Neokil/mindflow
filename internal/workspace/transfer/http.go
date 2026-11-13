package transfer

import (
	"fmt"
	"mime"
	"mindflow/internal/account/session"
	"net/http"
)

// HTTP supplies authenticated, CSRF-protected transfer endpoints.
type HTTP interface {
	Endpoint(pattern string, auth bool, admin bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	CurrentSession(request *http.Request) session.Principal
	Decode(writer http.ResponseWriter, request *http.Request, target any) error
}
type handler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP exposes native transfer and consistent visual-export snapshots.
func (a *Service) RegisterHTTP(routes HTTP) {
	h := &handler{service: a, routes: routes}
	routes.Endpoint("POST /api/v1/workspaces/{id}/export", true, false, h.export)
	routes.Endpoint("POST /api/v1/workspaces/{id}/export/snapshot", true, false, h.snapshot)
	routes.Endpoint("POST /api/v1/workspaces/import/mindflow", true, false, h.importNew)
	routes.Endpoint("POST /api/v1/workspaces/{id}/import/mindflow", true, false, h.importPages)
}

func (h *handler) export(writer http.ResponseWriter, request *http.Request) (any, error) {
	var selection Selection
	err := h.routes.Decode(writer, request, &selection)
	if err != nil {
		return nil, err
	}
	data, err := h.service.Export(request.Context(), h.routes.CurrentSession(request), request.PathValue("id"), selection)
	if err != nil {
		return nil, err
	}
	writer.Header().Set("Content-Type", "application/vnd.mindflow+zip")
	writer.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "workspace.mindflow"}))
	writer.Header().Set("Cache-Control", "no-store")
	_, err = writer.Write(data) //nolint:gosec // G705: ZIP attachment with a fixed binary Content-Type; never served as HTML.
	if err != nil {
		return nil, fmt.Errorf("write export response: %w", err)
	}

	return responseWritten{}, nil
}

func (h *handler) snapshot(writer http.ResponseWriter, request *http.Request) (any, error) {
	var selection Selection
	err := h.routes.Decode(writer, request, &selection)
	if err != nil {
		return nil, err
	}
	writer.Header().Set("Cache-Control", "no-store")

	return h.service.Snapshot(request.Context(), h.routes.CurrentSession(request), request.PathValue("id"), selection)
}

func (h *handler) importNew(writer http.ResponseWriter, request *http.Request) (any, error) {
	return h.importFile(writer, request, "")
}

func (h *handler) importPages(writer http.ResponseWriter, request *http.Request) (any, error) {
	return h.importFile(writer, request, request.PathValue("id"))
}

func (h *handler) importFile(writer http.ResponseWriter, request *http.Request, target string) (any, error) {
	request.Body = http.MaxBytesReader(writer, request.Body, maxArchiveBytes+multipartMemory)
	err := request.ParseMultipartForm(multipartMemory) //nolint:gosec // G120: MaxBytesReader bounds the complete request before multipart parsing.
	if request.MultipartForm != nil {
		defer func() { _ = request.MultipartForm.RemoveAll() }()
	}
	if err != nil {
		return nil, invalid("Choose a .mindflow export up to 64 MiB.")
	}
	file, _, err := request.FormFile("file")
	if err != nil {
		return nil, invalid("Choose a .mindflow export.")
	}
	defer func() { _ = file.Close() }()

	return h.service.Import(request.Context(), h.routes.CurrentSession(request), file, request.FormValue("name"), target)
}

type responseWritten struct{}

// ResponseWritten prevents the HTTP boundary from adding JSON to a binary download.
func (responseWritten) ResponseWritten() {}
