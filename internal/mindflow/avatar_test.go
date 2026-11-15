package mindflow_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"mindflow/internal/account"
	"net/http/httptest"
	"testing"
)

func TestAvatarAuthorizationPersistenceAndRemoval(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	var raw, body bytes.Buffer
	err := png.Encode(&raw, image.NewNRGBA(image.Rect(0, 0, 160, 80)))
	if err != nil {
		t.Fatal(err)
	}
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.Write(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	err = form.Close()
	if err != nil {
		t.Fatal(err)
	}
	send := func(csrf string) *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(t.Context(), "POST", "/api/v1/auth/avatar", bytes.NewReader(body.Bytes()))
		request.AddCookie(owner.cookie)
		request.Header.Set("X-CSRF-Token", csrf)
		request.Header.Set("Content-Type", form.FormDataContentType())
		response := httptest.NewRecorder()
		owner.a.ServeHTTP(response, request)

		return response
	}
	if send("").Code != 403 {
		t.Fatal("avatar upload bypassed CSRF")
	}
	if response := send(owner.csrf); response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	var identity struct {
		User account.User `json:"user"`
	}
	response := owner.ok("GET", "/api/v1/auth/me", nil)
	err = json.Unmarshal(response.Body.Bytes(), &identity)
	if err != nil {
		t.Fatal(err)
	}
	if identity.User.AvatarVersion == "" {
		t.Fatal("avatar version not persisted")
	}
	path := "/api/v1/accounts/" + identity.User.ID + "/avatar"
	if (&testClient{a: owner.a, t: t}).request("GET", path, nil).Code != 401 {
		t.Fatal("anonymous avatar access allowed")
	}
	versioned := owner.ok("GET", path+"?v="+identity.User.AvatarVersion, nil)
	if versioned.Header().Get("Cache-Control") != immutableCache {
		t.Fatal("versioned avatar not cached")
	}
	if owner.request("GET", path+"?v=stale", nil).Code != 404 {
		t.Fatal("stale avatar URL served new content")
	}
	content := owner.ok("GET", path, nil)
	if content.Header().Get("Cache-Control") != noStoreCache {
		t.Fatal("unversioned avatar cached")
	}
	config, kind, err := image.DecodeConfig(content.Body)
	if err != nil || kind != "png" || config.Width != 128 || config.Height != 128 {
		t.Fatal("unprocessed avatar returned", config, kind, err)
	}
	owner.ok("DELETE", "/api/v1/auth/avatar", nil)
	if owner.request("GET", path, nil).Code != 404 {
		t.Fatal("removed avatar still available")
	}
}
