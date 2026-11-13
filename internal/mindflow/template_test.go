package mindflow_test

import (
	"encoding/json"
	"mindflow/internal/workspace"
	imageasset "mindflow/internal/workspace/image"
	"mindflow/internal/workspace/page"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/storage"
	"mindflow/internal/workspace/template"
	"testing"
)

func TestTemplatesRespectMembershipAndRetainImages(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, pageID := owner.workspace()
	asset := owner.uploadImage()
	elements := []canvas.Element{
		{ID: "banner", Type: imageElementKind, Width: 200, Height: 100, AssetID: asset.ID, GroupID: "heading", Locked: true},
		{ID: "title", Type: "text", Width: 200, Height: 100, Text: "Reusable heading", FontSize: 24, Color: "#ffffff", Align: "center", GroupID: "heading", Locked: true},
	}
	document := canvas.NewDocument()
	document.Elements = elements
	lease := owner.acquire(pageID, "template-editor", "acquire")
	owner.ok("PUT", "/api/v1/pages/"+pageID+"/document", canvas.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "template-editor", MutationID: "template-source-save", Document: document})
	path := "/api/v1/workspaces/" + workspaceID + "/templates"
	command := template.CreateCommand{Name: "Heading", Elements: elements}
	var created template.Summary
	response := owner.ok("POST", path, command)
	err := json.Unmarshal(response.Body.Bytes(), &created)
	if err != nil {
		t.Fatal(err)
	}
	if owner.request("POST", path, command).Code != 400 {
		t.Fatal("duplicate template name accepted")
	}
	member, memberID := sharedUser(owner, "template-reader@example.com")
	if member.request("GET", path, nil).Code != 404 {
		t.Fatal("private template list leaked")
	}
	if member.request("GET", path+"/"+created.ID, nil).Code != 404 {
		t.Fatal("private template leaked")
	}
	membersPath := "/api/v1/workspaces/" + workspaceID + "/members"
	owner.ok("PUT", membersPath, map[string]string{emailField: "template-reader@example.com", roleField: readAccessRole})
	member.ok("GET", path, nil)
	var snapshot template.Snapshot
	response = member.ok("GET", path+"/"+created.ID, nil)
	err = json.Unmarshal(response.Body.Bytes(), &snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Elements) != 2 || snapshot.Elements[0].Locked || snapshot.Elements[0].GroupID != "" {
		t.Fatal("template did not contain independent editable elements")
	}
	if member.request("POST", path, template.CreateCommand{Name: "Read-only creation", Elements: elements}).Code != 403 || member.request("DELETE", path+"/"+created.ID, nil).Code != 403 {
		t.Fatal("read-only member changed templates")
	}
	// Retention does not depend on a surviving source page or its page_assets links.
	owner.ok("DELETE", "/api/v1/pages/"+pageID, page.DeletePageCommand{Revision: 1, Confirm: true})
	member.ok("GET", "/api/v1/assets/"+asset.ID, nil)
	_, err = owner.db.ExecContext(t.Context(), "UPDATE assets SET created_at=0 WHERE id=?", asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	err = imageasset.NewService(storage.NewRepository(owner.db), func(string, string, string) {}).Cleanup(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	member.ok("GET", "/api/v1/assets/"+asset.ID, nil)
	owner.ok("PUT", membersPath, map[string]string{emailField: "template-reader@example.com", roleField: "full"})
	hidden := member.uploadImage()
	if member.request("POST", path, template.CreateCommand{Name: "Foreign image", Elements: []canvas.Element{{ID: "foreign", Type: imageElementKind, Width: 100, Height: 100, AssetID: hidden.ID}}}).Code != 400 {
		t.Fatal("template accepted an image outside its workspace")
	}
	owner.ok("DELETE", membersPath+"/"+memberID, nil)
	if member.request("GET", path+"/"+created.ID, nil).Code != 404 || member.request("GET", "/api/v1/assets/"+asset.ID, nil).Code != 404 {
		t.Fatal("revoked template access remained")
	}
	owner.ok("DELETE", path+"/"+created.ID, nil)
	owner.ok("GET", path, nil)
	var count int
	err = owner.db.QueryRowContext(t.Context(), "SELECT count(*) FROM template_assets").Scan(&count)
	if err != nil || count != 0 {
		t.Fatal("removed template retained image links", err)
	}
}

func TestWorkspaceTemplateCopyKeepsImagesAfterSourceDeletion(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, _ := owner.workspace()
	asset := owner.uploadImage()
	response := owner.ok("POST", "/api/v1/workspaces/"+workspaceID+"/templates", template.CreateCommand{Name: "Banner", Elements: []canvas.Element{{ID: "banner", Type: imageElementKind, Width: 200, Height: 100, AssetID: asset.ID}}})
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	response = owner.ok("POST", "/api/v1/workspaces/"+workspaceID+"/duplicate", nil)
	var copied struct {
		ID string `json:"id"`
	}
	err := json.Unmarshal(response.Body.Bytes(), &copied)
	if err != nil {
		t.Fatal(err)
	}
	owner.ok("DELETE", "/api/v1/workspaces/"+workspaceID, workspace.DeleteWorkspaceCommand{Revision: 1, Confirm: true})
	member, _ := sharedUser(owner, "template-copy@example.com")
	owner.ok("PUT", "/api/v1/workspaces/"+copied.ID+"/members", map[string]string{emailField: "template-copy@example.com", roleField: fullAccessRole})
	response = member.ok("GET", "/api/v1/workspaces/"+copied.ID+"/templates", nil)
	var summaries []template.Summary
	err = json.Unmarshal(response.Body.Bytes(), &summaries)
	if err != nil || len(summaries) != 1 {
		t.Fatal("template library did not copy", err)
	}
	member.ok("GET", "/api/v1/assets/"+asset.ID, nil)
	var pageID string
	err = owner.db.QueryRowContext(t.Context(), "SELECT id FROM pages WHERE workspace_id=?", copied.ID).Scan(&pageID)
	if err != nil {
		t.Fatal(err)
	}
	document := canvas.NewDocument()
	document.Elements = []canvas.Element{{ID: "inserted-banner", Type: imageElementKind, Width: 200, Height: 100, AssetID: asset.ID}}
	lease := member.acquire(pageID, "template-copy-editor", "acquire")
	member.ok("PUT", "/api/v1/pages/"+pageID+"/document", canvas.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "template-copy-editor", MutationID: "template-insert-copy", Document: document})
	owner.ok("DELETE", "/api/v1/workspaces/"+copied.ID, workspace.DeleteWorkspaceCommand{Revision: 1, Confirm: true})
	if owner.request("GET", "/api/v1/assets/"+asset.ID, nil).Code != 404 {
		t.Fatal("workspace deletion retained template-only image")
	}
}
