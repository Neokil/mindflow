package mindflow_test

import (
	"encoding/json"
	"mindflow/internal/account"
	"net/http"
	"testing"
)

const (
	profileStatusField = "status"
	profileNameField   = "displayName"
	profileTestName    = "Ada Lovelace"
)

func TestProfilesAndPrivateAccountDirectory(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	if owner.request("GET", "/api/v1/accounts/directory", nil).Code != http.StatusUnauthorized {
		t.Fatal("anonymous account directory exposed")
	}
	owner.setup()
	profile := owner.ok("PATCH", "/api/v1/auth/profile", map[string]string{profileNameField: "  " + profileTestName + "  "})
	var updated account.User
	err := json.Unmarshal(profile.Body.Bytes(), &updated)
	if err != nil {
		t.Fatal(err)
	}
	if updated.DisplayName != profileTestName {
		t.Fatal("profile update did not normalize the name")
	}
	reader, readerID := sharedUser(owner, "reader@example.com")
	_, secondID := sharedUser(owner, "other@example.com")
	reader.ok("PATCH", "/api/v1/auth/profile", map[string]string{profileNameField: profileTestName})
	owner.ok("POST", "/api/v1/admin/users", map[string]string{emailField: "pending@example.com"})
	_, inactiveID := sharedUser(owner, "inactive@example.com")
	owner.ok("PATCH", "/api/v1/admin/users/"+inactiveID, map[string]string{profileStatusField: account.StatusInactive})
	directory := reader.ok("GET", "/api/v1/accounts/directory", nil)
	var entries []map[string]string
	err = json.Unmarshal(directory.Body.Bytes(), &entries)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("directory includes inactive or invited users: %s", directory.Body.String())
	}
	found := map[string]bool{}
	names := 0
	for _, entry := range entries {
		if len(entry) != 3 || entry["id"] == "" || entry[profileNameField] == "" {
			t.Fatal("invalid public directory entry")
		}
		for _, key := range []string{"email", "role", profileStatusField, "passwordHash"} {
			if _, exists := entry[key]; exists {
				t.Fatal("private account metadata exposed")
			}
		}
		if entry[profileNameField] == profileTestName {
			names++
		}
		found[entry["id"]] = true
	}
	if names != 2 || !found[readerID] || !found[secondID] {
		t.Fatal("duplicate display names did not retain independent account IDs")
	}
	var current struct {
		User account.User `json:"user"`
	}
	err = json.Unmarshal(owner.ok("GET", "/api/v1/auth/me", nil).Body.Bytes(), &current)
	if err != nil {
		t.Fatal(err)
	}
	if current.User.DisplayName != profileTestName || current.User.ID != updated.ID {
		t.Fatal("profile name not retained in authentication")
	}
	if owner.request("PATCH", "/api/v1/auth/profile", map[string]string{profileNameField: "\n"}).Code != http.StatusBadRequest {
		t.Fatal("invalid profile name accepted")
	}
	csrf := owner.csrf
	owner.csrf = ""
	if owner.request("PATCH", "/api/v1/auth/profile", map[string]string{profileNameField: "Blocked"}).Code != http.StatusForbidden {
		t.Fatal("profile changed without CSRF")
	}
	owner.csrf = csrf
}
