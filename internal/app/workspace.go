package app

import (
	"net/http"

	"mindflow/internal/domain"
)

func (a *App) workspaceRoutes() {
	a.endpoint("GET /api/v1/workspaces", true, false, a.listWorkspaces)
	a.endpoint("POST /api/v1/workspaces", true, false, a.createWorkspace)
	a.endpoint("PATCH /api/v1/workspaces/{id}", true, false, a.renameWorkspace)
	a.endpoint("DELETE /api/v1/workspaces/{id}", true, false, a.deleteWorkspace)
	a.endpoint("POST /api/v1/workspaces/{id}/duplicate", true, false, a.duplicateWorkspace)
	a.endpoint("GET /api/v1/workspaces/{id}/pages", true, false, a.listPages)
	a.endpoint("POST /api/v1/workspaces/{id}/pages", true, false, a.createPage)
	a.endpoint("PUT /api/v1/workspaces/{id}/page-order", true, false, a.reorderPages)
	a.endpoint("PATCH /api/v1/pages/{id}", true, false, a.renamePage)
	a.endpoint("DELETE /api/v1/pages/{id}", true, false, a.deletePage)
	a.endpoint("POST /api/v1/pages/{id}/duplicate", true, false, a.duplicatePage)
	a.endpoint("GET /api/v1/pages/{id}/document", true, false, a.getDocument)
	a.endpoint("PUT /api/v1/pages/{id}/document", true, false, a.saveDocument)
	a.endpoint("POST /api/v1/pages/recovery-copy", true, false, a.recoveryCopy)
	a.endpoint("POST /api/v1/pages/{id}/lease", true, false, a.lease)
	a.endpoint("GET /api/v1/pages/{id}/lease", true, false, a.leaseStatus)
	a.endpoint("GET /api/v1/events", true, false, a.events)
	a.endpoint("POST /api/v1/assets", true, false, a.uploadAsset)
	a.endpoint("GET /api/v1/assets/{id}", true, false, a.getAsset)
	a.endpoint("GET /api/v1/assets/{id}/meta", true, false, a.assetMeta)
	a.endpoint("POST /api/v1/assets/{id}/variant", true, false, a.requestVariant)
}
func (a *App) listWorkspaces(w http.ResponseWriter, r *http.Request) (any, error) {

	return a.service.ListWorkspaces(r.Context(), who(r))
}
func (a *App) createWorkspace(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.CreateWorkspaceCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.CreateWorkspace(r.Context(), who(r), v)
}
func (a *App) renameWorkspace(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.RenameWorkspaceCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.RenameWorkspace(r.Context(), who(r), r.PathValue("id"), v)
}
func (a *App) deleteWorkspace(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.DeleteWorkspaceCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.DeleteWorkspace(r.Context(), who(r), r.PathValue("id"), v)
}
func (a *App) duplicateWorkspace(w http.ResponseWriter, r *http.Request) (any, error) {

	return a.service.DuplicateWorkspace(r.Context(), who(r), r.PathValue("id"))
}
func (a *App) listPages(w http.ResponseWriter, r *http.Request) (any, error) {

	return a.service.ListPages(r.Context(), who(r), r.PathValue("id"))
}
func (a *App) createPage(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.CreatePageCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.CreatePage(r.Context(), who(r), r.PathValue("id"), v)
}
func (a *App) renamePage(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.RenamePageCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.RenamePage(r.Context(), who(r), r.PathValue("id"), v)
}
func (a *App) deletePage(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.DeletePageCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.DeletePage(r.Context(), who(r), r.PathValue("id"), v)
}
func (a *App) duplicatePage(w http.ResponseWriter, r *http.Request) (any, error) {

	return a.service.DuplicatePage(r.Context(), who(r), r.PathValue("id"))
}
func (a *App) reorderPages(w http.ResponseWriter, r *http.Request) (any, error) {
	var v domain.ReorderPagesCommand
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	return a.service.ReorderPages(r.Context(), who(r), r.PathValue("id"), v)
}
