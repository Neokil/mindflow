package mindflow_test

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	"mindflow/internal/workspace"
	imageasset "mindflow/internal/workspace/image"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/preview"
	"mindflow/internal/workspace/sharing"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestImageOptimisationPreservesOriginalAndCachesOnlyStableRepresentations(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, pageID := owner.workspace()
	asset := owner.uploadImage()
	path := "/api/v1/assets/" + asset.ID
	original := owner.ok("GET", path, nil)
	if original.Header().Get("Cache-Control") != immutableCache {
		t.Fatal("original is not cached")
	}
	fallback := owner.ok("GET", path+"?bucket=101", nil)
	if fallback.Header().Get("Cache-Control") != noStoreCache {
		t.Fatal("fallback must not be cached as the future variant")
	}
	document := canvas.NewDocument()
	document.Elements = []canvas.Element{{ID: "small-image", Type: "image", AssetID: asset.ID, Width: 101, Height: 101}}
	lease := owner.acquire(pageID, "image-optimiser", "acquire")
	owner.ok("PUT", "/api/v1/pages/"+pageID+"/document", canvas.SaveDocumentCommand{ClientID: "image-optimiser", Epoch: lease.Epoch, MutationID: "optimise-save", Document: document})
	endpoint := "/api/v1/pages/" + pageID + "/optimise-images"
	reader, readerID := sharedUser(owner, "image-reader@example.com")
	owner.ok("PUT", "/api/v1/workspaces/"+workspaceID+"/members", sharing.Command{UserID: readerID, Role: readAccessRole})
	if reader.request("POST", endpoint, canvas.OptimiseImagesCommand{Scale: 1}).Code != 403 {
		t.Fatal("reader optimised another user's images")
	}
	result := decoded[canvas.OptimiseImagesResult](t, owner.ok("POST", endpoint, canvas.OptimiseImagesCommand{Scale: 1}))
	if result.Optimised != 1 || result.Images != 1 {
		t.Fatal("missing exact-size optimisation", result)
	}
	variant := owner.ok("GET", path+"?bucket=101", nil)
	config, _, err := image.DecodeConfig(bytes.NewReader(variant.Body.Bytes()))
	if err != nil || config.Width != 101 || config.Height != 101 || variant.Header().Get("Cache-Control") != immutableCache {
		t.Fatal("wrong variant", config, err, owner.ok("GET", path+"/meta", nil).Body.String())
	}
	if !bytes.Equal(original.Body.Bytes(), owner.ok("GET", path, nil).Body.Bytes()) {
		t.Fatal("optimisation modified the original")
	}
	again := decoded[canvas.OptimiseImagesResult](t, owner.ok("POST", endpoint, canvas.OptimiseImagesCommand{Scale: 1}))
	if again.Optimised != 0 {
		t.Fatal("identical variant regenerated")
	}
	metadata := decoded[imageasset.AssetInfo](t, owner.ok("GET", path+"/meta", nil))
	if len(metadata.Variants) == 0 {
		t.Fatal("optimised version not discoverable")
	}
}

const noStoreCache = "no-store"

const immutableCache = "private, max-age=31536000, immutable"

func storeThumbnail(client *testClient, workspaceID, pageID string, revision int, raw []byte) *httptest.ResponseRecorder {
	client.t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	err := form.WriteField("pageId", pageID)
	if err != nil {
		client.t.Fatal(err)
	}
	err = form.WriteField("revision", strconv.Itoa(revision))
	if err != nil {
		client.t.Fatal(err)
	}
	file, err := form.CreateFormFile("file", "preview.png")
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
	request := httptest.NewRequestWithContext(client.t.Context(), "POST", "/api/v1/workspaces/"+workspaceID+"/preview", &body)
	request.AddCookie(client.cookie)
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.Header.Set("X-CSRF-Token", client.csrf)
	response := httptest.NewRecorder()
	client.a.ServeHTTP(response, request)

	return response
}

func TestStoredWorkspacePreviewIsVersionedAndRejectsStaleOrUnauthorizedWrites(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	workspaceID, pageID := owner.workspace()
	var raw bytes.Buffer
	err := png.Encode(&raw, image.NewNRGBA(image.Rect(0, 0, 200, 100)))
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/workspaces/" + workspaceID
	state := decoded[preview.State](t, owner.ok("GET", path+"/preview", nil))
	if !state.Needed || !state.CanWrite {
		t.Fatal("missing preview not scheduled", state)
	}
	if storeThumbnail(owner, workspaceID, pageID, 1, raw.Bytes()).Code != 409 {
		t.Fatal("stale preview accepted")
	}
	saved := decoded[preview.State](t, storeThumbnail(owner, workspaceID, pageID, 0, raw.Bytes()))
	blobPath := path + "/previews/" + saved.PreviewID
	if owner.ok("GET", blobPath, nil).Header().Get("Cache-Control") != immutableCache {
		t.Fatal("stored preview not immutable")
	}
	list := decoded[[]workspace.Workspace](t, owner.ok("GET", "/api/v1/workspaces", nil))
	if list[0].PreviewID != saved.PreviewID || !list[0].PreviewEmpty {
		t.Fatal("stored preview absent from list", list)
	}
	reader, readerID := sharedUser(owner, "thumbnail-reader@example.com")
	if reader.request("GET", blobPath, nil).Code != 404 {
		t.Fatal("private preview disclosed")
	}
	owner.ok("PUT", path+"/members", sharing.Command{UserID: readerID, Role: readAccessRole})
	reader.ok("GET", blobPath, nil)
	if storeThumbnail(reader, workspaceID, pageID, 0, raw.Bytes()).Code != 403 {
		t.Fatal("reader replaced preview")
	}
	if storeThumbnail(owner, workspaceID, pageID, 0, []byte("bad PNG")).Code != 400 {
		t.Fatal("invalid preview accepted")
	}
	owner.ok("DELETE", path+"/members/"+readerID, nil)
	if reader.request("GET", blobPath, nil).Code != 404 {
		t.Fatal("revoked member can still fetch preview")
	}
}
