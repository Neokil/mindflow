package mindflow_test

import (
	"database/sql"
	"encoding/json"
	"mindflow/internal/email"
	"mindflow/internal/installation"
	"mindflow/internal/mindflow"
	"mindflow/internal/mindflow/storage"
	"mindflow/internal/workspace/page/canvas"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type testClient struct {
	clock  *atomic.Int64
	a      *mindflow.App
	db     *sql.DB
	cookie *http.Cookie
	csrf   string
	t      *testing.T
}

func fixture(t *testing.T) *testClient {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.sqlite")
	dbValue, err := storage.OpenData(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dbValue.Close() })
	clock := &atomic.Int64{}
	application, err := mindflow.New(t.Context(), dbValue, mindflow.Config{Dev: true, OneNoteCodec: os.Getenv("MINDFLOW_ONENOTE_CODEC"), Bootstrap: "operator-secret", Now: func() time.Time { return time.Now().Add(time.Duration(clock.Load()) * time.Millisecond) }})
	if err != nil {
		t.Fatal(err)
	}

	probe, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = probe.Close() })

	return &testClient{a: application, db: probe, t: t, clock: clock}
}

func (c *testClient) request(method, path string, body any) *httptest.ResponseRecorder {
	c.t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		c.t.Fatal(err)
	}
	request := httptest.NewRequestWithContext(c.t.Context(), method, path, strings.NewReader(string(raw)))
	request.Header.Set("Content-Type", "application/json")
	if c.cookie != nil {
		request.AddCookie(c.cookie)
		request.Header.Set("X-CSRF-Token", c.csrf)
	}
	w := httptest.NewRecorder()
	c.a.ServeHTTP(w, request)

	return w
}

func (c *testClient) ok(method, path string, body any) *httptest.ResponseRecorder {
	c.t.Helper()
	w := c.request(method, path, body)
	if w.Code != 200 {
		c.t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
	}

	return w
}

func setupBody() map[string]any {
	return map[string]any{"credential": "operator-secret", emailField: "owner@example.com", passwordField: "a long owner password", "settings": installation.Settings{PublicURL: "http://localhost:5173", SMTP: email.Config{Host: "smtp.example.com", Port: 587, Mode: "starttls", From: "mindflow@example.com"}}} //nolint:gosec // G101: deterministic credentials for a disposable test database, never real accounts.
}

func (c *testClient) setup() {
	c.t.Helper()
	c.ok("POST", "/api/v1/setup/complete", setupBody())
	c.login("owner@example.com", "a long owner password")
}

func (c *testClient) login(email, password string) {
	c.t.Helper()
	w := c.ok("POST", "/api/v1/auth/login", map[string]string{emailField: email, passwordField: password})
	c.cookie = w.Result().Cookies()[0]
	var result struct {
		CSRF string `json:"csrf"`
	}
	checkErr80 := json.Unmarshal(w.Body.Bytes(), &result)
	if checkErr80 != nil {
		c.t.Fatal(checkErr80)
	}
	c.csrf = result.CSRF
}

func (c *testClient) token(email, purpose string) string {
	c.t.Helper()
	var raw string
	err := c.db.QueryRowContext(c.t.Context(), "SELECT payload FROM mail_jobs WHERE user_id=(SELECT id FROM users WHERE email=?) AND dedup_key=(SELECT id FROM users WHERE email=?)||':'||?", email, email, purpose).Scan(&raw)
	if err != nil {
		c.t.Fatal(err)
	}
	var job struct {
		Body string `json:"body"`
	}
	checkErr92 := json.Unmarshal([]byte(raw), &job)
	if checkErr92 != nil {
		c.t.Fatal(checkErr92)
	}
	parts := strings.Split(job.Body, "/")

	return strings.Split(parts[len(parts)-1], "\n")[0]
}

func TestSetupAuthenticationAndLastAdmin(t *testing.T) {
	t.Parallel()

	client := fixture(t)
	badSetup := setupBody()
	badSetup["credential"] = "wrong"
	if client.request("POST", "/api/v1/setup/complete", badSetup).Code != 403 {
		t.Fatal("setup accepted wrong credential")
	}
	client.setup()
	if client.request("POST", "/api/v1/setup/complete", setupBody()).Code != 409 {
		t.Fatal("setup repeated")
	}
	var uid string
	checkErr112 := client.db.QueryRowContext(t.Context(), "SELECT id FROM users WHERE email='owner@example.com'").Scan(&uid)
	if checkErr112 != nil {
		t.Fatal(checkErr112)
	}
	if client.request("PATCH", "/api/v1/admin/users/"+uid, map[string]string{"status": "inactive"}).Code != 400 {
		t.Fatal("last admin disabled")
	}
	if client.request("DELETE", "/api/v1/admin/users/"+uid, map[string]bool{confirmField: true}).Code != 400 {
		t.Fatal("last admin deleted")
	}
	csrf := client.csrf
	client.csrf = ""
	if client.request("POST", "/api/v1/auth/logout", map[string]any{}).Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	client.csrf = csrf
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/logout", strings.NewReader("{}"))
	r.AddCookie(client.cookie)
	r.Header.Set("Origin", "https://evil.example")
	r.Header.Set("X-CSRF-Token", client.csrf)
	w := httptest.NewRecorder()
	client.a.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross origin accepted")
	}
	client.ok("POST", "/api/v1/auth/logout", map[string]any{})
	if client.request("GET", "/api/v1/auth/me", nil).Code != 401 {
		t.Fatal("logout left session active")
	}
}

func TestInvitationResetAndDeactivation(t *testing.T) {
	t.Parallel()

	client := fixture(t)
	client.setup()
	client.ok("POST", "/api/v1/admin/users", map[string]string{emailField: "friend@example.com", roleField: userRole})
	old := client.token("friend@example.com", "invite")
	var uid string
	checkErr148 := client.db.QueryRowContext(t.Context(), "SELECT id FROM users WHERE email='friend@example.com'").Scan(&uid)
	if checkErr148 != nil {
		t.Fatal(checkErr148)
	}
	client.ok("POST", "/api/v1/admin/users/"+uid+"/invite", map[string]any{})
	token := client.token("friend@example.com", "invite")
	if client.request("POST", "/api/v1/auth/accept-invite", map[string]string{tokenField: old, passwordField: friendPassword}).Code != 400 {
		t.Fatal("superseded invite accepted")
	}
	client.ok("POST", "/api/v1/auth/accept-invite", map[string]string{tokenField: token, passwordField: friendPassword})
	if client.request("POST", "/api/v1/auth/accept-invite", map[string]string{tokenField: token, passwordField: friendPassword}).Code != 400 {
		t.Fatal("token reused")
	}
	friend := &testClient{a: client.a, db: client.db, t: t, clock: client.clock}
	friend.login("friend@example.com", friendPassword)
	if friend.request("GET", "/api/v1/admin/users", nil).Code != 403 {
		t.Fatal("regular user reached administration")
	}
	friend.ok("POST", "/api/v1/auth/forgot-password", map[string]string{emailField: "friend@example.com"})
	reset := client.token("friend@example.com", "reset")
	friend.ok("POST", "/api/v1/auth/reset-password", map[string]string{tokenField: reset, passwordField: "a new friend password"})
	if friend.request("GET", "/api/v1/auth/me", nil).Code != 401 {
		t.Fatal("reset failed to revoke session")
	}
	friend.login("friend@example.com", "a new friend password")
	client.ok("PATCH", "/api/v1/admin/users/"+uid, map[string]string{"status": "inactive"})
	if friend.request("GET", "/api/v1/auth/me", nil).Code != 401 {
		t.Fatal("inactive user retained session")
	}
	client.ok("DELETE", "/api/v1/admin/users/"+uid, map[string]bool{confirmField: true})
	var n int
	checkErr176 := client.db.QueryRowContext(t.Context(), "SELECT count(*) FROM mail_jobs WHERE user_id=?", uid).Scan(&n)
	if checkErr176 != nil {
		t.Fatal(checkErr176)
	}
	if n != 0 {
		t.Fatal("user mail_jobs survived deletion")
	}
}

func emptyDocument() canvas.Document { return canvas.NewDocument() }

func TestSMTPPasswordIsRedacted(t *testing.T) {
	t.Parallel()

	client := fixture(t)
	client.setup()
	settings, ok := setupBody()["settings"].(installation.Settings)
	if !ok {
		t.Fatal("setup fixture does not contain settings")
	}
	settings.SMTP.Password = "smtp-secret-never-returned"
	raw, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.db.ExecContext(t.Context(), "UPDATE app_settings SET config=? WHERE id=1", string(raw))
	if err != nil {
		t.Fatal(err)
	}
	response := client.ok("GET", "/api/v1/admin/settings", nil)
	if strings.Contains(response.Body.String(), settings.SMTP.Password) {
		t.Fatal("SMTP password exposed")
	}
	if !strings.Contains(response.Body.String(), `"passwordConfigured":true`) {
		t.Fatal("missing configured-password indicator")
	}
}

const (
	emailField     = "email"
	passwordField  = "password"
	confirmField   = "confirm"
	friendPassword = "a long friend password"
)

const tokenField = "token"

func TestAccountDeletionCleansIndependentDomainsAtomically(t *testing.T) {
	t.Parallel()
	client := fixture(t)
	client.setup()
	client.ok("POST", "/api/v1/admin/users", map[string]string{emailField: "delete@example.com", roleField: userRole})
	var userID string
	err := client.db.QueryRowContext(t.Context(), "SELECT id FROM users WHERE email='delete@example.com'").Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.db.ExecContext(t.Context(), `INSERT INTO workspaces(id,owner_id,name,created_at,updated_at) VALUES('deleted-workspace',?,'Delete',1,1);
 INSERT INTO pages(id,workspace_id,name,position,document,updated_at) VALUES('deleted-page','deleted-workspace','Delete',0,'{}',1);
 INSERT INTO assets(id,owner_id,original,mime,width,height,digest,created_at) VALUES('deleted-asset',?,x'00','image/png',1,1,'digest',1);
 INSERT INTO asset_variants(asset_id,bucket,mime,data,width,height) VALUES('deleted-asset',1,'image/png',x'00',1,1);
 INSERT INTO page_assets(page_id,asset_id) VALUES('deleted-page','deleted-asset');
 INSERT INTO page_leases(page_id,session_id) VALUES('deleted-page','opaque-session');
 INSERT INTO save_receipts(page_id,client_id,mutation_id,digest,revision,expires_at) VALUES('deleted-page','client','mutation','digest',1,9999999999);
 INSERT INTO image_jobs(id,user_id,asset_id,payload,run_at) VALUES('deleted-preview',?,'deleted-asset','{}',1);
 CREATE TRIGGER fail_mail_cleanup BEFORE DELETE ON mail_jobs BEGIN SELECT RAISE(ABORT,'test cleanup failure'); END;`, userID, userID, userID)
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "/api/v1/admin/users/" + userID
	if client.request("DELETE", endpoint, map[string]bool{confirmField: true}).Code != 500 {
		t.Fatal("cleanup failure was not reported")
	}
	var count int
	err = client.db.QueryRowContext(t.Context(), "SELECT count(*) FROM workspaces WHERE owner_id=?", userID).Scan(&count)
	if err != nil || count != 1 {
		t.Fatalf("failed deletion did not roll back workspace cleanup: %d, %v", count, err)
	}
	_, err = client.db.ExecContext(t.Context(), "DROP TRIGGER fail_mail_cleanup")
	if err != nil {
		t.Fatal(err)
	}
	client.ok("DELETE", endpoint, map[string]bool{confirmField: true})
	for _, table := range []string{"workspaces", nativePagesKind, "assets", "asset_variants", "page_assets", "page_leases", "save_receipts", "image_jobs", mailJobsTable, "account_tokens"} {
		query := "SELECT count(*) FROM " + table
		var arguments []any
		if table == mailJobsTable {
			query += " WHERE user_id=?"
			arguments = []any{userID}
		}
		err = client.db.QueryRowContext(t.Context(), query, arguments...).Scan(&count)
		if err != nil || count != 0 {
			t.Errorf("%s retained deleted user data: %d, %v", table, count, err)
		}
	}
	client.ok("GET", "/api/v1/auth/me", nil)
}

const roleField = "role"

const mailJobsTable = "mail_jobs"
