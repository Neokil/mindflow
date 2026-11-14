package mindflow_test

import (
	"mindflow/internal/workspace"
	"mindflow/internal/workspace/page"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/sharing"
	"testing"
)

func TestWorkspacePreviewTracksFirstSavedPageAndMembership(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, firstID := owner.workspace()
	list := decoded[[]workspace.Workspace](t, owner.ok("GET", "/api/v1/workspaces", nil))
	if len(list) != 1 || list[0].FirstPageID != firstID || list[0].PreviewRevision != 0 || list[0].PageCount != 1 {
		t.Fatal("initial first-page preview metadata", list)
	}
	second := decoded[workspace.Created](t, owner.ok("POST", "/api/v1/workspaces/"+workspaceID+"/pages", map[string]string{nameField: "Another page"}))
	lease := owner.acquire(second.ID, "preview-editor", "acquire")
	owner.ok("PUT", "/api/v1/pages/"+second.ID+"/document", canvas.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "preview-editor", MutationID: "preview-saved", Document: canvas.NewDocument()})
	list = decoded[[]workspace.Workspace](t, owner.ok("GET", "/api/v1/workspaces", nil))
	if list[0].FirstPageID != firstID || list[0].PreviewRevision != 0 || list[0].PageCount != 2 {
		t.Fatal("editing a later page changed the cover", list)
	}
	owner.ok("PUT", "/api/v1/workspaces/"+workspaceID+"/page-order", workspace.ReorderPagesCommand{IDs: []string{second.ID, firstID}, Revision: list[0].Revision})
	list = decoded[[]workspace.Workspace](t, owner.ok("GET", "/api/v1/workspaces", nil))
	if list[0].FirstPageID != second.ID || list[0].PreviewRevision != 1 {
		t.Fatal("cover did not follow tab order", list)
	}
	reader, readerID := sharedUser(owner, "preview-reader@example.com")
	hidden := decoded[[]workspace.Workspace](t, reader.ok("GET", "/api/v1/workspaces", nil))
	if len(hidden) != 0 {
		t.Fatal("private preview identity leaked")
	}
	owner.ok("PUT", "/api/v1/workspaces/"+workspaceID+"/members", sharing.Command{UserID: readerID, Role: readAccessRole})
	shared := decoded[[]workspace.Workspace](t, reader.ok("GET", "/api/v1/workspaces", nil))
	if len(shared) != 1 || shared[0].FirstPageID != second.ID {
		t.Fatal("reader cannot preview shared first page", shared)
	}
	owner.ok("DELETE", "/api/v1/pages/"+second.ID, page.DeletePageCommand{Revision: 1, Confirm: true})
	owner.ok("DELETE", "/api/v1/pages/"+firstID, page.DeletePageCommand{Revision: 1, Confirm: true})
	list = decoded[[]workspace.Workspace](t, owner.ok("GET", "/api/v1/workspaces", nil))
	if list[0].FirstPageID != "" || list[0].PreviewRevision != 0 || list[0].PageCount != 0 {
		t.Fatal("empty workspace retained stale preview metadata", list)
	}
}
