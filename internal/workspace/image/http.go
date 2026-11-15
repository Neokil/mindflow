package image

import (
	"io"
	"mindflow/internal/account/session"
	"net/http"
	"strconv"
)

// HTTP is the request boundary needed by this domain's routes.
type HTTP interface {
	Endpoint(pattern string, authenticated bool, administrator bool, handler func(http.ResponseWriter, *http.Request) (any, error))
	Decode(writer http.ResponseWriter, request *http.Request, value any) error
	CurrentSession(request *http.Request) session.Principal
}
type httpHandler struct {
	service *Service
	routes  HTTP
}

// RegisterHTTP registers routes owned by this domain.
func (a *Service) RegisterHTTP(routes HTTP) {
	handler := &httpHandler{service: a, routes: routes}
	routes.Endpoint("POST /api/v1/assets", true, false, handler.uploadAsset)
	routes.Endpoint("GET /api/v1/assets/{id}", true, false, handler.getAsset)
	routes.Endpoint("GET /api/v1/assets/{id}/meta", true, false, handler.assetMeta)
	routes.Endpoint("POST /api/v1/assets/{id}/variant", true, false, handler.requestVariant)
}

func (a *httpHandler) uploadAsset(w http.ResponseWriter, request *http.Request) (any, error) {
	request.Body = http.MaxBytesReader(w, request.Body, multipartRequestBytes)
	parseMultipartFormErr := request.ParseMultipartForm(multipartMemoryBytes) //nolint:gosec // G120: MaxBytesReader bounds the complete multipart body above.
	if parseMultipartFormErr != nil {
		return nil, failure("upload_limit", "Choose an image up to 20 MB.")
	}
	if request.MultipartForm != nil {
		defer func() { _ = request.MultipartForm.RemoveAll() }()
	}
	file, _, err := request.FormFile("file")
	if err != nil {
		return nil, invalid("Choose an image file.")
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		return nil, wrap("media.go", err)
	}
	if len(raw) > maxUploadBytes {
		return nil, failure("upload_limit", "Choose an image up to 20 MB.")
	}

	return a.service.UploadWorkspaceAsset(request.Context(), a.routes.CurrentSession(request), request.FormValue("workspaceId"), raw)
}

func (a *httpHandler) getAsset(writer http.ResponseWriter, r *http.Request) (any, error) {
	bucket, _ := strconv.Atoi(r.URL.Query().Get("bucket"))
	blob, err := a.service.GetAsset(r.Context(), a.routes.CurrentSession(r), r.PathValue("id"), bucket)
	if err != nil {
		return nil, err
	}
	if blob.Immutable {
		writer.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	}
	writer.Header().Set("Content-Type", blob.MIME)
	if blob.MIME == "application/pdf" {
		writer.Header().Set("Content-Disposition", "attachment; filename=document.pdf")
	}
	writer.Header().Set("Content-Length", strconv.Itoa(len(blob.Data)))
	writer.Header().Set("X-Image-Width", strconv.Itoa(blob.Width))
	writer.Header().Set("X-Image-Height", strconv.Itoa(blob.Height))
	_, _ = writer.Write(blob.Data)

	return responseWritten{}, nil
}

func (a *httpHandler) assetMeta(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.service.AssetMeta(r.Context(), a.routes.CurrentSession(r), r.PathValue("id"))
}

func (a *httpHandler) requestVariant(w http.ResponseWriter, request *http.Request) (any, error) {
	var value RequestVariantCommand
	err := a.routes.Decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.service.RequestVariant(request.Context(), a.routes.CurrentSession(request), request.PathValue("id"), value)
}

type responseWritten struct{}

// ResponseWritten marks a body already written by the domain handler.
func (responseWritten) ResponseWritten() {}

const (
	multipartRequestBytes = 21 << 20
	multipartMemoryBytes  = 1 << 20
	maxUploadBytes        = 20 << 20
)
