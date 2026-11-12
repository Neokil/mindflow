package app

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

type Workspace struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Revision  int    `json:"revision"`
	UpdatedAt int64  `json:"updatedAt"`
	PageCount int    `json:"pageCount"`
}
type Page struct {
	ID               string `json:"id"`
	WorkspaceID      string `json:"workspaceId"`
	Name             string `json:"name"`
	Position         int    `json:"position"`
	Revision         int    `json:"revision"`
	MetadataRevision int    `json:"metadataRevision"`
}

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
	rows, err := a.DB.Query(`SELECT w.id,w.name,w.revision,w.updated_at,(SELECT count(*) FROM pages p WHERE p.workspace_id=w.id) FROM workspaces w WHERE w.owner_id=? ORDER BY w.updated_at DESC,w.id`, who(r).ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Workspace{}
	for rows.Next() {
		var v Workspace
		if err = rows.Scan(&v.ID, &v.Name, &v.Revision, &v.UpdatedAt, &v.PageCount); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func insertPage(tx *sql.Tx, wid, name, raw string) (string, error) {
	id := randomID()
	_, err := tx.Exec("INSERT INTO pages(id,workspace_id,name,position,document,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,? FROM pages WHERE workspace_id=?", id, wid, name, raw, now(), wid)
	return id, err
}
func (a *App) createWorkspace(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		Name string `json:"name"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	v.Name = strings.TrimSpace(v.Name)
	if err := validName(v.Name); err != nil {
		return nil, err
	}
	id := randomID()
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES(?,?,?,?,?)", id, who(r).ID, v.Name, now(), now()); err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(emptyDocument())
	pid, err := insertPage(tx, id, "Untitled", string(raw))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(who(r).ID, "workspace", id)
	return map[string]string{"id": id, "pageId": pid}, nil
}
func (a *App) renameWorkspace(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		Name     string `json:"name"`
		Revision int    `json:"revision"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	if err := validName(v.Name); err != nil {
		return nil, err
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var revision int
	if err = tx.QueryRow("SELECT revision FROM workspaces WHERE id=? AND owner_id=?", r.PathValue("id"), who(r).ID).Scan(&revision); err != nil {
		return nil, err
	}
	if revision != v.Revision {
		return nil, problem(409, "metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	if _, err = tx.Exec("UPDATE workspaces SET name=?,revision=revision+1,updated_at=? WHERE id=?", strings.TrimSpace(v.Name), now(), r.PathValue("id")); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(who(r).ID, "workspace", r.PathValue("id"))
	return map[string]bool{"ok": true}, nil
}
func (a *App) deleteWorkspace(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		Revision int  `json:"revision"`
		Confirm  bool `json:"confirm"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	if !v.Confirm {
		return nil, bad("Confirm workspace deletion.")
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var rev int
	if err = tx.QueryRow("SELECT revision FROM workspaces WHERE id=? AND owner_id=?", r.PathValue("id"), who(r).ID).Scan(&rev); err != nil {
		return nil, err
	}
	if rev != v.Revision {
		return nil, problem(409, "metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	if _, err = tx.Exec("DELETE FROM assets WHERE owner_id=? AND id IN(SELECT pa.asset_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE p.workspace_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE pa.asset_id=assets.id AND p.workspace_id<>?)", who(r).ID, r.PathValue("id"), r.PathValue("id")); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("DELETE FROM workspaces WHERE id=? AND owner_id=?", r.PathValue("id"), who(r).ID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(who(r).ID, "workspace", r.PathValue("id"))
	return map[string]bool{"ok": true}, nil
}
func copyPage(tx *sql.Tx, source, wid, name string) (string, error) {
	var raw string
	if err := tx.QueryRow("SELECT document FROM pages WHERE id=?", source).Scan(&raw); err != nil {
		return "", err
	}
	cloned, err := cloneDocument(raw)
	if err != nil {
		return "", err
	}
	id, err := insertPage(tx, wid, name, cloned)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec("INSERT INTO page_assets(page_id,asset_id) SELECT ?,asset_id FROM page_assets WHERE page_id=?", id, source)
	return id, err
}
func (a *App) duplicateWorkspace(w http.ResponseWriter, r *http.Request) (any, error) {
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var name string
	source := r.PathValue("id")
	if err = tx.QueryRow("SELECT name FROM workspaces WHERE id=? AND owner_id=?", source, who(r).ID).Scan(&name); err != nil {
		return nil, err
	}
	id := randomID()
	name = copyName(name)
	if _, err = tx.Exec("INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES(?,?,?,?,?)", id, who(r).ID, name, now(), now()); err != nil {
		return nil, err
	}
	rows, err := tx.Query("SELECT id,name FROM pages WHERE workspace_id=? ORDER BY position", source)
	if err != nil {
		return nil, err
	}
	type entry struct{ id, name string }
	items := []entry{}
	for rows.Next() {
		var e entry
		if err = rows.Scan(&e.id, &e.name); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, e)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for _, e := range items {
		if _, err = copyPage(tx, e.id, id, e.name); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(who(r).ID, "workspace", id)
	return map[string]string{"id": id}, nil
}
func copyName(name string) string {
	r := []rune(name)
	if len(r) > 150 {
		r = r[:150]
	}
	return string(r) + " copy"
}
func (a *App) listPages(w http.ResponseWriter, r *http.Request) (any, error) {
	var id string
	if err := a.DB.QueryRow("SELECT id FROM workspaces WHERE id=? AND owner_id=?", r.PathValue("id"), who(r).ID).Scan(&id); err != nil {
		return nil, err
	}
	rows, err := a.DB.Query("SELECT id,workspace_id,name,position,revision,metadata_revision FROM pages WHERE workspace_id=? ORDER BY position,id", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pages := []Page{}
	for rows.Next() {
		var p Page
		if err = rows.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Position, &p.Revision, &p.MetadataRevision); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}
func (a *App) createPage(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		Name string `json:"name"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	if err := validName(v.Name); err != nil {
		return nil, err
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var wid string
	if err = tx.QueryRow("SELECT id FROM workspaces WHERE id=? AND owner_id=?", r.PathValue("id"), who(r).ID).Scan(&wid); err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(emptyDocument())
	id, err := insertPage(tx, wid, v.Name, string(raw))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(who(r).ID, "workspace", wid)
	return map[string]string{"id": id}, nil
}
func ownedPage(tx *sql.Tx, id, owner string) (Page, error) {
	var p Page
	err := tx.QueryRow("SELECT p.id,p.workspace_id,p.name,p.position,p.revision,p.metadata_revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?", id, owner).Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Position, &p.Revision, &p.MetadataRevision)
	return p, err
}
func (a *App) renamePage(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		Name     string `json:"name"`
		Revision int    `json:"revision"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	if err := validName(v.Name); err != nil {
		return nil, err
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	p, err := ownedPage(tx, r.PathValue("id"), who(r).ID)
	if err != nil {
		return nil, err
	}
	if p.MetadataRevision != v.Revision {
		return nil, problem(409, "metadata_conflict", "Page changed. Refresh and retry.")
	}
	if _, err = tx.Exec("UPDATE pages SET name=?,metadata_revision=metadata_revision+1 WHERE id=?", strings.TrimSpace(v.Name), p.ID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(who(r).ID, "workspace", p.WorkspaceID)
	return map[string]bool{"ok": true}, nil
}
func (a *App) deletePage(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		Revision int  `json:"revision"`
		Confirm  bool `json:"confirm"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	if !v.Confirm {
		return nil, bad("Confirm page deletion.")
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	p, err := ownedPage(tx, r.PathValue("id"), who(r).ID)
	if err != nil {
		return nil, err
	}
	if p.MetadataRevision != v.Revision {
		return nil, problem(409, "metadata_conflict", "Page changed. Refresh and retry.")
	}
	if _, err = tx.Exec("DELETE FROM assets WHERE owner_id=? AND id IN(SELECT asset_id FROM page_assets WHERE page_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa WHERE pa.asset_id=assets.id AND pa.page_id<>?)", who(r).ID, p.ID, p.ID); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("DELETE FROM pages WHERE id=?", p.ID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(who(r).ID, "workspace", p.WorkspaceID)
	return map[string]bool{"ok": true}, nil
}
func (a *App) duplicatePage(w http.ResponseWriter, r *http.Request) (any, error) {
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	p, err := ownedPage(tx, r.PathValue("id"), who(r).ID)
	if err != nil {
		return nil, err
	}
	id, err := copyPage(tx, p.ID, p.WorkspaceID, copyName(p.Name))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(who(r).ID, "workspace", p.WorkspaceID)
	return map[string]string{"id": id}, nil
}
func (a *App) reorderPages(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		IDs      []string `json:"ids"`
		Revision int      `json:"revision"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var rev int
	wid := r.PathValue("id")
	if err = tx.QueryRow("SELECT revision FROM workspaces WHERE id=? AND owner_id=?", wid, who(r).ID).Scan(&rev); err != nil {
		return nil, err
	}
	if rev != v.Revision {
		return nil, problem(409, "metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	var n int
	if err = tx.QueryRow("SELECT count(*) FROM pages WHERE workspace_id=?", wid).Scan(&n); err != nil {
		return nil, err
	}
	if n != len(v.IDs) {
		return nil, bad("Include every page exactly once.")
	}
	seen := map[string]bool{}
	for i, id := range v.IDs {
		if seen[id] {
			return nil, bad("Duplicate page in order.")
		}
		seen[id] = true
		res, e := tx.Exec("UPDATE pages SET position=? WHERE id=? AND workspace_id=?", i, id, wid)
		if e != nil {
			return nil, e
		}
		if count, _ := res.RowsAffected(); count != 1 {
			return nil, notFound()
		}
	}
	if _, err = tx.Exec("UPDATE workspaces SET revision=revision+1 WHERE id=?", wid); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	a.publish(who(r).ID, "workspace", wid)
	return map[string]bool{"ok": true}, nil
}
