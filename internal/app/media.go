package app

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"mindflow/internal/domain"
)

func (a *App) assetMeta(w http.ResponseWriter, r *http.Request) (any, error) {

	return a.service.AssetMeta(r.Context(), who(r), r.PathValue("id"))
}
func (a *App) requestVariant(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.RequestVariantCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.RequestVariant(r.Context(), who(r), r.PathValue("id"), v)
}
func (a *App) uploadAsset(w http.ResponseWriter, r *http.Request) (any, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 21<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		return nil, problem(413, "upload_limit", "Choose an image up to 20 MB.")
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, bad("Choose an image file.")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, (20<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > 20<<20 {
		return nil, problem(413, "upload_limit", "Choose an image up to 20 MB.")
	}
	return a.service.UploadAsset(r.Context(), who(r), raw)
}
func (a *App) getAsset(w http.ResponseWriter, r *http.Request) (any, error) {
	bucket, _ := strconv.Atoi(r.URL.Query().Get("bucket"))
	blob, err := a.service.GetAsset(r.Context(), who(r), r.PathValue("id"), bucket)
	if err != nil {
		return nil, err
	}
	w.Header().Set("Content-Type", blob.MIME)
	w.Header().Set("Content-Length", fmt.Sprint(len(blob.Data)))
	w.Header().Set("X-Image-Width", fmt.Sprint(blob.Width))
	w.Header().Set("X-Image-Height", fmt.Sprint(blob.Height))
	_, _ = w.Write(blob.Data)
	return nil, nil
}
