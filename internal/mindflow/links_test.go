package mindflow_test

import (
	"mindflow/internal/workspace"
	"mindflow/internal/workspace/page"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/transfer"
	"strconv"
	"testing"
)

func TestElementLinksStayWithinWorkspaceAndFollowCopies(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, firstID := owner.workspace()
	second := decoded[page.Created](t, owner.ok("POST", "/api/v1/workspaces/"+workspaceID+"/pages", map[string]string{"name": "Target"}))
	target := canvas.NewDocument()
	target.Elements = []canvas.Element{{ID: linkTargetID, Type: "text", Text: "Destination", FontSize: 24, Align: "center", Color: linkTestColor, Width: 200, Height: 100}}
	saveLinkedDocument(owner, second.ID, "target-editor", target, 0)
	source := canvas.NewDocument()
	source.Elements = []canvas.Element{{ID: "source", Type: transferShapeType, Shape: "circle", Fill: linkTestColor, Border: linkTestColor, Width: 100, Height: 100, Link: &canvas.ElementLink{PageID: second.ID, ElementID: linkTargetID}}}
	saveLinkedDocument(owner, firstID, "source-editor", source, 0)
	// An unchanged dangling link is retained through edits and becomes usable on restore.
	owner.ok("DELETE", "/api/v1/pages/"+second.ID, page.DeletePageCommand{Revision: 1, Confirm: true})
	source.Elements[0].X = 50
	saveLinkedDocument(owner, firstID, "source-editor", source, 1)
	owner.ok("POST", "/api/v1/trash/pages/"+second.ID+"/restore", nil)
	copied := decoded[workspace.Duplicated](t, owner.ok("POST", "/api/v1/workspaces/"+workspaceID+"/duplicate", nil))
	assertCopiedLinks(t, owner, copied.ID, firstID, second.ID)
	raw := owner.ok("POST", "/api/v1/workspaces/"+workspaceID+"/export", transfer.Selection{}).Body.Bytes()
	imported := decoded[transfer.Result](t, importNative(owner, raw, "", owner.csrf))
	assertCopiedLinks(t, owner, imported.ID, firstID, second.ID)
	// A new link to another private workspace is invalid even for its owner.
	_, otherPage := owner.workspace()
	source.Elements[0].Link = &canvas.ElementLink{PageID: otherPage, ElementID: linkTargetID}
	lease := owner.acquire(firstID, "source-editor", "acquire")
	response := owner.request("PUT", "/api/v1/pages/"+firstID+"/document", canvas.SaveDocumentCommand{ClientID: "source-editor", MutationID: "invalid-link", Epoch: lease.Epoch, BaseRevision: 2, Document: source})
	if response.Code != 400 {
		t.Fatal("cross-workspace link accepted", response.Code, response.Body.String())
	}
}

func saveLinkedDocument(client *testClient, pageID, editor string, document canvas.Document, revision int) {
	client.t.Helper()
	lease := client.acquire(pageID, editor, "acquire")
	client.ok("PUT", "/api/v1/pages/"+pageID+"/document", canvas.SaveDocumentCommand{ClientID: editor, MutationID: editor + strconv.Itoa(revision), Epoch: lease.Epoch, BaseRevision: revision, Document: document})
}

func assertCopiedLinks(t *testing.T, client *testClient, workspaceID, oldFirst, oldSecond string) {
	t.Helper()
	pages := decoded[[]page.Page](t, client.ok("GET", "/api/v1/workspaces/"+workspaceID+"/pages", nil))
	if len(pages) != 2 {
		t.Fatal("copied pages missing", pages)
	}
	documents := make([]canvas.Document, len(pages))
	for i, entry := range pages {
		snapshot := decoded[struct {
			Document canvas.Document `json:"document"`
		}](t, client.ok("GET", "/api/v1/pages/"+entry.ID+"/document", nil))
		documents[i] = snapshot.Document
	}
	link := documents[0].Elements[0].Link
	if pages[0].ID == oldFirst || pages[1].ID == oldSecond || link == nil || link.PageID != pages[1].ID || link.ElementID != documents[1].Elements[0].ID {
		t.Fatal("copied link targets original content", link, pages)
	}
}

const linkTargetID = "link-target"

const linkTestColor = "#fefefe"
