package authentication_test

import (
	"encoding/json"
	"mindflow/internal/authentication"
	"strings"
	"testing"
)

func TestAdministratorAndInvitationRules(t *testing.T) {
	t.Parallel()

	if authentication.CheckLastAdministrator("admin", "active", "user", "active", 1) == nil {
		t.Fatal("last administrator demoted")
	}
	if authentication.CheckLastAdministrator("admin", "active", "user", "active", 2) != nil {
		t.Fatal("valid demotion refused")
	}
	_, _, changeAccountErr := authentication.ChangeAccount("user", "invited", "", "active")
	if changeAccountErr == nil {
		t.Fatal("invitation bypassed")
	}
	role, status, err := authentication.ChangeAccount("user", "inactive", "admin", "active")
	if err != nil || role != "admin" || status != "active" {
		t.Fatal("valid account update refused")
	}
}

func TestCredentialsCannotBeSerializedIntoResponses(t *testing.T) {
	t.Parallel()

	for _, value := range []any{authentication.Principal{SessionID: "raw-session", CSRF: "private-csrf"}, authentication.LoginResult{CSRF: "public-csrf", Cookie: "raw-cookie"}} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"raw-session", "private-csrf", "raw-cookie"} {
			if strings.Contains(string(encoded), secret) {
				t.Fatalf("serialized credential %s", secret)
			}
		}
	}
}
