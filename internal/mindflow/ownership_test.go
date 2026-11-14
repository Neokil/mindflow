package mindflow_test

import (
	"mindflow/internal/workspace"
	"mindflow/internal/workspace/ownership"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/template"
	"testing"
)

func TestOwnershipTransferRetainsMediaAfterFormerOwnerDeletion(t *testing.T) {
	t.Parallel()
	admin := fixture(t)
	admin.setup()
	previous, previousID := sharedUser(admin, "previous-owner@example.com")
	recipient, recipientID := sharedUser(admin, "next-owner@example.com")
	workspaceID, pageID := previous.workspace()
	asset := previous.uploadImage()
	doc := canvas.NewDocument()
	doc.Elements = []canvas.Element{{ID: "owned-image", Type: imageElementKind, Width: 200, Height: 100, AssetID: asset.ID}}
	lease := previous.acquire(pageID, "ownership-editor", "acquire")
	previous.ok("PUT", "/api/v1/pages/"+pageID+"/document", canvas.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "ownership-editor", MutationID: "ownership-save", Document: doc})
	previous.ok("POST", "/api/v1/workspaces/"+workspaceID+"/templates", template.CreateCommand{Name: "Banner", Elements: doc.Elements})
	previous.ok("POST", "/api/v1/workspaces/"+workspaceID+"/duplicate", nil)
	exclusive := previous.uploadImage()
	doc.Elements = append(doc.Elements, canvas.Element{ID: "exclusive-image", Type: imageElementKind, Width: 100, Height: 100, AssetID: exclusive.ID})
	previous.ok("PUT", "/api/v1/pages/"+pageID+"/document", canvas.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "ownership-editor", MutationID: "ownership-exclusive", BaseRevision: 1, Document: doc})

	summaries := decoded[[]workspace.Workspace](t, previous.ok("GET", "/api/v1/workspaces", nil))
	var revision int
	for _, w := range summaries {
		if w.ID == workspaceID {
			revision = w.Revision
		}
	}
	command := ownership.Command{UserID: recipientID, Revision: revision, Confirm: true}
	path := "/api/v1/workspaces/" + workspaceID + "/ownership"
	if recipient.request("POST", path, ownership.Command{UserID: previousID, Revision: revision, Confirm: true}).Code != 404 || admin.request("POST", path, command).Code != 404 {
		t.Fatal("unrelated account transferred ownership")
	}
	previous.ok("POST", path, command)
	current := decoded[[]workspace.Workspace](t, recipient.ok("GET", "/api/v1/workspaces", nil))
	if len(current) != 1 || current[0].Role != "owner" {
		t.Fatal("recipient did not become owner", current)
	}
	former := decoded[[]workspace.Workspace](t, previous.ok("GET", "/api/v1/workspaces", nil))
	for _, w := range former {
		if w.ID == workspaceID && w.Role != "full" {
			t.Fatal("former owner did not retain full access")
		}
	}
	if previous.request("POST", path, command).Code != 403 {
		t.Fatal("former owner still has ownership privileges")
	}
	snapshot := decoded[struct {
		Document canvas.Document `json:"document"`
	}](t, recipient.ok("GET", "/api/v1/pages/"+pageID+"/document", nil))
	if snapshot.Document.Elements[0].AssetID == asset.ID {
		t.Fatal("shared media was not made independent")
	}
	if snapshot.Document.Elements[1].AssetID != exclusive.ID {
		t.Fatal("exclusive media was unnecessarily cloned")
	}
	admin.ok("DELETE", "/api/v1/admin/users/"+previousID, map[string]bool{confirmField: true})
	recipient.ok("GET", "/api/v1/assets/"+snapshot.Document.Elements[0].AssetID, nil)
	recipient.ok("GET", "/api/v1/assets/"+exclusive.ID, nil)
	library := decoded[[]template.Summary](t, recipient.ok("GET", "/api/v1/workspaces/"+workspaceID+"/templates", nil))
	if len(library) != 1 {
		t.Fatal("template library missing")
	}
	saved := decoded[template.Snapshot](t, recipient.ok("GET", "/api/v1/workspaces/"+workspaceID+"/templates/"+library[0].ID, nil))
	if saved.Elements[0].AssetID != snapshot.Document.Elements[0].AssetID {
		t.Fatal("template did not follow transferred media")
	}
}
