package app

import (
	"mindflow/internal/workspace"
	"net/http"
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

func (a *App) listWorkspaces(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.workspace.ListWorkspaces(r.Context(), who(r))
}

func (a *App) createWorkspace(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.CreateWorkspaceCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.CreateWorkspace(request.Context(), who(request), value)
}

func (a *App) renameWorkspace(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.RenameWorkspaceCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.RenameWorkspace(request.Context(), who(request), request.PathValue("id"), value)
}

func (a *App) deleteWorkspace(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.DeleteWorkspaceCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.DeleteWorkspace(request.Context(), who(request), request.PathValue("id"), value)
}

func (a *App) duplicateWorkspace(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.workspace.DuplicateWorkspace(r.Context(), who(r), r.PathValue("id"))
}

func (a *App) listPages(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.workspace.ListPages(r.Context(), who(r), r.PathValue("id"))
}

func (a *App) createPage(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.CreatePageCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.CreatePage(request.Context(), who(request), request.PathValue("id"), value)
}

func (a *App) renamePage(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.RenamePageCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.RenamePage(request.Context(), who(request), request.PathValue("id"), value)
}

func (a *App) deletePage(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.DeletePageCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.DeletePage(request.Context(), who(request), request.PathValue("id"), value)
}

func (a *App) duplicatePage(_ http.ResponseWriter, r *http.Request) (any, error) {
	return a.workspace.DuplicatePage(r.Context(), who(r), r.PathValue("id"))
}

func (a *App) reorderPages(w http.ResponseWriter, request *http.Request) (any, error) {
	var value workspace.ReorderPagesCommand
	err := decode(w, request, &value)
	if err != nil {
		return nil, err
	}

	return a.workspace.ReorderPages(request.Context(), who(request), request.PathValue("id"), value)
}
