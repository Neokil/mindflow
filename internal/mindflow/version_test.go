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
	if len(value.Version) != 64 || response.Header().Get("Cache-Control") != noStoreCache {
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

func TestPublicVersionLinksOnlyCanonicalReleaseTags(t *testing.T) {
	t.Parallel()
	client := fixture(t)
	for _, label := range []string{releaseVersionForTest, "v1.3.0-rc.1", "dev", "v01.2.0", "v1.2.0/elsewhere"} {
		client.a.Config.Release = label
		response := client.ok("GET", "/api/v1/version", nil)
		var result struct {
			Release         string `json:"release"`
			ReleaseNotesURL string `json:"releaseNotesURL"`
		}
		err := json.Unmarshal(response.Body.Bytes(), &result)
		if err != nil {
			t.Fatal(err)
		}
		expected := "unreleased"
		notes := "https://github.com/Neokil/mindflow/releases"
		if label == releaseVersionForTest || label == "v1.3.0-rc.1" {
			expected = label
			notes += "/tag/" + label
		}
		if result.Release != expected || result.ReleaseNotesURL != notes {
			t.Fatal("incorrect release metadata", result)
		}
	}
}

const releaseVersionForTest = "v1.2.0"
