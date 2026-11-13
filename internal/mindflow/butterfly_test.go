package mindflow_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"mindflow/internal/workspace/butterfly"
	"mindflow/internal/workspace/page/canvas"
	"net/http/httptest"
	"os"
	"testing"
)

func importButterfly(client *testClient, raw []byte, csrf string) *httptest.ResponseRecorder {
	client.t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "sample.bfly")
	if err != nil {
		client.t.Fatal(err)
	}
	_, err = file.Write(raw)
	if err != nil {
		client.t.Fatal(err)
	}
	err = form.WriteField("name", "Imported example")
	if err != nil {
		client.t.Fatal(err)
	}
	err = form.Close()
	if err != nil {
		client.t.Fatal(err)
	}
	request := httptest.NewRequestWithContext(client.t.Context(), "POST", "/api/v1/workspaces/import/butterfly", &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.Header.Set("X-CSRF-Token", csrf)
	if client.cookie != nil {
		request.AddCookie(client.cookie)
	}
	response := httptest.NewRecorder()
	client.a.ServeHTTP(response, request)

	return response
}

func butterflySample(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../workspace/butterfly/testdata/two-pages.bfly")
	if err != nil {
		t.Fatal(err)
	}

	return raw
}

func TestButterflyImportIsPrivateAndSurvivesEdits(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	raw := butterflySample(t)
	if importButterfly(owner, raw, "").Code != 403 {
		t.Fatal("import bypassed CSRF")
	}
	response := importButterfly(owner, raw, owner.csrf)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	var result butterfly.Result
	err := json.Unmarshal(response.Body.Bytes(), &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.PageCount != 2 || result.ElementCount != 6 {
		t.Fatal("import counts changed", result)
	}
	var snapshot struct {
		Document canvas.Document `json:"document"`
	}
	documentPath := "/api/v1/pages/" + result.PageID + "/document"
	encoded := owner.ok("GET", documentPath, nil)
	err = json.Unmarshal(encoded.Body.Bytes(), &snapshot)
	if err != nil {
		t.Fatal(err)
	}
	assetID := snapshot.Document.Elements[1].AssetID
	owner.ok("GET", "/api/v1/assets/"+assetID, nil)
	other, _ := sharedUser(owner, "other-import@example.com")
	if other.request("GET", documentPath, nil).Code != 404 || other.request("GET", "/api/v1/assets/"+assetID, nil).Code != 404 {
		t.Fatal("import was not private")
	}
	lease := owner.acquire(result.PageID, "import-editor", "acquire")
	owner.ok("PUT", documentPath, canvas.SaveDocumentCommand{BaseRevision: 0, Epoch: lease.Epoch, ClientID: "import-editor", MutationID: "import-first-edit", Document: snapshot.Document})
}

func TestFailedButterflyImportRollsBackPagesAndImages(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	_, err := owner.db.ExecContext(t.Context(), `CREATE TRIGGER fail_second_import BEFORE INSERT ON pages WHEN NEW.name='Page 2' BEGIN SELECT RAISE(ABORT,'test import rollback'); END`)
	if err != nil {
		t.Fatal(err)
	}
	response := importButterfly(owner, butterflySample(t), owner.csrf)
	if response.Code != 500 {
		t.Fatal("expected storage failure", response.Code)
	}
	var count int
	err = owner.db.QueryRowContext(t.Context(), "SELECT (SELECT count(*) FROM workspaces)+(SELECT count(*) FROM pages)+(SELECT count(*) FROM assets)+(SELECT count(*) FROM page_assets)").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed import left a partial workspace or image")
	}
}
