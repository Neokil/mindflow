package app

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func (c *testClient) workspace() (string, string) {
	c.t.Helper()
	w := c.ok("POST", "/api/v1/workspaces", map[string]string{"name": "Ideas"})
	var v struct {
		ID     string `json:"id"`
		PageID string `json:"pageId"`
	}
	json.Unmarshal(w.Body.Bytes(), &v)
	return v.ID, v.PageID
}
func (c *testClient) acquire(page, client, action string) leaseResult {
	c.t.Helper()
	w := c.ok("POST", "/api/v1/pages/"+page+"/lease", map[string]any{"clientId": client, "action": action})
	var l leaseResult
	json.Unmarshal(w.Body.Bytes(), &l)
	return l
}
func TestLeaseFencingIdempotencyAndRecovery(t *testing.T) {
	c := fixture(t)
	c.setup()
	wid, page := c.workspace()
	first := c.acquire(page, "first-tab", "acquire")
	second := c.acquire(page, "second-tab", "acquire")
	if !first.Editable || second.Editable {
		t.Fatal("multiple writers acquired control")
	}
	doc := emptyDocument()
	doc.Elements = append(doc.Elements, Element{ID: "rectangle", Type: "shape", X: 10, Y: 20, Width: 100, Height: 80, Shape: "rectangle", Fill: "#ffffff", Border: "#000000", BorderWidth: 1})
	req := saveRequest{BaseRevision: 0, Epoch: first.Epoch, ClientID: "first-tab", MutationID: "mutation-1", Document: doc}
	c.ok("PUT", "/api/v1/pages/"+page+"/document", req)
	// A lost acknowledgment can be retried without increasing the revision.
	c.ok("PUT", "/api/v1/pages/"+page+"/document", req)
	var rev int
	c.db.QueryRow("SELECT revision FROM pages WHERE id=?", page).Scan(&rev)
	if rev != 1 {
		t.Fatalf("retry applied twice: %d", rev)
	}
	control := c.acquire(page, "second-tab", "takeover")
	if control.Epoch <= first.Epoch {
		t.Fatal("takeover did not fence old holder")
	}
	req.BaseRevision = 1
	req.MutationID = "mutation-2"
	if c.request("PUT", "/api/v1/pages/"+page+"/document", req).Code != 409 {
		t.Fatal("stale editor overwrote page")
	}
	req.Epoch = control.Epoch
	req.ClientID = "second-tab"
	req.BaseRevision = 0
	if c.request("PUT", "/api/v1/pages/"+page+"/document", req).Code != 409 {
		t.Fatal("stale revision accepted")
	}
	req.BaseRevision = 1
	c.ok("PUT", "/api/v1/pages/"+page+"/document", req)
	c.ok("POST", "/api/v1/pages/recovery-copy", map[string]any{"workspaceId": wid, "name": "Recovered", "document": doc})
	var count int
	c.db.QueryRow("SELECT count(*) FROM pages WHERE workspace_id=?", wid).Scan(&count)
	if count != 2 {
		t.Fatal("recovery did not create an independent page")
	}
	c.db.Exec("UPDATE page_leases SET expires_at=0 WHERE page_id=?", page)
	req.BaseRevision = 2
	req.MutationID = "mutation-3"
	if c.request("PUT", "/api/v1/pages/"+page+"/document", req).Code != 409 {
		t.Fatal("expired lease accepted")
	}
}
func (c *testClient) uploadImage() assetInfo {
	c.t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, 1024, 1024))
	im.Set(1, 1, color.NRGBA{R: 255, A: 255})
	var pngData bytes.Buffer
	png.Encode(&pngData, im)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	file, _ := mw.CreateFormFile("file", "test.png")
	file.Write(pngData.Bytes())
	mw.Close()
	r := httptest.NewRequest("POST", "/api/v1/assets", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("X-CSRF-Token", c.csrf)
	r.AddCookie(c.cookie)
	w := httptest.NewRecorder()
	c.a.ServeHTTP(w, r)
	if w.Code != 200 {
		c.t.Fatalf("upload: %d %s", w.Code, w.Body.String())
	}
	var result assetInfo
	json.Unmarshal(w.Body.Bytes(), &result)
	return result
}
func TestPrivateAssetsDuplicationAndUserCascade(t *testing.T) {
	c := fixture(t)
	c.setup()
	wid, page := c.workspace()
	asset := c.uploadImage()
	lease := c.acquire(page, "owner-tab", "acquire")
	doc := emptyDocument()
	doc.Elements = append(doc.Elements, Element{ID: "image1", Type: "image", X: 0, Y: 0, Width: 100, Height: 100, AssetID: asset.ID})
	c.ok("PUT", "/api/v1/pages/"+page+"/document", saveRequest{Epoch: lease.Epoch, ClientID: "owner-tab", MutationID: "image-save", Document: doc})
	c.a.repository.RunVariant(context.Background())
	meta := c.ok("GET", "/api/v1/assets/"+asset.ID+"/meta", nil)
	if !strings.Contains(meta.Body.String(), "128") {
		t.Fatal("downscaled image missing")
	}
	w := c.ok("POST", "/api/v1/workspaces/"+wid+"/duplicate", map[string]any{})
	var copied struct {
		ID string `json:"id"`
	}
	json.Unmarshal(w.Body.Bytes(), &copied)
	c.ok("DELETE", "/api/v1/workspaces/"+wid, map[string]any{"revision": 1, "confirm": true})
	c.ok("GET", "/api/v1/assets/"+asset.ID+"/meta", nil)
	c.ok("POST", "/api/v1/admin/users", map[string]string{"email": "admin2@example.com", "role": "admin"})
	token := c.token("admin2@example.com", "invite")
	c.ok("POST", "/api/v1/auth/accept-invite", map[string]string{"token": token, "password": "second admin password"})
	other := &testClient{a: c.a, db: c.db, t: t}
	other.login("admin2@example.com", "second admin password")
	for _, path := range []string{"/api/v1/assets/" + asset.ID, "/api/v1/assets/" + asset.ID + "?bucket=128", "/api/v1/assets/" + asset.ID + "/meta", "/api/v1/workspaces/" + copied.ID + "/pages"} {
		if other.request("GET", path, nil).Code != 404 {
			t.Fatalf("administrator accessed private content: %s", path)
		}
	}
	_, otherPage := other.workspace()
	lease2 := other.acquire(otherPage, "other-tab", "acquire")
	if other.request("PUT", "/api/v1/pages/"+otherPage+"/document", saveRequest{Epoch: lease2.Epoch, ClientID: "other-tab", MutationID: "asset-theft", Document: doc}).Code != 400 {
		t.Fatal("foreign asset attached to another user")
	}
	var owner string
	c.db.QueryRow("SELECT id FROM users WHERE email='owner@example.com'").Scan(&owner)
	other.ok("DELETE", "/api/v1/admin/users/"+owner, map[string]bool{"confirm": true})
	for _, table := range []string{"assets", "workspaces"} {
		var n int
		column := "owner_id"
		c.db.QueryRow("SELECT count(*) FROM "+table+" WHERE "+column+"=?", owner).Scan(&n)
		if n != 0 {
			t.Fatalf("%s survived user deletion", table)
		}
	}
	if c.request("GET", "/api/v1/auth/me", nil).Code != 401 {
		t.Fatal("deleted user still authenticated")
	}
}
