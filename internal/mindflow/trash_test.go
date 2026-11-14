package mindflow_test

import (
	"mindflow/internal/workspace"
	imageasset "mindflow/internal/workspace/image"
	"mindflow/internal/workspace/page"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/storage"
	"mindflow/internal/workspace/trash"
	"testing"
	"time"
)

func TestTrashPreservesAndRestoresPagesAndWorkspaces(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, pageID := owner.workspace()
	asset := owner.uploadImage()
	doc := canvas.NewDocument()
	doc.Elements = []canvas.Element{{ID: "trash-image", Type: imageElementKind, Width: 100, Height: 100, AssetID: asset.ID}}
	lease := owner.acquire(pageID, "trash-editor", "acquire")
	owner.ok("PUT", "/api/v1/pages/"+pageID+"/document", canvas.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "trash-editor", MutationID: "trash-source", Document: doc})
	reader, _ := sharedUser(owner, "trash-reader@example.com")
	owner.ok("PUT", "/api/v1/workspaces/"+workspaceID+"/members", map[string]string{emailField: "trash-reader@example.com", roleField: readAccessRole})
	owner.ok("DELETE", "/api/v1/pages/"+pageID, page.DeletePageCommand{Revision: 1, Confirm: true})
	if owner.request("GET", "/api/v1/pages/"+pageID+"/document", nil).Code != 404 || reader.request("GET", "/api/v1/assets/"+asset.ID, nil).Code != 404 {
		t.Fatal("trashed content still readable")
	}
	items := decoded[[]trash.Item](t, owner.ok("GET", "/api/v1/trash", nil))
	if len(items) != 1 || items[0].ID != pageID {
		t.Fatal("page missing from trash", items)
	}
	if reader.request("POST", "/api/v1/trash/pages/"+pageID+"/restore", nil).Code != 403 {
		t.Fatal("reader restored a page")
	}
	list := decoded[[]workspace.Workspace](t, owner.ok("GET", "/api/v1/workspaces", nil))
	owner.ok("DELETE", "/api/v1/workspaces/"+workspaceID, workspace.DeleteWorkspaceCommand{Revision: list[0].Revision, Confirm: true})
	if len(decoded[[]trash.Item](t, owner.ok("GET", "/api/v1/trash", nil))) != 1 {
		t.Fatal("child trash duplicated under deleted workspace")
	}
	if len(decoded[[]workspace.Workspace](t, reader.ok("GET", "/api/v1/workspaces", nil))) != 0 {
		t.Fatal("member sees trashed workspace")
	}
	owner.ok("POST", "/api/v1/trash/workspaces/"+workspaceID+"/restore", nil)
	if owner.request("GET", "/api/v1/pages/"+pageID+"/document", nil).Code != 404 {
		t.Fatal("workspace restore revived independently trashed page")
	}
	owner.ok("POST", "/api/v1/trash/pages/"+pageID+"/restore", nil)
	owner.ok("GET", "/api/v1/assets/"+asset.ID, nil)
	owner.ok("PUT", "/api/v1/workspaces/"+workspaceID+"/members", map[string]string{emailField: "trash-reader@example.com", roleField: fullAccessRole})
	owner.ok("DELETE", "/api/v1/pages/"+pageID, page.DeletePageCommand{Revision: 3, Confirm: true})
	reader.ok("POST", "/api/v1/trash/pages/"+pageID+"/restore", nil)
}

func TestTrashExpiryPermanentlyPurgesAndAccountDeletionBypassesRetention(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	member, userID := sharedUser(owner, "trash-expiry@example.com")
	workspaceID, pageID := member.workspace()
	member.ok("DELETE", "/api/v1/pages/"+pageID, page.DeletePageCommand{Revision: 1, Confirm: true})
	expired := time.Now().Add(-31 * 24 * time.Hour).Unix()
	_, err := owner.db.ExecContext(t.Context(), "UPDATE pages SET deleted_at=? WHERE id=?", expired, pageID)
	if err != nil {
		t.Fatal(err)
	}
	if member.request("POST", "/api/v1/trash/pages/"+pageID+"/restore", nil).Code != 404 {
		t.Fatal("expired page restored")
	}
	repository := storage.NewRepository(owner.db)
	service := trash.NewService(repository, imageasset.NewService(repository, func(string, string, string) {}), func(string, string, string) {}, time.Now)
	err = service.Cleanup(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var count int
	err = owner.db.QueryRowContext(t.Context(), "SELECT count(*) FROM pages WHERE id=?", pageID).Scan(&count)
	if err != nil || count != 0 {
		t.Fatal("expired page retained", err)
	}
	list := decoded[[]workspace.Workspace](t, member.ok("GET", "/api/v1/workspaces", nil))
	member.ok("DELETE", "/api/v1/workspaces/"+workspaceID, workspace.DeleteWorkspaceCommand{Revision: list[0].Revision, Confirm: true})
	owner.ok("DELETE", "/api/v1/admin/users/"+userID, map[string]bool{confirmField: true})
	err = owner.db.QueryRowContext(t.Context(), "SELECT count(*) FROM workspaces WHERE id=?", workspaceID).Scan(&count)
	if err != nil || count != 0 {
		t.Fatal("account deletion retained trashed data", err)
	}
}
