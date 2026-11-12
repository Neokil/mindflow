package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"mindflow/internal/application"
	"mindflow/internal/domain"
)

func (a *Repositories) ListWorkspaces(ctx context.Context, actor domain.Principal) (result []domain.Workspace, err error) {
	defer mapRepositoryError(&err)

	rows, err := a.DB.QueryContext(ctx, `SELECT w.id,w.name,w.revision,w.updated_at,(SELECT count(*) FROM pages p WHERE p.workspace_id=w.id) FROM workspaces w WHERE w.owner_id=? ORDER BY w.updated_at DESC,w.id`, actor.ID)
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
func (a *Repositories) CreateWorkspace(ctx context.Context, actor domain.Principal, v domain.CreateWorkspaceCommand) (result application.WorkspaceCreated, err error) {
	defer mapRepositoryError(&err)

	v.Name = strings.TrimSpace(v.Name)
	if err := validName(v.Name); err != nil {
		return application.WorkspaceCreated{}, err
	}
	id := randomID()
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return application.WorkspaceCreated{}, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES(?,?,?,?,?)", id, actor.ID, v.Name, now(), now()); err != nil {
		return application.WorkspaceCreated{}, err
	}
	raw, _ := json.Marshal(emptyDocument())
	pid, err := insertPage(tx, id, "Untitled", string(raw))
	if err != nil {
		return application.WorkspaceCreated{}, err
	}
	if err = tx.Commit(); err != nil {
		return application.WorkspaceCreated{}, err
	}
	a.publish(actor.ID, "workspace", id)
	return application.WorkspaceCreated{ID: id, PageID: pid}, nil

}
func (a *Repositories) RenameWorkspace(ctx context.Context, actor domain.Principal, id string, v domain.RenameWorkspaceCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	if err := validName(v.Name); err != nil {
		return application.Completed{}, err
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return application.Completed{}, err
	}
	defer tx.Rollback()
	var revision int
	if err = tx.QueryRow("SELECT revision FROM workspaces WHERE id=? AND owner_id=?", id, actor.ID).Scan(&revision); err != nil {
		return application.Completed{}, err
	}
	if revision != v.Revision {
		return application.Completed{}, domain.Fault("metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	if _, err = tx.Exec("UPDATE workspaces SET name=?,revision=revision+1,updated_at=? WHERE id=?", strings.TrimSpace(v.Name), now(), id); err != nil {
		return application.Completed{}, err
	}
	if err = tx.Commit(); err != nil {
		return application.Completed{}, err
	}
	a.publish(actor.ID, "workspace", id)
	return application.Completed{OK: true}, nil

}
func (a *Repositories) DeleteWorkspace(ctx context.Context, actor domain.Principal, id string, v domain.DeleteWorkspaceCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	if !v.Confirm {
		return application.Completed{}, bad("Confirm workspace deletion.")
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return application.Completed{}, err
	}
	defer tx.Rollback()
	var rev int
	if err = tx.QueryRow("SELECT revision FROM workspaces WHERE id=? AND owner_id=?", id, actor.ID).Scan(&rev); err != nil {
		return application.Completed{}, err
	}
	if rev != v.Revision {
		return application.Completed{}, domain.Fault("metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	if _, err = tx.Exec("DELETE FROM assets WHERE owner_id=? AND id IN(SELECT pa.asset_id FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE p.workspace_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa JOIN pages p ON p.id=pa.page_id WHERE pa.asset_id=assets.id AND p.workspace_id<>?)", actor.ID, id, id); err != nil {
		return application.Completed{}, err
	}
	if _, err = tx.Exec("DELETE FROM workspaces WHERE id=? AND owner_id=?", id, actor.ID); err != nil {
		return application.Completed{}, err
	}
	if err = tx.Commit(); err != nil {
		return application.Completed{}, err
	}
	a.publish(actor.ID, "workspace", id)
	return application.Completed{OK: true}, nil

}
func (a *Repositories) DuplicateWorkspace(ctx context.Context, actor domain.Principal, resourceID string) (result application.Created, err error) {
	defer mapRepositoryError(&err)

	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return application.Created{}, err
	}
	defer tx.Rollback()
	var name string
	source := resourceID
	if err = tx.QueryRow("SELECT name FROM workspaces WHERE id=? AND owner_id=?", source, actor.ID).Scan(&name); err != nil {
		return application.Created{}, err
	}
	id := randomID()
	name = copyName(name)
	if _, err = tx.Exec("INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES(?,?,?,?,?)", id, actor.ID, name, now(), now()); err != nil {
		return application.Created{}, err
	}
	rows, err := tx.Query("SELECT id,name FROM pages WHERE workspace_id=? ORDER BY position", source)
	if err != nil {
		return application.Created{}, err
	}
	type entry struct{ id, name string }
	items := []entry{}
	for rows.Next() {
		var e entry
		if err = rows.Scan(&e.id, &e.name); err != nil {
			rows.Close()
			return application.Created{}, err
		}
		items = append(items, e)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return application.Created{}, err
	}
	rows.Close()
	for _, e := range items {
		if _, err = copyPage(tx, e.id, id, e.name); err != nil {
			return application.Created{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return application.Created{}, err
	}
	a.publish(actor.ID, "workspace", id)
	return application.Created{ID: id}, nil

}
func (a *Repositories) ListPages(ctx context.Context, actor domain.Principal, resourceID string) (result []domain.Page, err error) {
	defer mapRepositoryError(&err)

	var id string
	if err := a.DB.QueryRowContext(ctx, "SELECT id FROM workspaces WHERE id=? AND owner_id=?", resourceID, actor.ID).Scan(&id); err != nil {
		return nil, err
	}
	rows, err := a.DB.QueryContext(ctx, "SELECT id,workspace_id,name,position,revision,metadata_revision FROM pages WHERE workspace_id=? ORDER BY position,id", id)
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
func (a *Repositories) CreatePage(ctx context.Context, actor domain.Principal, resourceID string, v domain.CreatePageCommand) (result application.Created, err error) {
	defer mapRepositoryError(&err)

	if err := validName(v.Name); err != nil {
		return application.Created{}, err
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return application.Created{}, err
	}
	defer tx.Rollback()
	var wid string
	if err = tx.QueryRow("SELECT id FROM workspaces WHERE id=? AND owner_id=?", resourceID, actor.ID).Scan(&wid); err != nil {
		return application.Created{}, err
	}
	raw, _ := json.Marshal(emptyDocument())
	id, err := insertPage(tx, wid, v.Name, string(raw))
	if err != nil {
		return application.Created{}, err
	}
	if err = tx.Commit(); err != nil {
		return application.Created{}, err
	}
	a.publish(actor.ID, "workspace", wid)
	return application.Created{ID: id}, nil

}
func (a *Repositories) RenamePage(ctx context.Context, actor domain.Principal, id string, v domain.RenamePageCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	if err := validName(v.Name); err != nil {
		return application.Completed{}, err
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return application.Completed{}, err
	}
	defer tx.Rollback()
	p, err := ownedPage(tx, id, actor.ID)
	if err != nil {
		return application.Completed{}, err
	}
	if p.MetadataRevision != v.Revision {
		return application.Completed{}, domain.Fault("metadata_conflict", "Page changed. Refresh and retry.")
	}
	if _, err = tx.Exec("UPDATE pages SET name=?,metadata_revision=metadata_revision+1 WHERE id=?", strings.TrimSpace(v.Name), p.ID); err != nil {
		return application.Completed{}, err
	}
	if err = tx.Commit(); err != nil {
		return application.Completed{}, err
	}
	a.publish(actor.ID, "workspace", p.WorkspaceID)
	return application.Completed{OK: true}, nil

}
func (a *Repositories) DeletePage(ctx context.Context, actor domain.Principal, id string, v domain.DeletePageCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	if !v.Confirm {
		return application.Completed{}, bad("Confirm page deletion.")
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return application.Completed{}, err
	}
	defer tx.Rollback()
	p, err := ownedPage(tx, id, actor.ID)
	if err != nil {
		return application.Completed{}, err
	}
	if p.MetadataRevision != v.Revision {
		return application.Completed{}, domain.Fault("metadata_conflict", "Page changed. Refresh and retry.")
	}
	if _, err = tx.Exec("DELETE FROM assets WHERE owner_id=? AND id IN(SELECT asset_id FROM page_assets WHERE page_id=?) AND NOT EXISTS(SELECT 1 FROM page_assets pa WHERE pa.asset_id=assets.id AND pa.page_id<>?)", actor.ID, p.ID, p.ID); err != nil {
		return application.Completed{}, err
	}
	if _, err = tx.Exec("DELETE FROM pages WHERE id=?", p.ID); err != nil {
		return application.Completed{}, err
	}
	if err = tx.Commit(); err != nil {
		return application.Completed{}, err
	}
	a.publish(actor.ID, "workspace", p.WorkspaceID)
	return application.Completed{OK: true}, nil

}
func (a *Repositories) DuplicatePage(ctx context.Context, actor domain.Principal, resourceID string) (result application.Created, err error) {
	defer mapRepositoryError(&err)

	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return application.Created{}, err
	}
	defer tx.Rollback()
	p, err := ownedPage(tx, resourceID, actor.ID)
	if err != nil {
		return application.Created{}, err
	}
	id, err := copyPage(tx, p.ID, p.WorkspaceID, copyName(p.Name))
	if err != nil {
		return application.Created{}, err
	}
	if err = tx.Commit(); err != nil {
		return application.Created{}, err
	}
	a.publish(actor.ID, "workspace", p.WorkspaceID)
	return application.Created{ID: id}, nil

}
func (a *Repositories) ReorderPages(ctx context.Context, actor domain.Principal, id string, v domain.ReorderPagesCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return application.Completed{}, err
	}
	defer tx.Rollback()
	var rev int
	wid := id
	if err = tx.QueryRow("SELECT revision FROM workspaces WHERE id=? AND owner_id=?", wid, actor.ID).Scan(&rev); err != nil {
		return application.Completed{}, err
	}
	if rev != v.Revision {
		return application.Completed{}, domain.Fault("metadata_conflict", "Workspace changed. Refresh and retry.")
	}
	var n int
	if err = tx.QueryRow("SELECT count(*) FROM pages WHERE workspace_id=?", wid).Scan(&n); err != nil {
		return application.Completed{}, err
	}
	if n != len(v.IDs) {
		return application.Completed{}, bad("Include every page exactly once.")
	}
	seen := map[string]bool{}
	for i, id := range v.IDs {
		if seen[id] {
			return application.Completed{}, bad("Duplicate page in order.")
		}
		seen[id] = true
		res, e := tx.Exec("UPDATE pages SET position=? WHERE id=? AND workspace_id=?", i, id, wid)
		if e != nil {
			return application.Completed{}, e
		}
		if count, _ := res.RowsAffected(); count != 1 {
			return application.Completed{}, notFound()
		}
	}
	if _, err = tx.Exec("UPDATE workspaces SET revision=revision+1 WHERE id=?", wid); err != nil {
		return application.Completed{}, err
	}
	if err = tx.Commit(); err != nil {
		return application.Completed{}, err
	}
	a.publish(actor.ID, "workspace", wid)
	return application.Completed{OK: true}, nil

}
func insertPage(tx *sql.Tx, wid, name, raw string) (string, error) {
	id := randomID()
	_, err := tx.Exec("INSERT INTO pages(id,workspace_id,name,position,document,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,? FROM pages WHERE workspace_id=?", id, wid, name, raw, now(), wid)
	return id, err
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
func copyName(name string) string {
	r := []rune(name)
	if len(r) > 150 {
		r = r[:150]
	}
	return string(r) + " copy"
}
func ownedPage(tx *sql.Tx, id, owner string) (Page, error) {
	var p Page
	err := tx.QueryRow("SELECT p.id,p.workspace_id,p.name,p.position,p.revision,p.metadata_revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?", id, owner).Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Position, &p.Revision, &p.MetadataRevision)
	return p, err
}
func cloneDocument(raw string) (string, error) {
	var d Document
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return "", err
	}
	groups := map[string]string{}
	for i := range d.Elements {
		d.Elements[i].ID = randomID()
		g := d.Elements[i].GroupID
		if g != "" {
			if groups[g] == "" {
				groups[g] = randomID()
			}
			d.Elements[i].GroupID = groups[g]
		}
	}
	data, err := json.Marshal(d)
	return string(data), err
}
