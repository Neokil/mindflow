package app

import (
	"net/http"

	"mindflow/internal/domain"
)

func (a *App) getDocument(w http.ResponseWriter, r *http.Request) (any, error) {

	return a.service.GetDocument(r.Context(), who(r), r.PathValue("id"))
}
func (a *App) saveDocument(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.SaveDocumentCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.SaveDocument(r.Context(), who(r), r.PathValue("id"), v)
}
func (a *App) recoveryCopy(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.RecoveryCopyCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.RecoveryCopy(r.Context(), who(r), v)
}
