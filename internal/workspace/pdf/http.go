package pdf

import (
	"io"
	"mindflow/internal/account/session"
	"net/http"
	"sync"
)

// HTTP provides the shared session, origin, CSRF and request-size boundary.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	Decode(writer http.ResponseWriter, request *http.Request, value any) error
	CurrentSession(request *http.Request) session.Principal
}

type handler struct {
	service   *Service
	routes    HTTP
	uploading sync.Mutex
}

// RegisterHTTP exposes protected PDF uploads and atomic page creation.
func (a *Service) RegisterHTTP(routes HTTP) {
	h := &handler{service: a, routes: routes}
	routes.Endpoint("POST /api/v1/workspaces/{id}/pdf-assets", true, false, h.upload)
	routes.Endpoint("POST /api/v1/workspaces/{id}/pdf-pages", true, false, h.pages)
}

func (a *handler) upload(w http.ResponseWriter, request *http.Request) (any, error) {
	if !a.uploading.TryLock() {
		return nil, invalid("Another PDF upload is running. Please retry shortly.")
	}
	defer a.uploading.Unlock()
	request.Body = http.MaxBytesReader(w, request.Body, maxPDFBytes+maxPreviewBytes+multipartOverhead)
	err := request.ParseMultipartForm(multipartMemory) //nolint:gosec // G120: MaxBytesReader bounds the complete request.
	if request.MultipartForm != nil {
		defer func() { _ = request.MultipartForm.RemoveAll() }()
	}
	if err != nil {
		return nil, invalid("Choose a PDF up to 50 MiB.")
	}
	data, err := readFile(request, "file", maxPDFBytes)
	if err != nil {
		return nil, err
	}
	preview, err := readFile(request, "preview", maxPreviewBytes)
	if err != nil {
		return nil, err
	}

	return a.service.Upload(request.Context(), a.routes.CurrentSession(request), request.PathValue("id"), data, preview)
}

func readFile(request *http.Request, field string, limit int64) ([]byte, error) {
	file, _, err := request.FormFile(field)
	if err != nil {
		return nil, invalid("PDF and preview files are required.")
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, invalid("PDF or preview exceeds its size limit.")
	}

	return data, nil
}

func (a *handler) pages(w http.ResponseWriter, request *http.Request) (any, error) {
	var command struct {
		Pages []PageImage `json:"pages"`
	}
	err := a.routes.Decode(w, request, &command)
	if err != nil {
		return nil, err
	}
	ids, err := a.service.ImportPages(request.Context(), a.routes.CurrentSession(request), request.PathValue("id"), command.Pages)

	return map[string]any{"ids": ids}, err
}

const (
	multipartMemory   = 1 << 20
	multipartOverhead = 1 << 20
)
