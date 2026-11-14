package mindflow_test

import (
	"context"
	"encoding/json"
	accountstorage "mindflow/internal/account/storage"
	"mindflow/internal/account/updates"
	"mindflow/internal/email"
	emailstorage "mindflow/internal/email/storage"
	"mindflow/internal/release"
	releasestorage "mindflow/internal/release/storage"
	"strings"
	"testing"
)

type updateAddress struct{}

func (updateAddress) PublicAddress(context.Context) (string, error) {
	return "http://localhost:5173", nil
}

func TestReleaseConsentUpgradeDeduplicationAndUnsubscribe(t *testing.T) {
	t.Parallel()
	client := fixture(t)
	client.setup()
	result := client.ok("GET", "/api/v1/auth/product-updates", nil)
	if !strings.Contains(result.Body.String(), `"enabled":false`) {
		t.Fatal("updates must default off")
	}
	client.ok("PATCH", "/api/v1/auth/product-updates", map[string]bool{"enabled": true})
	subscriptions := updates.NewService(accountstorage.NewRepository(client.db), email.NewOutbox(emailstorage.NewRepository(client.db)), updateAddress{})
	releases := release.NewService(releasestorage.NewRepository(client.db), subscriptions)
	observe := func(version string) {
		t.Helper()
		err := releases.Observe(t.Context(), version)
		if err != nil {
			t.Fatal(err)
		}
	}
	observe("v1.0.0")
	observe("v1.0.0")
	observe("dev")
	observe("v1.1.0")
	observe("v1.1.0")
	observe("v1.0.0")
	observe("v1.1.0")
	var count int
	err := client.db.QueryRowContext(t.Context(), "SELECT count(*) FROM mail_jobs WHERE dedup_key LIKE '%:release-%'").Scan(&count)
	if err != nil || count != 1 {
		t.Fatal("upgrade announcement duplicated", count, err)
	}
	var raw string
	err = client.db.QueryRowContext(t.Context(), "SELECT payload FROM mail_jobs WHERE dedup_key LIKE '%:release-%'").Scan(&raw)
	if err != nil {
		t.Fatal(err)
	}
	var message struct {
		Body string `json:"body"`
	}
	err = json.Unmarshal([]byte(raw), &message)
	if err != nil {
		t.Fatal(err)
	}
	_, token, found := strings.Cut(message.Body, "unsubscribe?token=")
	if !found {
		t.Fatal("missing unsubscribe")
	}
	observe("v1.2.0")
	client.ok("GET", "/api/v1/auth/product-updates/unsubscribe?token="+token, nil)
	client.ok("GET", "/api/v1/auth/product-updates/unsubscribe?token="+token, nil)
	result = client.ok("GET", "/api/v1/auth/product-updates", nil)
	if !strings.Contains(result.Body.String(), `"enabled":false`) {
		t.Fatal("unsubscribe did not withdraw consent")
	}
	observe("v1.3.0")
	err = client.db.QueryRowContext(t.Context(), "SELECT count(*) FROM mail_jobs WHERE dedup_key LIKE '%:release-%'").Scan(&count)
	if err != nil || count != 0 {
		t.Fatal("release mail remained after unsubscribe", count, err)
	}
}
