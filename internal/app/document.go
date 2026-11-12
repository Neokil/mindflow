package app

import (
	"mindflow/internal/workspace"
	"net/http"
)

func (a *App) getDocument(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.workspace.GetDocument(r.Context(), who(r), r.PathValue("id"))
}

func (a *App) saveDocument(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.SaveDocumentCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.SaveDocument(request.Context(), who(request), request.PathValue("id"), value)
}

func (a *App) recoveryCopy(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.RecoveryCopyCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.RecoveryCopy(request.Context(), who(request), value)
}
