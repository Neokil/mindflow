package app

import (
	"io"
	"mindflow/internal/fault"
	"mindflow/internal/workspace"
	"net/http"
	"strconv"
)

func (a *App) assetMeta(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.workspace.AssetMeta(r.Context(), who(r), r.PathValue("id"))
}

func (a *App) requestVariant(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.RequestVariantCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.RequestVariant(request.Context(), who(request), request.PathValue("id"), value)
}

func (a *App) uploadAsset(w http.ResponseWriter, request *http.Request) (any, error) {
	request.Body = http.MaxBytesReader(w, request.Body, multipartRequestBytes)
	parseMultipartFormErr := request.ParseMultipartForm(multipartMemoryBytes) //nolint:gosec // G120: MaxBytesReader bounds the complete multipart body above.
	if parseMultipartFormErr != nil {
		return nil, problem(http.StatusRequestEntityTooLarge, "upload_limit", "Choose an image up to 20 MB.")
	}
	if request.MultipartForm != nil {
		defer func() { _ = request.MultipartForm.RemoveAll() }()
	}
	file, _, err := request.FormFile("file")
	if err != nil {
		return nil, bad("Choose an image file.")
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		return nil, fault.Wrap("media.go", err)
	}
	if len(raw) > maxUploadBytes {
		return nil, problem(http.StatusRequestEntityTooLarge, "upload_limit", "Choose an image up to 20 MB.")
	}

	return a.workspace.UploadAsset(request.Context(), who(request), raw)
}

func (a *App) getAsset(writer http.ResponseWriter, r *http.Request) (any, error) {
	bucket, _ := strconv.Atoi(r.URL.Query().Get("bucket"))
	blob, err := a.workspace.GetAsset(r.Context(), who(r), r.PathValue("id"), bucket)
	if err != nil {
		return nil, err
	}
	writer.Header().Set("Content-Type", blob.MIME)
	writer.Header().Set("Content-Length", strconv.Itoa(len(blob.Data)))
	writer.Header().Set("X-Image-Width", strconv.Itoa(blob.Width))
	writer.Header().Set("X-Image-Height", strconv.Itoa(blob.Height))
	_, _ = writer.Write(blob.Data)

	return responseWritten{}, nil
}
