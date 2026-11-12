package authentication_test

import (
	"encoding/json"
	"mindflow/internal/authentication"
	"mindflow/internal/session"
	"strings"
	"testing"
)

func TestCredentialsCannotBeSerializedIntoResponses(t *testing.T) {
	t.Parallel()

	for _, value := range []any{session.Principal{SessionID: "raw-session", CSRF: "private-csrf"}, authentication.LoginResult{CSRF: "public-csrf", Cookie: "raw-cookie"}} {
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
