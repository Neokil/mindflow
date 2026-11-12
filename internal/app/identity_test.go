package app

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"mindflow/internal/domain"

	"mindflow/internal/store"
)

type testClient struct {
	a      *App
	db     *sql.DB
	cookie *http.Cookie
	csrf   string
	t      *testing.T
}

func fixture(t *testing.T) *testClient {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	a, err := New(db, Config{Dev: true, Bootstrap: "operator-secret"})
	if err != nil {
		t.Fatal(err)
	}
	return &testClient{a: a, db: db, t: t}
}
func (c *testClient) request(method, path string, body any) *httptest.ResponseRecorder {
	c.t.Helper()
	raw, _ := json.Marshal(body)
	r := httptest.NewRequest(method, path, strings.NewReader(string(raw)))
	r.Header.Set("Content-Type", "application/json")
	if c.cookie != nil {
		r.AddCookie(c.cookie)
		r.Header.Set("X-CSRF-Token", c.csrf)
	}
	w := httptest.NewRecorder()
	c.a.ServeHTTP(w, r)
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
	return map[string]any{"credential": "operator-secret", "email": "owner@example.com", "password": "a long owner password", "settings": Settings{PublicURL: "http://localhost:5173", SMTP: SMTPConfig{Host: "smtp.example.com", Port: 587, Mode: "starttls", From: "mindflow@example.com"}}}
}
func (c *testClient) setup() {
	c.t.Helper()
	c.ok("POST", "/api/v1/setup/complete", setupBody())
	c.login("owner@example.com", "a long owner password")
}
func (c *testClient) login(email, password string) {
	c.t.Helper()
	w := c.ok("POST", "/api/v1/auth/login", map[string]string{"email": email, "password": password})
	c.cookie = w.Result().Cookies()[0]
	var result struct {
		CSRF string `json:"csrf"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	c.csrf = result.CSRF
}
func (c *testClient) token(email, purpose string) string {
	c.t.Helper()
	var raw string
	if err := c.db.QueryRow("SELECT payload FROM jobs WHERE user_id=(SELECT id FROM users WHERE email=?) AND dedup_key=(SELECT id FROM users WHERE email=?)||':'||?", email, email, purpose).Scan(&raw); err != nil {
		c.t.Fatal(err)
	}
	var job struct{ Body string }
	json.Unmarshal([]byte(raw), &job)
	parts := strings.Split(job.Body, "/")
	return strings.Split(parts[len(parts)-1], "\n")[0]
}
func TestSetupAuthenticationAndLastAdmin(t *testing.T) {
	c := fixture(t)
	badSetup := setupBody()
	badSetup["credential"] = "wrong"
	if c.request("POST", "/api/v1/setup/complete", badSetup).Code != 403 {
		t.Fatal("setup accepted wrong credential")
	}
	c.setup()
	if c.request("POST", "/api/v1/setup/complete", setupBody()).Code != 409 {
		t.Fatal("setup repeated")
	}
	var uid string
	c.db.QueryRow("SELECT id FROM users WHERE email='owner@example.com'").Scan(&uid)
	if c.request("PATCH", "/api/v1/admin/users/"+uid, map[string]string{"status": "inactive"}).Code != 400 {
		t.Fatal("last admin disabled")
	}
	if c.request("DELETE", "/api/v1/admin/users/"+uid, map[string]bool{"confirm": true}).Code != 400 {
		t.Fatal("last admin deleted")
	}
	csrf := c.csrf
	c.csrf = ""
	if c.request("POST", "/api/v1/auth/logout", map[string]any{}).Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	c.csrf = csrf
	r := httptest.NewRequest("POST", "/api/v1/auth/logout", strings.NewReader("{}"))
	r.AddCookie(c.cookie)
	r.Header.Set("Origin", "https://evil.example")
	r.Header.Set("X-CSRF-Token", c.csrf)
	w := httptest.NewRecorder()
	c.a.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross origin accepted")
	}
	c.ok("POST", "/api/v1/auth/logout", map[string]any{})
	if c.request("GET", "/api/v1/auth/me", nil).Code != 401 {
		t.Fatal("logout left session active")
	}
}
func TestInvitationResetAndDeactivation(t *testing.T) {
	c := fixture(t)
	c.setup()
	c.ok("POST", "/api/v1/admin/users", map[string]string{"email": "friend@example.com", "role": "user"})
	old := c.token("friend@example.com", "invite")
	var uid string
	c.db.QueryRow("SELECT id FROM users WHERE email='friend@example.com'").Scan(&uid)
	c.ok("POST", "/api/v1/admin/users/"+uid+"/invite", map[string]any{})
	token := c.token("friend@example.com", "invite")
	if c.request("POST", "/api/v1/auth/accept-invite", map[string]string{"token": old, "password": "a long friend password"}).Code != 400 {
		t.Fatal("superseded invite accepted")
	}
	c.ok("POST", "/api/v1/auth/accept-invite", map[string]string{"token": token, "password": "a long friend password"})
	if c.request("POST", "/api/v1/auth/accept-invite", map[string]string{"token": token, "password": "a long friend password"}).Code != 400 {
		t.Fatal("token reused")
	}
	friend := &testClient{a: c.a, db: c.db, t: t}
	friend.login("friend@example.com", "a long friend password")
	if friend.request("GET", "/api/v1/admin/users", nil).Code != 403 {
		t.Fatal("regular user reached administration")
	}
	friend.ok("POST", "/api/v1/auth/forgot-password", map[string]string{"email": "friend@example.com"})
	reset := c.token("friend@example.com", "reset")
	friend.ok("POST", "/api/v1/auth/reset-password", map[string]string{"token": reset, "password": "a new friend password"})
	if friend.request("GET", "/api/v1/auth/me", nil).Code != 401 {
		t.Fatal("reset failed to revoke session")
	}
	friend.login("friend@example.com", "a new friend password")
	c.ok("PATCH", "/api/v1/admin/users/"+uid, map[string]string{"status": "inactive"})
	if friend.request("GET", "/api/v1/auth/me", nil).Code != 401 {
		t.Fatal("inactive user retained session")
	}
	c.ok("DELETE", "/api/v1/admin/users/"+uid, map[string]bool{"confirm": true})
	var n int
	c.db.QueryRow("SELECT count(*) FROM jobs WHERE user_id=?", uid).Scan(&n)
	if n != 0 {
		t.Fatal("user jobs survived deletion")
	}
}

func emptyDocument() domain.Document { return domain.NewDocument() }

func TestSMTPPasswordIsRedacted(t *testing.T) {
	c := fixture(t)
	c.setup()
	settings := setupBody()["settings"].(Settings)
	settings.SMTP.Password = "smtp-secret-never-returned"
	raw, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.db.Exec("UPDATE app_settings SET config=? WHERE id=1", string(raw)); err != nil {
		t.Fatal(err)
	}
	response := c.ok("GET", "/api/v1/admin/settings", nil)
	if strings.Contains(response.Body.String(), settings.SMTP.Password) {
		t.Fatal("SMTP password exposed")
	}
	if !strings.Contains(response.Body.String(), `"passwordConfigured":true`) {
		t.Fatal("missing configured-password indicator")
	}
}
