package app_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"mindflow/internal/workspace"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func (c *testClient) workspace() (string, string) {
	c.t.Helper()
	w := c.ok("POST", "/api/v1/workspaces", map[string]string{"name": "Ideas"})
	var value struct {
		ID     string `json:"id"`
		PageID string `json:"pageId"`
	}
	checkErr24 := json.Unmarshal(w.Body.Bytes(), &value)
	if checkErr24 != nil {
		c.t.Fatal(checkErr24)
	}

	return value.ID, value.PageID
}

func (c *testClient) acquire(page, client, action string) workspace.LeaseResult {
	c.t.Helper()
	w := c.ok("POST", "/api/v1/pages/"+page+"/lease", map[string]any{"clientId": client, "action": action})
	var lease workspace.LeaseResult
	checkErr33 := json.Unmarshal(w.Body.Bytes(), &lease)
	if checkErr33 != nil {
		c.t.Fatal(checkErr33)
	}

	return lease
}

func TestLeaseFencingIdempotencyAndRecovery(t *testing.T) {
	t.Parallel()

	client := fixture(t)
	client.setup()
	wid, page := client.workspace()
	first := client.acquire(page, "first-tab", "acquire")
	second := client.acquire(page, "second-tab", "acquire")
	if !first.Editable || second.Editable {
		t.Fatal("multiple writers acquired control")
	}
	doc := emptyDocument()
	doc.Elements = append(doc.Elements, workspace.Element{ID: "rectangle", Type: "shape", X: 10, Y: 20, Width: 100, Height: 80, Shape: "rectangle", Fill: "#ffffff", Border: "#000000", BorderWidth: 1})
	req := workspace.SaveDocumentCommand{BaseRevision: 0, Epoch: first.Epoch, ClientID: "first-tab", MutationID: "mutation-1", Document: doc}
	client.ok("PUT", "/api/v1/pages/"+page+"/document", req)
	// A lost acknowledgment can be retried without increasing the revision.
	client.ok("PUT", "/api/v1/pages/"+page+"/document", req)
	var rev int
	checkErr56 := client.db.QueryRowContext(t.Context(), "SELECT revision FROM pages WHERE id=?", page).Scan(&rev)
	if checkErr56 != nil {
		t.Fatal(checkErr56)
	}
	if rev != 1 {
		t.Fatalf("retry applied twice: %d", rev)
	}
	control := client.acquire(page, "second-tab", "takeover")
	if control.Epoch <= first.Epoch {
		t.Fatal("takeover did not fence old holder")
	}
	req.BaseRevision = 1
	req.MutationID = "mutation-2"
	if client.request("PUT", "/api/v1/pages/"+page+"/document", req).Code != 409 {
		t.Fatal("stale editor overwrote page")
	}
	req.Epoch = control.Epoch
	req.ClientID = "second-tab"
	req.BaseRevision = 0
	if client.request("PUT", "/api/v1/pages/"+page+"/document", req).Code != 409 {
		t.Fatal("stale revision accepted")
	}
	req.BaseRevision = 1
	client.ok("PUT", "/api/v1/pages/"+page+"/document", req)
	client.ok("POST", "/api/v1/pages/recovery-copy", map[string]any{"workspaceId": wid, "name": "Recovered", "document": doc})
	var count int
	checkErr79 := client.db.QueryRowContext(t.Context(), "SELECT count(*) FROM pages WHERE workspace_id=?", wid).Scan(&count)
	if checkErr79 != nil {
		t.Fatal(checkErr79)
	}
	if count != 2 {
		t.Fatal("recovery did not create an independent page")
	}
	_, checkErr83 := client.db.ExecContext(t.Context(), "UPDATE page_leases SET expires_at=0 WHERE page_id=?", page)
	if checkErr83 != nil {
		t.Fatal(checkErr83)
	}
	req.BaseRevision = 2
	req.MutationID = "mutation-3"
	if client.request("PUT", "/api/v1/pages/"+page+"/document", req).Code != 409 {
		t.Fatal("expired lease accepted")
	}
}

func (c *testClient) uploadImage() workspace.AssetInfo {
	c.t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, 1024, 1024))
	im.Set(1, 1, color.NRGBA{R: 255, A: 255})
	var pngData bytes.Buffer
	checkErr96 := png.Encode(&pngData, im)
	if checkErr96 != nil {
		c.t.Fatal(checkErr96)
	}
	var body bytes.Buffer
	multipartWriter := multipart.NewWriter(&body)
	file, _ := multipartWriter.CreateFormFile("file", "test.png")
	_, checkErr100 := file.Write(pngData.Bytes())
	if checkErr100 != nil {
		c.t.Fatal(checkErr100)
	}
	checkErr101 := multipartWriter.Close()
	if checkErr101 != nil {
		c.t.Fatal(checkErr101)
	}
	r := httptest.NewRequestWithContext(c.t.Context(), http.MethodPost, "/api/v1/assets", &body)
	r.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	r.Header.Set("X-CSRF-Token", c.csrf)
	r.AddCookie(c.cookie)
	writer := httptest.NewRecorder()
	c.a.ServeHTTP(writer, r)
	if writer.Code != 200 {
		c.t.Fatalf("upload: %d %s", writer.Code, writer.Body.String())
	}
	var result workspace.AssetInfo
	checkErr112 := json.Unmarshal(writer.Body.Bytes(), &result)
	if checkErr112 != nil {
		c.t.Fatal(checkErr112)
	}

	return result
}

func TestPrivateAssetsDuplicationAndUserCascade(t *testing.T) {
	t.Parallel()

	client := fixture(t)
	client.setup()
	wid, page := client.workspace()
	asset := client.uploadImage()
	lease := client.acquire(page, "owner-tab", "acquire")
	doc := emptyDocument()
	doc.Elements = append(doc.Elements, workspace.Element{ID: "image1", Type: "image", X: 0, Y: 0, Width: 100, Height: 100, AssetID: asset.ID})
	client.ok("PUT", "/api/v1/pages/"+page+"/document", workspace.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "owner-tab", MutationID: "image-save", Document: doc})
	client.a.RunVariantForTest(t.Context())
	meta := client.ok("GET", "/api/v1/assets/"+asset.ID+"/meta", nil)
	if !strings.Contains(meta.Body.String(), "128") {
		t.Fatal("downscaled image missing")
	}
	w := client.ok("POST", "/api/v1/workspaces/"+wid+"/duplicate", map[string]any{})
	var copied struct {
		ID string `json:"id"`
	}
	checkErr137 := json.Unmarshal(w.Body.Bytes(), &copied)
	if checkErr137 != nil {
		t.Fatal(checkErr137)
	}
	client.ok("DELETE", "/api/v1/workspaces/"+wid, map[string]any{"revision": 1, confirmField: true})
	client.ok("GET", "/api/v1/assets/"+asset.ID+"/meta", nil)
	client.ok("POST", "/api/v1/admin/users", map[string]string{emailField: "admin2@example.com", "role": "admin"})
	token := client.token("admin2@example.com", "invite")
	client.ok("POST", "/api/v1/auth/accept-invite", map[string]string{tokenField: token, passwordField: "second admin password"})
	other := &testClient{a: client.a, db: client.db, t: t}
	other.login("admin2@example.com", "second admin password")
	for _, path := range []string{"/api/v1/assets/" + asset.ID, "/api/v1/assets/" + asset.ID + "?bucket=128", "/api/v1/assets/" + asset.ID + "/meta", "/api/v1/workspaces/" + copied.ID + "/pages"} {
		if other.request("GET", path, nil).Code != 404 {
			t.Fatalf("administrator accessed private content: %s", path)
		}
	}
	_, otherPage := other.workspace()
	lease2 := other.acquire(otherPage, "other-tab", "acquire")
	if other.request("PUT", "/api/v1/pages/"+otherPage+"/document", workspace.SaveDocumentCommand{Epoch: lease2.Epoch, ClientID: "other-tab", MutationID: "asset-theft", Document: doc}).Code != 400 {
		t.Fatal("foreign asset attached to another user")
	}
	var owner string
	checkErr156 := client.db.QueryRowContext(t.Context(), "SELECT id FROM users WHERE email='owner@example.com'").Scan(&owner)
	if checkErr156 != nil {
		t.Fatal(checkErr156)
	}
	other.ok("DELETE", "/api/v1/admin/users/"+owner, map[string]bool{confirmField: true})
	for _, table := range []string{"assets", "workspaces"} {
		var count int
		column := "owner_id"
		checkErr161 := client.db.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table+" WHERE "+column+"=?", owner).Scan(&count)
		if checkErr161 != nil {
			t.Fatal(checkErr161)
		}
		if count != 0 {
			t.Fatalf("%s survived user deletion", table)
		}
	}
	if client.request("GET", "/api/v1/auth/me", nil).Code != 401 {
		t.Fatal("deleted user still authenticated")
	}
}
