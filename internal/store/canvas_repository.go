package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"mindflow/internal/application"
	"mindflow/internal/domain"
)

func (a *Repositories) GetDocument(ctx context.Context, actor domain.Principal, id string) (result application.PageSnapshot, err error) {
	defer mapRepositoryError(&err)

	var raw string
	var revision, metadata int
	var name, workspace string
	err = a.db.QueryRowContext(ctx, `SELECT p.document,p.revision,p.metadata_revision,p.name,p.workspace_id FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?`, id, actor.ID).Scan(&raw, &revision, &metadata, &name, &workspace)
	if err != nil {
		return application.PageSnapshot{}, err
	}
	return application.PageSnapshot{Document: json.RawMessage(raw), Revision: revision, MetadataRevision: metadata, Name: name, WorkspaceID: workspace}, nil

}
func (a *Repositories) SaveDocument(ctx context.Context, actor domain.Principal, id string, v domain.SaveDocumentCommand) (result application.SaveReceipt, err error) {
	defer mapRepositoryError(&err)

	if len(v.MutationID) < 8 || len(v.MutationID) > 128 || len(v.ClientID) < 8 || len(v.ClientID) > 128 {
		return application.SaveReceipt{}, bad("Invalid save identity.")
	}
	if err := validateDocument(v.Document); err != nil {
		return application.SaveReceipt{}, err
	}
	raw, _ := json.Marshal(v.Document)
	if len(raw) > 16<<20 {
		return application.SaveReceipt{}, domain.Fault("document_limit", "The page exceeds the 16 MB document limit.")
	}
	requestRaw, _ := json.Marshal(v)
	hash := digest(string(requestRaw))
	p := actor
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return application.SaveReceipt{}, err
	}
	defer tx.Rollback()
	var revision int
	if err = tx.QueryRow("SELECT p.revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?", id, p.ID).Scan(&revision); err != nil {
		return application.SaveReceipt{}, err
	}
	var receiptHash string
	var receiptRev int
	err = tx.QueryRow("SELECT digest,revision FROM save_receipts WHERE page_id=? AND client_id=? AND mutation_id=? AND expires_at>?", id, v.ClientID, v.MutationID, now()).Scan(&receiptHash, &receiptRev)
	if err == nil {
		if hash != receiptHash {
			return application.SaveReceipt{}, bad("Mutation ID was reused with different content.")
		}
		return application.SaveReceipt{Revision: receiptRev, MutationID: v.MutationID}, nil
	}
	if err != sql.ErrNoRows {
		return application.SaveReceipt{}, err
	}
	var session, client sql.NullString
	var epoch, expires int64
	err = tx.QueryRow("SELECT session_id,client_id,epoch,expires_at FROM page_leases WHERE page_id=?", id).Scan(&session, &client, &epoch, &expires)
	if err != nil {
		if err == sql.ErrNoRows {
			return application.SaveReceipt{}, domain.Fault("lease_lost", "Editor control expired.")
		}
		return application.SaveReceipt{}, err
	}
	state := domain.LeaseState{SessionID: session.String, ClientID: client.String, Epoch: epoch, ExpiresAt: expires}
	if err = state.CheckSave(p.SessionID, v.ClientID, v.Epoch, v.BaseRevision, revision, now()); err != nil {
		return application.SaveReceipt{}, err
	}
	if err = validateAssets(tx, p.ID, id, v.Document); err != nil {
		return application.SaveReceipt{}, err
	}
	revision++
	if _, err = tx.Exec("UPDATE pages SET document=?,revision=?,updated_at=? WHERE id=?", string(raw), revision, now(), id); err != nil {
		return application.SaveReceipt{}, err
	}
	if _, err = tx.Exec("UPDATE workspaces SET updated_at=? WHERE id=(SELECT workspace_id FROM pages WHERE id=?)", now(), id); err != nil {
		return application.SaveReceipt{}, err
	}
	if _, err = tx.Exec("INSERT INTO save_receipts(page_id,client_id,mutation_id,digest,revision,expires_at) VALUES(?,?,?,?,?,?)", id, v.ClientID, v.MutationID, hash, revision, now()+86400); err != nil {
		return application.SaveReceipt{}, err
	}
	if err = tx.Commit(); err != nil {
		return application.SaveReceipt{}, err
	}
	a.publish(p.ID, "page", id)
	a.enqueueImageVariants(ctx, p.ID, v.Document)
	return application.SaveReceipt{Revision: revision, MutationID: v.MutationID}, nil

}
func (a *Repositories) RecoveryCopy(ctx context.Context, actor domain.Principal, v domain.RecoveryCopyCommand) (result application.Created, err error) {
	defer mapRepositoryError(&err)

	if err := validateDocument(v.Document); err != nil {
		return application.Created{}, err
	}
	if strings.TrimSpace(v.Name) == "" {
		v.Name = "Recovered page"
	}
	if err := validName(v.Name); err != nil {
		return application.Created{}, err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return application.Created{}, err
	}
	defer tx.Rollback()
	var wid string
	if err = tx.QueryRow("SELECT id FROM workspaces WHERE id=? AND owner_id=?", v.WorkspaceID, actor.ID).Scan(&wid); err != nil {
		return application.Created{}, err
	}
	raw, _ := json.Marshal(v.Document)
	cloned, err := cloneDocument(string(raw))
	if err != nil {
		return application.Created{}, err
	}
	id := randomID()
	if _, err = tx.Exec("INSERT INTO pages(id,workspace_id,name,position,document,revision,updated_at) SELECT ?,?,?,COALESCE(MAX(position),-1)+1,?,1,? FROM pages WHERE workspace_id=?", id, wid, v.Name, cloned, now(), wid); err != nil {
		return application.Created{}, err
	}
	if err = validateAssets(tx, actor.ID, id, v.Document); err != nil {
		return application.Created{}, err
	}
	if err = tx.Commit(); err != nil {
		return application.Created{}, err
	}
	a.publish(actor.ID, "workspace", wid)
	return application.Created{ID: id}, nil

}
func (a *Repositories) Lease(ctx context.Context, actor domain.Principal, id string, v domain.LeaseCommand) (result domain.LeaseResult, err error) {
	defer mapRepositoryError(&err)

	if len(v.ClientID) < 8 || len(v.ClientID) > 128 {
		return domain.LeaseResult{}, bad("Invalid editor identity.")
	}
	if v.Action != "acquire" && v.Action != "takeover" && v.Action != "renew" && v.Action != "release" {
		return domain.LeaseResult{}, bad("Invalid editor action.")
	}
	p := actor
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.LeaseResult{}, err
	}
	defer tx.Rollback()
	page, err := ownedPage(tx, id, p.ID)
	if err != nil {
		return domain.LeaseResult{}, err
	}
	if _, err = tx.Exec("INSERT OR IGNORE INTO page_leases(page_id) VALUES(?)", page.ID); err != nil {
		return domain.LeaseResult{}, err
	}
	var session sql.NullString
	var client string
	var epoch, expires int64
	if err = tx.QueryRow("SELECT session_id,client_id,epoch,expires_at FROM page_leases WHERE page_id=?", page.ID).Scan(&session, &client, &epoch, &expires); err != nil {
		return domain.LeaseResult{}, err
	}
	state, changed, err := (domain.LeaseState{SessionID: session.String, ClientID: client, Epoch: epoch, ExpiresAt: expires}).Advance(p.SessionID, v, now())
	if err != nil {
		return domain.LeaseResult{}, err
	}
	epoch, expires, client = state.Epoch, state.ExpiresAt, state.ClientID
	session = sql.NullString{String: state.SessionID, Valid: state.SessionID != ""}
	if changed {
		if _, err = tx.Exec("UPDATE page_leases SET session_id=?,client_id=?,epoch=?,expires_at=? WHERE page_id=?", session, client, epoch, expires, page.ID); err != nil {
			return domain.LeaseResult{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return domain.LeaseResult{}, err
	}
	if changed && v.Action != "renew" {
		a.publish(p.ID, "lease", page.ID)
	}
	return leaseResult{Editable: session.String == p.SessionID && client == v.ClientID && expires > now(), Epoch: epoch, ExpiresAt: expires, Revision: page.Revision}, nil

}
func (a *Repositories) LeaseStatus(ctx context.Context, actor domain.Principal, id string, clientID string) (result domain.LeaseResult, err error) {
	defer mapRepositoryError(&err)

	var rev int
	if err := a.db.QueryRowContext(ctx, "SELECT p.revision FROM pages p JOIN workspaces w ON w.id=p.workspace_id WHERE p.id=? AND w.owner_id=?", id, actor.ID).Scan(&rev); err != nil {
		return domain.LeaseResult{}, err
	}
	var session sql.NullString
	var client string
	var epoch, expires int64
	err = a.db.QueryRowContext(ctx, "SELECT session_id,client_id,epoch,expires_at FROM page_leases WHERE page_id=?", id).Scan(&session, &client, &epoch, &expires)
	if err != nil && err != sql.ErrNoRows {
		return domain.LeaseResult{}, err
	}
	return leaseResult{Editable: session.String == actor.SessionID && client == clientID && expires > now(), Epoch: epoch, ExpiresAt: expires, Revision: rev}, nil

}
func validateAssets(tx *sql.Tx, owner, pageID string, d Document) error {
	seen := map[string]bool{}
	for _, e := range d.Elements {
		if e.Type != "image" || seen[e.AssetID] {
			continue
		}
		seen[e.AssetID] = true
		var n int
		if err := tx.QueryRow("SELECT count(*) FROM assets WHERE id=? AND owner_id=?", e.AssetID, owner).Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			return domain.Fault("asset_missing", "An image is unavailable. Reattach it before saving.")
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO page_assets(page_id,asset_id) VALUES(?,?)", pageID, e.AssetID); err != nil {
			return err
		}
	}
	return nil
}
