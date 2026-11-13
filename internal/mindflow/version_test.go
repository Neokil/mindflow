package mindflow_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBrowserDocumentAndPublicVersionIdentifySameBuild(t *testing.T) {
	t.Parallel()
	client := fixture(t)
	response := client.ok("GET", "/api/v1/version", nil)
	var value struct {
		Version string `json:"version"`
	}
	err := json.Unmarshal(response.Body.Bytes(), &value)
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Version) != 64 || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("version is missing or cacheable")
	}
	document := client.ok("GET", "/", nil)
	if !strings.Contains(document.Body.String(), `name="mindflow-version" content="`+value.Version+`"`) {
		t.Fatal("browser document does not identify its build")
	}
	if document.Header().Get("Cache-Control") != "no-cache" {
		t.Fatal("browser document is cacheable without revalidation")
	}
}
