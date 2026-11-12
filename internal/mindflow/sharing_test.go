package mindflow_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"mindflow/internal/workspace/page/canvas"
	"net/http"
	"net/http/httptest"
	"testing"
)

func sharedUser(owner *testClient, email string) (*testClient, string) {
	owner.t.Helper()
	owner.ok("POST", "/api/v1/admin/users", map[string]string{emailField: email, roleField: userRole})
	owner.ok("POST", "/api/v1/auth/accept-invite", map[string]string{tokenField: owner.token(email, "invite"), passwordField: friendPassword})
	user := &testClient{a: owner.a, db: owner.db, t: owner.t}
	user.login(email, friendPassword)
	var userID string
	err := owner.db.QueryRowContext(owner.t.Context(), "SELECT id FROM users WHERE email=?", email).Scan(&userID)
	if err != nil {
		owner.t.Fatal(err)
	}

	return user, userID
}

func TestWorkspaceSharingRolesCommentsAndRevocation(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	member, userID := sharedUser(owner, "collaborator@example.com")
	workspaceID, pageID := owner.workspace()
	documentPath := "/api/v1/pages/" + pageID + "/document"
	leasePath := "/api/v1/pages/" + pageID + "/lease"
	commentsPath := "/api/v1/pages/" + pageID + "/comments"
	membersPath := "/api/v1/workspaces/" + workspaceID + "/members"
	asset := owner.uploadImage()
	hiddenAsset := owner.uploadImage()
	lease := owner.acquire(pageID, "owner-client", "acquire")
	doc := emptyDocument()
	doc.Elements = append(doc.Elements, canvas.Element{ID: "shared-image", Type: imageElementKind, Width: 100, Height: 100, AssetID: asset.ID})
	owner.ok("PUT", documentPath, canvas.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "owner-client", MutationID: "initial-shared-image", Document: doc})
	if member.request("GET", documentPath, nil).Code != 404 {
		t.Fatal("private document was visible")
	}
	for _, role := range []string{"read", "comment", fullAccessRole} {
		owner.ok("PUT", membersPath, map[string]string{emailField: "collaborator@example.com", roleField: role})
		member.ok("GET", documentPath, nil)
		member.ok("GET", "/api/v1/assets/"+asset.ID, nil)
		member.ok("GET", "/api/v1/assets/"+asset.ID+"/meta", nil)
		if member.request("GET", "/api/v1/assets/"+hiddenAsset.ID, nil).Code != 404 {
			t.Fatal("sharing exposed an unrelated private image")
		}
		if member.request("GET", membersPath, nil).Code != 403 {
			t.Fatal("member read owner-only sharing settings")
		}
		if member.request("PUT", membersPath, map[string]string{emailField: "owner@example.com", roleField: fullAccessRole}).Code != 403 {
			t.Fatal("member changed sharing")
		}
		response := member.ok("POST", leasePath, map[string]string{clientIDField: sharedClientID, actionField: "acquire"})
		var state struct {
			Editable   bool   `json:"editable"`
			Permission string `json:"permission"`
		}
		err := json.Unmarshal(response.Body.Bytes(), &state)
		if err != nil {
			t.Fatal(err)
		}
		if state.Editable || state.Permission != role {
			t.Fatalf("unexpected member lease: %+v", state)
		}
		current := owner.acquire(pageID, "owner-client", "acquire")
		if !current.Editable {
			t.Fatal("adding a member stole the owner's lease")
		}
		checkReadOnlyRestrictions(member, role, workspaceID, pageID, doc, lease.Epoch)
	}
	takeover := member.acquire(pageID, sharedClientID, "takeover")
	if !takeover.Editable {
		t.Fatal("full member could not take control")
	}
	if owner.request("PUT", documentPath, canvas.SaveDocumentCommand{BaseRevision: 1, Epoch: lease.Epoch, ClientID: "owner-client", MutationID: "stale-owner-save", Document: doc}).Code != 409 {
		t.Fatal("former writer retained control")
	}
	bad := doc
	bad.Elements = append(append([]canvas.Element{}, doc.Elements...), canvas.Element{ID: "private-theft", Type: imageElementKind, Width: 100, Height: 100, AssetID: hiddenAsset.ID})
	if member.request("PUT", documentPath, canvas.SaveDocumentCommand{BaseRevision: 1, Epoch: takeover.Epoch, ClientID: sharedClientID, MutationID: "hidden-asset-theft", Document: bad}).Code != 400 {
		t.Fatal("member attached a private image from another workspace")
	}
	member.ok("PUT", documentPath, canvas.SaveDocumentCommand{BaseRevision: 1, Epoch: takeover.Epoch, ClientID: sharedClientID, MutationID: "shared-save-success", Document: doc})
	checkCommentPermissions(owner, member, workspaceID, pageID)
	owner.ok("DELETE", membersPath+"/"+userID, map[string]any{})
	for _, path := range []string{documentPath, commentsPath, "/api/v1/assets/" + asset.ID, "/api/v1/assets/" + asset.ID + "/meta"} {
		if member.request("GET", path, nil).Code != 404 {
			t.Fatalf("revoked member retained %s", path)
		}
	}
	if member.request("POST", leasePath, map[string]string{clientIDField: sharedClientID, actionField: "takeover"}).Code != 404 {
		t.Fatal("revoked member could acquire a lease")
	}
	owner.acquire(pageID, "owner-client", "acquire")
}

func TestSharedUploadsSurviveCollaboratorDeletion(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	member, userID := sharedUser(owner, "uploader@example.com")
	workspaceID, pageID := owner.workspace()
	owner.ok("PUT", "/api/v1/workspaces/"+workspaceID+"/members", map[string]string{emailField: "uploader@example.com", roleField: fullAccessRole})
	var imageData bytes.Buffer
	err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 128, 128)))
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	err = writer.WriteField("workspaceId", workspaceID)
	if err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("file", "shared.png")
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write(imageData.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the actual multipart endpoint with this collaborator's cookie and CSRF.
	request := member.uploadRequest(&body, writer.FormDataContentType())
	var asset struct {
		ID string `json:"id"`
	}
	err = json.Unmarshal(request.Body.Bytes(), &asset)
	if err != nil {
		t.Fatal(err)
	}
	member.ok("POST", "/api/v1/assets/"+asset.ID+"/variant", map[string]int{"width": 32, "height": 32})
	owner.a.RunVariantForTest(t.Context())
	meta := owner.ok("GET", "/api/v1/assets/"+asset.ID+"/meta", nil)
	var variants struct {
		Variants []int `json:"variants"`
	}
	err = json.Unmarshal(meta.Body.Bytes(), &variants)
	if err != nil {
		t.Fatal(err)
	}
	if len(variants.Variants) == 0 {
		t.Fatal("shared upload preview was not generated under the workspace owner")
	}
	lease := member.acquire(pageID, "uploader-client", "acquire")
	doc := emptyDocument()
	doc.Elements = append(doc.Elements, canvas.Element{ID: "uploaded", Type: imageElementKind, Width: 100, Height: 100, AssetID: asset.ID})
	member.ok("PUT", "/api/v1/pages/"+pageID+"/document", canvas.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "uploader-client", MutationID: "uploaded-shared-image", Document: doc})
	owner.ok("DELETE", "/api/v1/admin/users/"+userID, map[string]bool{confirmField: true})
	result := owner.ok("GET", "/api/v1/assets/"+asset.ID, nil)
	if !bytes.Equal(result.Body.Bytes(), imageData.Bytes()) {
		t.Fatal("deleting collaborator changed a shared image")
	}
	owner.ok("GET", "/api/v1/pages/"+pageID+"/document", nil)
}

func (c *testClient) uploadRequest(body io.Reader, contentType string) *httptest.ResponseRecorder {
	c.t.Helper()
	request := httptest.NewRequestWithContext(c.t.Context(), http.MethodPost, "/api/v1/assets", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("X-CSRF-Token", c.csrf)
	request.AddCookie(c.cookie)
	response := httptest.NewRecorder()
	c.a.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		c.t.Fatalf("upload: %d %s", response.Code, response.Body.String())
	}

	return response
}

func checkCommentPermissions(owner, member *testClient, workspaceID, pageID string) {
	owner.t.Helper()
	t := owner.t
	commentsPath := "/api/v1/pages/" + pageID + "/comments"
	membersPath := "/api/v1/workspaces/" + workspaceID + "/members"
	root := owner.ok("POST", commentsPath, map[string]any{commentBodyField: "Review this", "x": 100, "y": 200})
	var thread struct {
		ID string `json:"id"`
	}
	err := json.Unmarshal(root.Body.Bytes(), &thread)
	if err != nil {
		t.Fatal(err)
	}
	replyPath := "/api/v1/comments/" + thread.ID
	member.ok("POST", replyPath+"/replies", map[string]string{commentBodyField: "Looks good"})
	if member.request("PATCH", replyPath, map[string]bool{resolvedField: true}).Code != 403 {
		t.Fatal("full access let a member resolve someone else's thread")
	}
	owner.ok("PUT", membersPath, map[string]string{emailField: "collaborator@example.com", roleField: "comment"})
	member.ok("POST", replyPath+"/replies", map[string]string{commentBodyField: "Still commenting after downgrade"})
	owner.ok("PATCH", replyPath, map[string]bool{resolvedField: true})
	if member.request("POST", replyPath+"/replies", map[string]string{commentBodyField: "Closed"}).Code != 400 {
		t.Fatal("resolved thread accepted a reply")
	}
	owner.ok("PATCH", replyPath, map[string]bool{resolvedField: false})
	own := member.ok("POST", commentsPath, map[string]any{commentBodyField: "My point", "x": -50, "y": 80})
	err = json.Unmarshal(own.Body.Bytes(), &thread)
	if err != nil {
		t.Fatal(err)
	}
	member.ok("PATCH", "/api/v1/comments/"+thread.ID, map[string]bool{resolvedField: true})
	member.ok("DELETE", "/api/v1/comments/"+thread.ID, map[string]any{})
}

const (
	userRole         = "user"
	imageElementKind = "image"
	fullAccessRole   = "full"
	actionField      = "action"
	nameField        = "name"
	commentBodyField = "body"
	resolvedField    = "resolved"
)

func checkReadOnlyRestrictions(member *testClient, role, workspaceID, pageID string, doc canvas.Document, epoch int64) {
	member.t.Helper()
	t := member.t
	documentPath := "/api/v1/pages/" + pageID + "/document"
	leasePath := "/api/v1/pages/" + pageID + "/lease"
	commentsPath := "/api/v1/pages/" + pageID + "/comments"
	if role != fullAccessRole {
		if member.request("POST", leasePath, map[string]string{clientIDField: sharedClientID, actionField: "takeover"}).Code != 403 {
			t.Fatal("non-editor stole control")
		}
		if member.request("POST", "/api/v1/workspaces/"+workspaceID+"/pages", map[string]string{nameField: "Forbidden"}).Code != 403 {
			t.Fatal("non-editor created a page")
		}
		if member.request("PUT", documentPath, canvas.SaveDocumentCommand{Epoch: epoch, ClientID: sharedClientID, MutationID: "forbidden-save", Document: doc}).Code != 403 {
			t.Fatal("non-editor saved content")
		}
	}
	if role == "read" && member.request("POST", commentsPath, map[string]any{commentBodyField: "Forbidden", "x": 10, "y": 20}).Code != 403 {
		t.Fatal("read-only user created a comment")
	}
}

const clientIDField = "clientId"

const sharedClientID = "shared-client"
