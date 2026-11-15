package transfer

import (
	"encoding/json"
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
	endpoint := &handler{service: a, routes: routes}
	routes.Endpoint("POST /api/v1/workspaces/{id}/export", true, false, endpoint.export)
	routes.Endpoint("POST /api/v1/workspaces/{id}/export/snapshot", true, false, endpoint.snapshot)
	routes.Endpoint("POST /api/v1/workspaces/import/mindflow", true, false, endpoint.importNew)
	routes.Endpoint("POST /api/v1/workspaces/{id}/import/mindflow", true, false, endpoint.importPages)
}

func (endpoint *handler) export(writer http.ResponseWriter, request *http.Request) (any, error) {
	var selection Selection
	err := endpoint.routes.Decode(writer, request, &selection)
	if err != nil {
		return nil, err
	}
	var data []byte
	var notes []string
	filename, contentType := "workspace.mindflow", "application/vnd.mindflow+zip"
	switch selection.Format {
	case "", "mindflow":
		data, err = endpoint.service.Export(request.Context(), endpoint.routes.CurrentSession(request), request.PathValue("id"), selection)
	case "butterfly":
		data, notes, err = endpoint.service.ExportButterfly(request.Context(), endpoint.routes.CurrentSession(request), request.PathValue("id"), selection)
		filename, contentType = "workspace.bfly", "application/zip"
	default:
		return nil, invalid("Unsupported export format.")
	}
	if err != nil {
		return nil, err
	}
	writer.Header().Set("Content-Type", contentType)
	writer.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	if len(notes) > 0 {
		encoded, marshalErr := json.Marshal(notes)
		if marshalErr != nil {
			return nil, fmt.Errorf("encode conversion notes: %w", marshalErr)
		}
		writer.Header().Set("X-Mindflow-Conversion-Notes", string(encoded))
	}
	writer.Header().Set("Cache-Control", "no-store")
	_, err = writer.Write(data) //nolint:gosec // G705: ZIP attachment with a fixed binary Content-Type; never served as HTML.
	if err != nil {
		return nil, fmt.Errorf("write export response: %w", err)
	}

	return responseWritten{}, nil
}

func (endpoint *handler) snapshot(writer http.ResponseWriter, request *http.Request) (any, error) {
	var selection Selection
	err := endpoint.routes.Decode(writer, request, &selection)
	if err != nil {
		return nil, err
	}
	writer.Header().Set("Cache-Control", "no-store")

	return endpoint.service.Snapshot(request.Context(), endpoint.routes.CurrentSession(request), request.PathValue("id"), selection)
}

func (endpoint *handler) importNew(writer http.ResponseWriter, request *http.Request) (any, error) {
	return endpoint.importFile(writer, request, "")
}

func (endpoint *handler) importPages(writer http.ResponseWriter, request *http.Request) (any, error) {
	return endpoint.importFile(writer, request, request.PathValue("id"))
}

func (endpoint *handler) importFile(writer http.ResponseWriter, request *http.Request, target string) (any, error) {
	request.Body = http.MaxBytesReader(writer, request.Body, maxArchiveBytes+multipartMemory)
	err := request.ParseMultipartForm(multipartMemory) //nolint:gosec // G120: MaxBytesReader bounds the complete request before multipart parsing.
	if request.MultipartForm != nil {
		defer func() { _ = request.MultipartForm.RemoveAll() }()
	}
	if err != nil {
		return nil, invalid("Choose a supported notebook export up to 64 MiB.")
	}
	file, _, err := request.FormFile("file")
	if err != nil {
		return nil, invalid("Choose a supported notebook export.")
	}
	defer func() { _ = file.Close() }()

	return endpoint.service.Import(request.Context(), endpoint.routes.CurrentSession(request), file, request.FormValue("name"), target)
}

type responseWritten struct{}

// ResponseWritten prevents the HTTP boundary from adding JSON to a binary download.
func (responseWritten) ResponseWritten() {}
