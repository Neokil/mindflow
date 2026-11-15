package mindflow_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"mime/multipart"
	"mindflow/internal/workspace/butterfly"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/template"
	"mindflow/internal/workspace/transfer"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
)

const (
	transferShapeType      = "shape"
	transferRectangleShape = "rectangle"
)

func importNative(client *testClient, raw []byte, target, csrf string) *httptest.ResponseRecorder {
	client.t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "workspace.mindflow")
	if err != nil {
		client.t.Fatal(err)
	}
	_, err = file.Write(raw)
	if err != nil {
		client.t.Fatal(err)
	}
	err = form.Close()
	if err != nil {
		client.t.Fatal(err)
	}
	path := "/api/v1/workspaces/import/mindflow"
	if target != "" {
		path = "/api/v1/workspaces/" + target + "/import/mindflow"
	}
	request := httptest.NewRequestWithContext(client.t.Context(), "POST", path, &body)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.Header.Set("X-CSRF-Token", csrf)
	if client.cookie != nil {
		request.AddCookie(client.cookie)
	}
	response := httptest.NewRecorder()
	client.a.ServeHTTP(response, request)

	return response
}

//nolint:ireturn // Generic test decoder returns the concrete type requested by each fixture.
func decoded[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	var result T
	err := json.Unmarshal(response.Body.Bytes(), &result)
	if err != nil {
		t.Fatal(err)
	}

	return result
}

func nativeSample(t *testing.T, owner *testClient) (string, []byte) {
	t.Helper()
	imported := decoded[transfer.Result](t, importButterfly(owner, butterflySample(t), owner.csrf))
	path := "/api/v1/workspaces/" + imported.ID + "/export"
	response := owner.ok("POST", path, transfer.Selection{IncludeTemplates: true})

	return imported.ID, response.Body.Bytes()
}

func TestNativeRoundTripPreservesMediaAndEditableContent(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, raw := nativeSample(t, owner)
	bundle, contents, err := transfer.Read(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Pages) != 2 || len(bundle.Assets) != 1 || len(contents) != 1 {
		t.Fatal("missing pages or image")
	}
	pages := decoded[[]struct {
		ID string `json:"id"`
	}](t, owner.ok("GET", "/api/v1/workspaces/"+workspaceID+"/pages", nil))
	source := decoded[struct {
		Document canvas.Document `json:"document"`
	}](t, owner.ok("GET", "/api/v1/pages/"+pages[0].ID+"/document", nil)).Document
	source.Elements[0].GroupID = "locked-group"
	source.Elements[0].Locked = true
	source.Elements[1].GroupID = "locked-group"
	source.Elements[1].Locked = true
	source.Elements[2].Pressures = make([]float64, len(source.Elements[2].Points)/2)
	for index := range source.Elements[2].Pressures {
		source.Elements[2].Pressures[index] = float64(index) / float64(len(source.Elements[2].Pressures)-1)
	}
	originalAsset := source.Elements[1].AssetID
	pdfBytes := []byte("%PDF-1.4\nretained document\n%%EOF\n")
	pdfAsset := decoded[struct {
		ID string `json:"id"`
	}](t, pdfUpload(owner, workspaceID, pdfBytes, owner.csrf))
	source.Elements = append(source.Elements, canvas.Element{ID: "pdf-element", Type: "pdf", Width: 200, Height: 300, AssetID: pdfAsset.ID})
	lease := owner.acquire(pages[0].ID, "native-editor", "acquire")
	owner.ok("PUT", "/api/v1/pages/"+pages[0].ID+"/document", canvas.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "native-editor", MutationID: "native-save", Document: source})
	owner.ok("POST", "/api/v1/workspaces/"+workspaceID+"/templates", template.CreateCommand{Name: "Heading", Elements: source.Elements[:2]})
	raw = owner.ok("POST", "/api/v1/workspaces/"+workspaceID+"/export", transfer.Selection{IncludeTemplates: true}).Body.Bytes()
	other, _ := sharedUser(owner, "native-recipient@example.com")
	result := decoded[transfer.Result](t, importNative(other, raw, "", other.csrf))
	if result.ID == workspaceID || result.PageCount != 2 || result.ElementCount != 7 || result.TemplateCount != 1 {
		t.Fatal("round-trip counts or identity", result)
	}
	imported := decoded[struct {
		Document canvas.Document `json:"document"`
	}](t, other.ok("GET", "/api/v1/pages/"+result.PageID+"/document", nil)).Document
	if imported.Elements[0].ID == source.Elements[0].ID || imported.Elements[0].GroupID == source.Elements[0].GroupID || !imported.Elements[0].Locked || imported.Elements[0].GroupID != imported.Elements[1].GroupID {
		t.Fatal("element or group identity/locking was not preserved correctly")
	}
	if !slices.Equal(imported.Elements[2].Pressures, source.Elements[2].Pressures) {
		t.Fatal("native transfer changed stroke pressure")
	}
	if imported.Elements[1].AssetID == originalAsset {
		t.Fatal("media retained foreign identity")
	}
	original := owner.ok("GET", "/api/v1/assets/"+originalAsset, nil).Body.Bytes()
	copied := other.ok("GET", "/api/v1/assets/"+imported.Elements[1].AssetID, nil).Body.Bytes()
	if !bytes.Equal(original, copied) {
		t.Fatal("image original changed")
	}
	if !bytes.Equal(pdfBytes, other.ok("GET", "/api/v1/assets/"+imported.Elements[3].AssetID, nil).Body.Bytes()) {
		t.Fatal("retained PDF changed")
	}
	other.ok("GET", "/api/v1/assets/"+imported.Elements[3].AssetID+"?bucket=1024", nil)
	if owner.request("GET", "/api/v1/pages/"+result.PageID+"/document", nil).Code != 404 {
		t.Fatal("import was not private")
	}
	appended := decoded[transfer.Result](t, importNative(other, raw, result.ID, other.csrf))
	if appended.PageCount != 2 || len(appended.Warnings) != 1 {
		t.Fatal("append or template name collision failed", appended)
	}
	entries := decoded[[]struct {
		ID string `json:"id"`
	}](t, other.ok("GET", "/api/v1/workspaces/"+result.ID+"/pages", nil))
	if len(entries) != 4 || entries[2].ID != appended.PageID {
		t.Fatal("append replaced/reordered existing pages")
	}
}

func TestNativeSelectionAndPermissions(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, raw := nativeSample(t, owner)
	reader, readerID := sharedUser(owner, "native-reader@example.com")
	path := "/api/v1/workspaces/" + workspaceID
	if reader.request("POST", path+"/export", transfer.Selection{}).Code != 404 {
		t.Fatal("private export leaked")
	}
	owner.ok("PUT", path+"/members", map[string]string{"userId": readerID, roleField: readAccessRole})
	pages := decoded[[]struct {
		ID string `json:"id"`
	}](t, reader.ok("GET", path+"/pages", nil))
	selected := reader.ok("POST", path+"/export", transfer.Selection{PageIDs: []string{pages[1].ID}})
	bundle, _, err := transfer.Read(selected.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Kind != nativePagesKind || len(bundle.Pages) != 1 || bundle.Pages[0].Name != "Page 2" || len(bundle.Assets) != 0 || len(bundle.Templates) != 0 {
		t.Fatal("selection included unrelated content")
	}
	if importNative(reader, raw, workspaceID, reader.csrf).Code != 403 {
		t.Fatal("reader appended pages")
	}
	if importNative(owner, raw, "", "").Code != 403 {
		t.Fatal("import bypassed CSRF")
	}
	if owner.request("POST", path+"/export", transfer.Selection{PageIDs: []string{"missing"}}).Code != 400 {
		t.Fatal("missing page accepted")
	}
	owner.ok("POST", path+"/export/snapshot", transfer.Selection{})
	owner.ok("DELETE", path+"/members/"+readerID, nil)
	if reader.request("POST", path+"/export/snapshot", transfer.Selection{}).Code != 404 {
		t.Fatal("revoked access exported")
	}
}

func TestNativeImportRollsBackAndRejectsMalformedFiles(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	_, raw := nativeSample(t, owner)
	before := decoded[[]struct {
		ID string `json:"id"`
	}](t, owner.ok("GET", "/api/v1/workspaces", nil))
	_, err := owner.db.ExecContext(t.Context(), `CREATE TRIGGER fail_native BEFORE INSERT ON pages WHEN NEW.name='Page 2' BEGIN SELECT RAISE(ABORT,'test rollback'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if importNative(owner, raw, "", owner.csrf).Code != 500 {
		t.Fatal("expected storage failure")
	}
	after := decoded[[]struct {
		ID string `json:"id"`
	}](t, owner.ok("GET", "/api/v1/workspaces", nil))
	if len(after) != len(before) {
		t.Fatal("failed import left partial workspace")
	}
	var assetCount int
	err = owner.db.QueryRowContext(t.Context(), "SELECT count(*) FROM assets").Scan(&assetCount)
	if err != nil || assetCount != 1 {
		t.Fatal("failed import left orphan media", assetCount, err)
	}

	if importNative(owner, []byte("not a ZIP"), "", owner.csrf).Code != 400 {
		t.Fatal("malformed file accepted")
	}
}

const nativePagesKind = "pages"

func TestButterflyExportRoundTripAndAccess(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, _ := nativeSample(t, owner)
	path := "/api/v1/workspaces/" + workspaceID + "/export"
	response := owner.ok("POST", path, transfer.Selection{Format: "butterfly"})
	if response.Header().Get("X-Mindflow-Conversion-Notes") == "" {
		t.Fatal("missing conversion notes")
	}
	archive, err := butterfly.Read(response.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.Pages) != 2 || len(archive.Images) != 1 || len(archive.Pages[0].Document.Elements) != 3 {
		t.Fatal("Butterfly export lost supported content")
	}
	imported := decoded[transfer.Result](t, importButterfly(owner, response.Body.Bytes(), owner.csrf))
	if imported.PageCount != 2 || imported.ElementCount != 6 {
		t.Fatal("Butterfly roundtrip", imported)
	}
	other, _ := sharedUser(owner, "butterfly-outsider@example.com")
	if other.request("POST", path, transfer.Selection{Format: "butterfly"}).Code != 404 {
		t.Fatal("export granted private access")
	}
	if owner.request("POST", path, transfer.Selection{Format: "unknown"}).Code != 400 {
		t.Fatal("unknown export format was accepted")
	}
}

func TestOneNoteNativeRoundTripAndPermissions(t *testing.T) {
	t.Parallel()
	if os.Getenv("MINDFLOW_ONENOTE_CODEC") == "" {
		t.Skip("Run make test to build the native OneNote codec")
	}
	owner := fixture(t)
	owner.setup()
	workspaceID, _ := nativeSample(t, owner)
	pages := decoded[[]struct {
		ID string `json:"id"`
	}](t, owner.ok("GET", "/api/v1/workspaces/"+workspaceID+"/pages", nil))
	source := decoded[struct {
		Document canvas.Document `json:"document"`
	}](t, owner.ok("GET", "/api/v1/pages/"+pages[0].ID+"/document", nil)).Document
	other, _ := sharedUser(owner, "onenote-recipient@example.com")
	path := "/api/v1/workspaces/" + workspaceID + "/export"
	for _, format := range []string{"one", "onepkg"} {
		if other.request("POST", path, transfer.Selection{Format: format}).Code != 404 {
			t.Fatal("native export bypassed private workspace permission")
		}
		exported := owner.ok("POST", path, transfer.Selection{Format: format})
		if len(exported.Body.Bytes()) < 100 {
			t.Fatal("empty native file")
		}
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		file, err := form.CreateFormFile("file", "notebook."+format)
		if err != nil {
			t.Fatal(err)
		}
		_, err = file.Write(exported.Body.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		err = form.Close()
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequestWithContext(t.Context(), "POST", "/api/v1/workspaces/import/onenote", &body)
		request.Header.Set("Content-Type", form.FormDataContentType())
		request.Header.Set("X-CSRF-Token", other.csrf)
		request.AddCookie(other.cookie)
		response := httptest.NewRecorder()
		other.a.ServeHTTP(response, request)
		imported := decoded[transfer.Result](t, response)
		if imported.PageCount != 2 || imported.ElementCount != 6 {
			t.Fatal("native pages/elements lost", imported)
		}
		document := decoded[struct {
			Document canvas.Document `json:"document"`
		}](t, other.ok("GET", "/api/v1/pages/"+imported.PageID+"/document", nil)).Document
		counts := map[string]int{}
		for _, element := range document.Elements {
			counts[element.Type]++
		}
		for _, axis := range []string{"x", "y"} {
			position := func(element canvas.Element) float64 {
				if axis == "x" {
					return element.X - element.Width/2
				}

				return element.Y - element.Height/2
			}
			originalDelta := position(source.Elements[1]) - position(source.Elements[0])
			restoredDelta := position(document.Elements[1]) - position(document.Elements[0])
			if math.Abs(originalDelta-restoredDelta) > 1 {
				t.Fatal("native transfer changed relative object position", axis, originalDelta, restoredDelta)
			}
		}
		if counts["stroke"] != 1 || counts["image"] != 1 || counts["text"] != 1 {
			t.Fatal("ink should remain editable and shapes should use individual image fallback", counts)
		}
	}
}

func TestOneNoteRejectsTooManyGeneratedImagesBeforeLaunchingCodec(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, _ := nativeSample(t, owner)
	pages := decoded[[]struct {
		ID string `json:"id"`
	}](t, owner.ok("GET", "/api/v1/workspaces/"+workspaceID+"/pages", nil))
	document := canvas.NewDocument()
	for index := range 1001 {
		document.Elements = append(document.Elements, canvas.Element{ID: fmt.Sprintf("shape-%d", index), Type: transferShapeType, Width: 2, Height: 2, Shape: transferRectangleShape, Fill: "#1b5e3b", Border: "#d9a441"})
	}
	lease := owner.acquire(pages[0].ID, "bounded-export", "acquire")
	owner.ok("PUT", "/api/v1/pages/"+pages[0].ID+"/document", canvas.SaveDocumentCommand{Epoch: lease.Epoch, ClientID: "bounded-export", MutationID: "bounded-save", Document: document})
	response := owner.request("POST", "/api/v1/workspaces/"+workspaceID+"/export", transfer.Selection{Format: "onepkg"})
	if response.Code != 400 || !strings.Contains(response.Body.String(), "Generated export media exceeds") {
		t.Fatal("generated media budget was not enforced before native conversion", response.Code, response.Body.String())
	}
}
