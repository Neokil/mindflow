package mindflow_test

import (
	"context"
	"encoding/json"
	accountstorage "mindflow/internal/account/storage"
	"mindflow/internal/account/updates"
	"mindflow/internal/email"
	emailstorage "mindflow/internal/email/storage"
	"mindflow/internal/installation"
	installationstorage "mindflow/internal/installation/storage"
	"mindflow/internal/release"
	releasestorage "mindflow/internal/release/storage"
	"strings"
	"testing"
	"time"
)

func TestReleaseConsentUpgradeDeduplicationAndUnsubscribe(t *testing.T) {
	t.Parallel()
	client := fixture(t)
	client.setup()
	result := client.ok("GET", "/api/v1/auth/product-updates", nil)
	if !strings.Contains(result.Body.String(), `"enabled":false`) {
		t.Fatal("updates must default off")
	}
	client.ok("PATCH", "/api/v1/auth/product-updates", map[string]bool{"enabled": true})
	client.db.SetMaxOpenConns(1)
	configuration := installation.NewService(installationstorage.NewRepository(client.db), nil, installation.Config{Dev: true}, nil)
	subscriptions := updates.NewService(accountstorage.NewRepository(client.db), email.NewOutbox(emailstorage.NewRepository(client.db)), configuration)
	releases := release.NewService(releasestorage.NewRepository(client.db), subscriptions)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	observe := func(version string) {
		t.Helper()
		err := releases.Observe(ctx, version)
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
		Body   string               `json:"body"`
		Update *email.ProductUpdate `json:"update"`
	}
	err = json.Unmarshal([]byte(raw), &message)
	if err != nil {
		t.Fatal(err)
	}
	if message.Update == nil || message.Update.Version != "v1.1.0" || message.Update.ReleaseURL != release.NotesURL("v1.1.0") {
		t.Fatal("release layout content was not queued", message.Update)
	}
	_, token, found := strings.Cut(message.Body, "unsubscribe?token=")
	if !found {
		t.Fatal("missing unsubscribe")
	}
	observe("v1.2.0")
	anonymous := &testClient{a: client.a, t: t}
	anonymous.ok("GET", "/api/v1/auth/product-updates/unsubscribe?token="+token, nil)
	anonymous.ok("GET", "/api/v1/auth/product-updates/unsubscribe?token="+token, nil)
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

func TestSignupConsentIsExplicitAndSeparateFromPasswordReset(t *testing.T) {
	t.Parallel()
	client := fixture(t)
	command := setupBody()
	command["productUpdates"] = true
	client.ok("POST", "/api/v1/setup/complete", command)
	client.login("owner@example.com", "a long owner password")
	response := client.ok("GET", "/api/v1/auth/product-updates", nil)
	if !strings.Contains(response.Body.String(), `"enabled":true`) {
		t.Fatal("setup consent was not saved")
	}
	client.ok("POST", "/api/v1/admin/users", map[string]string{emailField: "subscriber@example.com", roleField: userRole})
	token := client.token("subscriber@example.com", "invite")
	client.ok("POST", "/api/v1/auth/accept-invite", map[string]any{tokenField: token, passwordField: friendPassword, "productUpdates": true})
	client.login("subscriber@example.com", friendPassword)
	response = client.ok("GET", "/api/v1/auth/product-updates", nil)
	if !strings.Contains(response.Body.String(), `"enabled":true`) {
		t.Fatal("invitation consent was not saved")
	}
	client.ok("POST", "/api/v1/auth/forgot-password", map[string]string{emailField: "subscriber@example.com"})
	resetToken := client.token("subscriber@example.com", "reset")
	client.ok("POST", "/api/v1/auth/reset-password", map[string]string{tokenField: resetToken, passwordField: friendPassword})
	client.login("subscriber@example.com", friendPassword)
	response = client.ok("GET", "/api/v1/auth/product-updates", nil)
	if !strings.Contains(response.Body.String(), `"enabled":true`) {
		t.Fatal("password reset changed product-update consent")
	}
}
