package mindflow_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func pdfUpload(client *testClient, workspaceID string, data []byte, csrf string) *httptest.ResponseRecorder {
	client.t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "sample.pdf")
	if err != nil {
		client.t.Fatal(err)
	}
	_, err = file.Write(data)
	if err != nil {
		client.t.Fatal(err)
	}
	preview, err := form.CreateFormFile("preview", "preview.png")
	if err != nil {
		client.t.Fatal(err)
	}
	err = png.Encode(preview, image.NewNRGBA(image.Rect(0, 0, 10, 10)))
	if err != nil {
		client.t.Fatal(err)
	}
	err = form.Close()
	if err != nil {
		client.t.Fatal(err)
	}
	request := httptest.NewRequestWithContext(client.t.Context(), "POST", "/api/v1/workspaces/"+workspaceID+"/pdf-assets", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.Header.Set("X-CSRF-Token", csrf)
	if client.cookie != nil {
		request.AddCookie(client.cookie)
	}
	response := httptest.NewRecorder()
	client.a.ServeHTTP(response, request)

	return response
}

func TestPDFAssetsRequireWorkspaceAccess(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, _ := owner.workspace()
	reader, _ := sharedUser(owner, "pdf-reader@example.com")
	raw := []byte("%PDF-1.4\nopaque stored PDF payload\n%%EOF\n")
	if pdfUpload(owner, workspaceID, raw, "").Code != 403 {
		t.Fatal("PDF upload bypassed CSRF")
	}
	if pdfUpload(owner, workspaceID, []byte("not a PDF"), owner.csrf).Code != 400 {
		t.Fatal("accepted non-PDF")
	}
	response := pdfUpload(owner, workspaceID, raw, owner.csrf)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	var asset struct {
		ID string `json:"id"`
	}
	err := json.Unmarshal(response.Body.Bytes(), &asset)
	if err != nil {
		t.Fatal(err)
	}
	assetPath := "/api/v1/assets/" + asset.ID
	if reader.request("GET", assetPath, nil).Code != 404 {
		t.Fatal("private PDF exposed")
	}
	blob := owner.ok("GET", assetPath, nil)
	if blob.Header().Get("Content-Type") != "application/pdf" || blob.Header().Get("Content-Disposition") != "attachment; filename=document.pdf" || !bytes.Equal(blob.Body.Bytes(), raw) {
		t.Fatal("PDF original not retained safely")
	}
	owner.ok("GET", assetPath+"?bucket=1024", nil)
	owner.ok("PUT", "/api/v1/workspaces/"+workspaceID+"/members", map[string]string{"email": "pdf-reader@example.com", "role": "read"})
	reader.ok("GET", assetPath, nil)
	if pdfUpload(reader, workspaceID, raw, reader.csrf).Code != 403 {
		t.Fatal("read-only member uploaded PDF")
	}
	if reader.request("POST", "/api/v1/workspaces/"+workspaceID+"/pdf-pages", map[string]any{"pages": []map[string]any{{"name": "Blocked PDF page", "assetId": asset.ID, "width": 100, "height": 100}}}).Code != 403 {
		t.Fatal("read-only member imported pages")
	}
}
