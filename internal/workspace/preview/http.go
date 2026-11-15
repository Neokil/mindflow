package preview

import (
	"fmt"
	"io"
	"mindflow/internal/account/session"
	"net/http"
	"strconv"
)

// HTTP supplies authenticated, CSRF-protected routes.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	CurrentSession(request *http.Request) session.Principal
}

// RegisterHTTP exposes metadata and immutable image representations.
func (a *Service) RegisterHTTP(routes HTTP) {
	routes.Endpoint("GET /api/v1/workspaces/{id}/preview", true, false, func(_ http.ResponseWriter, request *http.Request) (any, error) {
		return a.Inspect(request.Context(), routes.CurrentSession(request).ID, request.PathValue("id"))
	})
	routes.Endpoint("POST /api/v1/workspaces/{id}/preview", true, false, func(writer http.ResponseWriter, request *http.Request) (any, error) {
		request.Body = http.MaxBytesReader(writer, request.Body, multipartLimit)
		err := request.ParseMultipartForm(maxPreviewBytes) //nolint:gosec // G120: the complete request is bounded by MaxBytesReader.
		if request.MultipartForm != nil {
			defer func() { _ = request.MultipartForm.RemoveAll() }()
		}
		if err != nil {
			return nil, invalid("Choose a preview up to 2 MiB.")
		}
		file, _, err := request.FormFile("file")
		if err != nil {
			return nil, invalid("Choose a PNG preview.")
		}
		defer func() { _ = file.Close() }()
		raw, err := io.ReadAll(io.LimitReader(file, maxPreviewBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read workspace preview: %w", err)
		}
		revision, err := strconv.Atoi(request.FormValue("revision"))
		if err != nil {
			return nil, invalid("Invalid page revision.")
		}

		return a.Store(request.Context(), routes.CurrentSession(request).ID, request.PathValue("id"), request.FormValue("pageId"), revision, raw)
	})
	routes.Endpoint("GET /api/v1/workspaces/{id}/previews/{content}", true, false, func(writer http.ResponseWriter, request *http.Request) (any, error) {
		raw, err := a.Get(request.Context(), routes.CurrentSession(request).ID, request.PathValue("id"), request.PathValue("content"))
		if err != nil {
			return nil, err
		}
		writer.Header().Set("Content-Type", "image/png")
		writer.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		_, err = writer.Write(raw) //nolint:gosec // G705: normalized PNG with a fixed image/png type.
		if err != nil {
			return nil, fmt.Errorf("write workspace preview: %w", err)
		}

		return responseWritten{}, nil
	})
}

type responseWritten struct{}

// ResponseWritten indicates an already written binary response.
func (responseWritten) ResponseWritten() {}

const multipartLimit = 3 << 20
