package mindflow_test

import (
	"encoding/json"
	"mindflow/internal/account/invitation"
	"strings"
	"testing"
)

func TestWelcomeQueuedExactlyOnceOnActivation(t *testing.T) {
	t.Parallel()
	owner := fixture(t)
	owner.setup()
	var raw string
	err := owner.db.QueryRowContext(t.Context(), "SELECT payload FROM mail_jobs WHERE dedup_key LIKE '%:welcome'").Scan(&raw)
	if err != nil {
		t.Fatal("administrator welcome missing", err)
	}
	friendResponse := owner.ok("POST", "/api/v1/admin/users", invitation.InviteCommand{Email: "welcome@example.com", Role: "user"})
	var created struct {
		ID string `json:"id"`
	}
	err = json.Unmarshal(friendResponse.Body.Bytes(), &created)
	if err != nil {
		t.Fatal(err)
	}
	token := owner.token("welcome@example.com", "invite")
	command := invitation.RedeemCommand{Token: token, Password: "welcome passphrase", DisplayName: "Simon <Friend>"}
	owner.ok("POST", "/api/v1/auth/accept-invite", command)
	err = owner.db.QueryRowContext(t.Context(), "SELECT payload FROM mail_jobs WHERE dedup_key=?", created.ID+":welcome").Scan(&raw)
	if err != nil {
		t.Fatal(err)
	}
	var message struct {
		Subject   string `json:"subject"`
		Body      string `json:"body"`
		TokenHash string `json:"tokenHash"`
	}
	err = json.Unmarshal([]byte(raw), &message)
	if err != nil {
		t.Fatal(err)
	}
	if message.Subject != "Welcome Simon <Friend>!" || message.TokenHash != "" {
		t.Fatal("incorrect welcome", message)
	}
	for _, link := range []string{"https://github.com/Neokil/mindflow/releases", "https://github.com/Neokil/mindflow/issues", "Open mindflow", "View patchnotes on GitHub", "Report an error or request a feature on GitHub"} {
		if !strings.Contains(message.Body, link) {
			t.Fatal("welcome link missing", link)
		}
	}
	if owner.request("POST", "/api/v1/auth/accept-invite", command).Code != 400 {
		t.Fatal("invitation redeemed twice")
	}
	var count int
	err = owner.db.QueryRowContext(t.Context(), "SELECT count(*) FROM mail_jobs WHERE dedup_key=?", created.ID+":welcome").Scan(&count)
	if err != nil || count != 1 {
		t.Fatal("duplicate welcome", count, err)
	}
}
