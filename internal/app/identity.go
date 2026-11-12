package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Mode     string `json:"mode"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	From     string `json:"from"`
	Name     string `json:"name"`
}
type Settings struct {
	PublicURL string     `json:"publicURL"`
	SMTP      SMTPConfig `json:"smtp"`
}
type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Appearance string `json:"appearance"`
}
type principal struct {
	User
	SessionID string
	CSRF      string
}
type principalKey struct{}
type apiError struct {
	Status        int
	Code, Message string
}

func (e *apiError) Error() string                { return e.Message }
func problem(status int, code, msg string) error { return &apiError{status, code, msg} }
func bad(msg string) error                       { return problem(400, "validation_failed", msg) }
func notFound() error                            { return problem(404, "not_found", "This item is unavailable.") }
func randomID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func digest(s string) string        { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func now() int64                    { return time.Now().Unix() }
func who(r *http.Request) principal { p, _ := r.Context().Value(principalKey{}).(principal); return p }
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return problem(415, "content_type", "Expected application/json.")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 17<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return bad("Invalid request body.")
	}
	if d.Decode(new(any)) != io.EOF {
		return bad("Expected one JSON value.")
	}
	return nil
}
func (a *App) endpoint(pattern string, auth, admin bool, fn func(http.ResponseWriter, *http.Request) (any, error)) {
	a.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var p principal
		if auth {
			var err error
			p, err = a.authenticate(r)
			if err != nil {
				a.fail(w, r, err)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), principalKey{}, p))
			if admin && p.Role != "admin" {
				a.fail(w, r, problem(403, "forbidden", "Administrator access required."))
				return
			}
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			if err := a.checkOrigin(r); err != nil {
				a.fail(w, r, err)
				return
			}
			if auth && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(p.CSRF)) != 1 {
				a.fail(w, r, problem(403, "csrf", "Refresh the page and try again."))
				return
			}
		}
		result, err := fn(w, r)
		if err != nil {
			a.fail(w, r, err)
			return
		}
		if result != nil {
			writeJSON(w, 200, result)
		}
	})
}
func (a *App) fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apiError
	if !errors.As(err, &ae) {
		if errors.Is(err, sql.ErrNoRows) {
			ae = &apiError{404, "not_found", "This item is unavailable."}
		} else {
			ae = &apiError{500, "server_error", "The operation could not be completed. Please retry."}
			a.Log.Error("request failed", "request", r.Header.Get("X-Request-ID"), "error", err)
		}
	}
	writeJSON(w, ae.Status, map[string]any{"code": ae.Code, "message": ae.Message, "requestID": r.Header.Get("X-Request-ID")})
}
func (a *App) settings() (Settings, error) {
	var raw string
	err := a.DB.QueryRow("SELECT config FROM app_settings WHERE id=1").Scan(&raw)
	var s Settings
	if err == nil {
		err = json.Unmarshal([]byte(raw), &s)
	}
	return s, err
}
func (a *App) checkOrigin(r *http.Request) error {
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return problem(403, "origin", "Cross-site requests are not allowed.")
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return nil
	}
	s, err := a.settings()
	if err != nil {
		return err
	}
	if origin == s.PublicURL {
		return nil
	}
	if a.Config.Dev && (origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" || origin == "http://"+r.Host) {
		return nil
	}
	// Before setup, the browser must be on the same public host.
	var initialized int
	if err = a.DB.QueryRow("SELECT initialized FROM app_settings WHERE id=1").Scan(&initialized); err != nil {
		return err
	}
	u, e := url.Parse(origin)
	if initialized == 0 && e == nil && u.Host == r.Host && u.Scheme == "https" {
		return nil
	}
	return problem(403, "origin", "Request origin does not match the application URL.")
}
func (a *App) authenticate(r *http.Request) (principal, error) {
	var p principal
	c, err := r.Cookie("mindflow_session")
	if err != nil {
		return p, problem(401, "session_expired", "Please sign in.")
	}
	parts := strings.SplitN(c.Value, ".", 2)
	if len(parts) != 2 {
		return p, problem(401, "session_expired", "Please sign in.")
	}
	var hash string
	err = a.DB.QueryRow(`SELECT u.id,u.email,u.role,u.status,u.appearance,s.id,s.csrf,s.verifier_hash FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.id=? AND s.expires_at>? AND u.status='active'`, parts[0], now()).Scan(&p.ID, &p.Email, &p.Role, &p.Status, &p.Appearance, &p.SessionID, &p.CSRF, &hash)
	if err != nil || subtle.ConstantTimeCompare([]byte(hash), []byte(digest(parts[1]))) != 1 {
		return p, problem(401, "session_expired", "Please sign in.")
	}
	return p, nil
}
func hashPassword(password string) (string, error) {
	if utf8.RuneCountInString(password) < 15 || utf8.RuneCountInString(password) > 128 {
		return "", bad("Use a password between 15 and 128 characters.")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	h := argon2.IDKey([]byte(password), salt, 2, 19*1024, 1, 32)
	return fmt.Sprintf("$argon2id$v=19$m=19456,t=2,p=1$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(h)), nil
}
func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[3] != "m=19456,t=2,p=1" || len(password) > 1024 {
		return false
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[4])
	if e != nil {
		return false
	}
	expected, e := base64.RawStdEncoding.DecodeString(parts[5])
	if e != nil {
		return false
	}
	h := argon2.IDKey([]byte(password), salt, 2, 19*1024, 1, 32)
	return subtle.ConstantTimeCompare(h, expected) == 1
}
func normalizeEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s || len(s) > 254 {
		return "", bad("Enter a valid email address.")
	}
	return s, nil
}
func validSettings(s Settings, dev bool) error {
	u, err := url.Parse(s.PublicURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && !(dev && u.Scheme == "http")) {
		return bad("Use the public HTTPS application URL without a path.")
	}
	if s.SMTP.Host == "" || strings.ContainsAny(s.SMTP.Host, "\r\n") || s.SMTP.Port < 1 || s.SMTP.Port > 65535 {
		return bad("Enter the SMTP host and port.")
	}
	if s.SMTP.Mode != "starttls" && s.SMTP.Mode != "tls" {
		return bad("Choose STARTTLS or TLS for SMTP.")
	}
	if _, err = normalizeEmail(s.SMTP.From); err != nil {
		return bad("Enter a valid sender email address.")
	}
	if strings.ContainsAny(s.SMTP.Name, "\r\n") {
		return bad("Invalid sender name.")
	}
	return nil
}
func (a *App) limited(r *http.Request, key string) bool {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host == "" {
		host = r.RemoteAddr
	}
	key += ":" + host
	a.limitMu.Lock()
	defer a.limitMu.Unlock()
	v := a.limits[key]
	if time.Since(v.Start) > time.Minute {
		v = rateWindow{Start: time.Now()}
	}
	v.Count++
	a.limits[key] = v
	if len(a.limits) > 10000 {
		for k, v := range a.limits {
			if time.Since(v.Start) > time.Minute {
				delete(a.limits, k)
			}
		}
	}
	return v.Count > 20
}

type rateWindow struct {
	Start time.Time
	Count int
}

func (a *App) identityRoutes() {
	a.endpoint("GET /api/v1/setup/status", false, false, func(w http.ResponseWriter, r *http.Request) (any, error) {
		var n int
		err := a.DB.QueryRow("SELECT initialized FROM app_settings WHERE id=1").Scan(&n)
		return map[string]bool{"initialized": n == 1}, err
	})
	a.endpoint("POST /api/v1/setup/complete", false, false, a.setup)
	a.endpoint("POST /api/v1/setup/test-email", false, false, a.setupTestEmail)
	a.endpoint("POST /api/v1/auth/login", false, false, a.login)
	a.endpoint("POST /api/v1/auth/logout", true, false, func(w http.ResponseWriter, r *http.Request) (any, error) {
		_, err := a.DB.Exec("DELETE FROM sessions WHERE id=?", who(r).SessionID)
		http.SetCookie(w, &http.Cookie{Name: "mindflow_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: !a.Config.Dev, SameSite: http.SameSiteLaxMode})
		return map[string]bool{"ok": true}, err
	})
	a.endpoint("GET /api/v1/auth/me", true, false, func(w http.ResponseWriter, r *http.Request) (any, error) {
		p := who(r)
		return map[string]any{"user": p.User, "csrf": p.CSRF}, nil
	})
	a.endpoint("PATCH /api/v1/auth/preferences", true, false, func(w http.ResponseWriter, r *http.Request) (any, error) {
		var v struct {
			Appearance string `json:"appearance"`
		}
		if err := decode(w, r, &v); err != nil {
			return nil, err
		}
		if v.Appearance != "system" && v.Appearance != "light" && v.Appearance != "dark" {
			return nil, bad("Invalid appearance.")
		}
		_, err := a.DB.Exec("UPDATE users SET appearance=? WHERE id=?", v.Appearance, who(r).ID)
		return map[string]bool{"ok": true}, err
	})
	a.endpoint("POST /api/v1/auth/forgot-password", false, false, a.forgot)
	a.endpoint("POST /api/v1/auth/reset-password", false, false, a.redeem)
	a.endpoint("POST /api/v1/auth/accept-invite", false, false, a.redeem)
	a.endpoint("GET /api/v1/admin/users", true, true, a.listUsers)
	a.endpoint("POST /api/v1/admin/users", true, true, a.invite)
	a.endpoint("PATCH /api/v1/admin/users/{id}", true, true, a.updateUser)
	a.endpoint("DELETE /api/v1/admin/users/{id}", true, true, a.deleteUser)
	a.endpoint("POST /api/v1/admin/users/{id}/invite", true, true, a.resendInvite)
	a.endpoint("POST /api/v1/admin/users/{id}/reset", true, true, a.adminReset)
	a.endpoint("GET /api/v1/admin/settings", true, true, func(w http.ResponseWriter, r *http.Request) (any, error) {
		s, err := a.settings()
		configured := s.SMTP.Password != ""
		s.SMTP.Password = ""
		return map[string]any{"settings": s, "passwordConfigured": configured}, err
	})
	a.endpoint("PUT /api/v1/admin/settings", true, true, a.updateSettings)
	a.endpoint("POST /api/v1/admin/settings/test-email", true, true, func(w http.ResponseWriter, r *http.Request) (any, error) {
		s, err := a.settings()
		if err != nil {
			return nil, err
		}
		err = sendMail(r.Context(), s.SMTP, who(r).Email, "mindflow test email", "Your mindflow email settings are working.")
		if err != nil {
			return nil, bad("The SMTP test failed. Check the host, credentials, and TLS settings.")
		}
		return map[string]bool{"ok": true}, nil
	})
}
func (a *App) setup(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.limited(r, "setup") {
		return nil, problem(429, "rate_limit", "Please retry in a minute.")
	}
	var v struct {
		Credential string   `json:"credential"`
		Email      string   `json:"email"`
		Password   string   `json:"password"`
		Settings   Settings `json:"settings"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	if a.Config.Bootstrap == "" || subtle.ConstantTimeCompare([]byte(v.Credential), []byte(a.Config.Bootstrap)) != 1 {
		return nil, problem(403, "bootstrap", "Incorrect setup credential. Check the server startup log.")
	}
	email, err := normalizeEmail(v.Email)
	if err != nil {
		return nil, err
	}
	v.Settings.PublicURL = strings.TrimRight(v.Settings.PublicURL, "/")
	if err = validSettings(v.Settings, a.Config.Dev); err != nil {
		return nil, err
	}
	a.hashMu.Lock()
	hash, err := hashPassword(v.Password)
	a.hashMu.Unlock()
	if err != nil {
		return nil, err
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var init int
	if err = tx.QueryRow("SELECT initialized FROM app_settings WHERE id=1").Scan(&init); err != nil {
		return nil, err
	}
	if init != 0 {
		return nil, problem(409, "initialized", "Setup is already complete.")
	}
	if _, err = tx.Exec("INSERT INTO users(id,email,password_hash,role,status,created_at) VALUES(?,?,?,'admin','active',?)", randomID(), email, hash, now()); err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(v.Settings)
	if _, err = tx.Exec("UPDATE app_settings SET initialized=1,config=? WHERE id=1", string(raw)); err != nil {
		return nil, err
	}
	return map[string]bool{"ok": true}, tx.Commit()
}
func (a *App) setupTestEmail(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.limited(r, "setup-mail") {
		return nil, problem(429, "rate_limit", "Please retry in a minute.")
	}
	var v struct {
		Credential string   `json:"credential"`
		Email      string   `json:"email"`
		Settings   Settings `json:"settings"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	var init int
	if err := a.DB.QueryRow("SELECT initialized FROM app_settings WHERE id=1").Scan(&init); err != nil {
		return nil, err
	}
	if init != 0 || a.Config.Bootstrap == "" || subtle.ConstantTimeCompare([]byte(v.Credential), []byte(a.Config.Bootstrap)) != 1 {
		return nil, problem(403, "bootstrap", "Setup is unavailable or the credential is incorrect.")
	}
	if err := validSettings(v.Settings, a.Config.Dev); err != nil {
		return nil, err
	}
	email, err := normalizeEmail(v.Email)
	if err != nil {
		return nil, err
	}
	if err = sendMail(r.Context(), v.Settings.SMTP, email, "mindflow test email", "Your mindflow email settings are working."); err != nil {
		return nil, bad("The SMTP test failed. Check your settings.")
	}
	return map[string]bool{"ok": true}, nil
}
func (a *App) login(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.limited(r, "login") {
		return nil, problem(429, "rate_limit", "Too many attempts. Please retry in a minute.")
	}
	var v struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	email := strings.ToLower(strings.TrimSpace(v.Email))
	var u User
	var hash string
	err := a.DB.QueryRow("SELECT id,email,role,status,appearance,password_hash FROM users WHERE email=?", email).Scan(&u.ID, &u.Email, &u.Role, &u.Status, &u.Appearance, &hash)
	if err != nil || hash == "" {
		hash = a.dummyHash
	}
	a.hashMu.Lock()
	valid := verifyPassword(hash, v.Password)
	a.hashMu.Unlock()
	if err != nil || !valid || u.Status != "active" {
		return nil, problem(401, "invalid_login", "Email or password is incorrect, or the account is unavailable.")
	}
	id, secret, csrf := randomID(), randomID(), randomID()
	_, err = a.DB.Exec("INSERT INTO sessions(id,user_id,verifier_hash,csrf,expires_at) VALUES(?,?,?,?,?)", id, u.ID, digest(secret), csrf, now()+30*86400)
	if err != nil {
		return nil, err
	}
	http.SetCookie(w, &http.Cookie{Name: "mindflow_session", Value: id + "." + secret, Path: "/", HttpOnly: true, Secure: !a.Config.Dev, SameSite: http.SameSiteLaxMode, MaxAge: 30 * 86400})
	return map[string]any{"user": u, "csrf": csrf}, nil
}
func (a *App) listUsers(w http.ResponseWriter, r *http.Request) (any, error) {
	rows, err := a.DB.Query(`SELECT u.id,u.email,u.role,u.status,u.appearance, COALESCE((SELECT last_error FROM jobs WHERE user_id=u.id AND kind='mail' ORDER BY run_at DESC LIMIT 1),'') FROM users u ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []any{}
	for rows.Next() {
		var u User
		var mailError string
		if err = rows.Scan(&u.ID, &u.Email, &u.Role, &u.Status, &u.Appearance, &mailError); err != nil {
			return nil, err
		}
		users = append(users, map[string]any{"user": u, "mailError": mailError})
	}
	return users, rows.Err()
}
func (a *App) issueToken(tx *sql.Tx, userID, email, purpose string, s Settings) error {
	if _, err := tx.Exec("DELETE FROM account_tokens WHERE user_id=? AND purpose=?", userID, purpose); err != nil {
		return err
	}
	// Superseded secret-bearing jobs must not remain deliverable.
	if _, err := tx.Exec("DELETE FROM jobs WHERE user_id=? AND kind='mail' AND dedup_key=?", userID, userID+":"+purpose); err != nil {
		return err
	}
	token := randomID()
	ttl := int64(3600)
	route := "reset-password"
	subject := "Reset your mindflow password"
	if purpose == "invite" {
		ttl = 7 * 86400
		route = "accept-invite"
		subject = "You're invited to mindflow"
	}
	if _, err := tx.Exec("INSERT INTO account_tokens(id,user_id,purpose,token_hash,expires_at) VALUES(?,?,?,?,?)", randomID(), userID, purpose, digest(token), now()+ttl); err != nil {
		return err
	}
	payload, _ := json.Marshal(mailJob{To: email, Subject: subject, Body: subject + "\n\n" + s.PublicURL + "/#" + route + "/" + token + "\n\nIf you weren't expecting this email, you can ignore it.", TokenHash: digest(token)})
	_, err := tx.Exec("INSERT INTO jobs(id,user_id,kind,dedup_key,payload,run_at) VALUES(?,?,'mail',?,?,?)", randomID(), userID, userID+":"+purpose, string(payload), now())
	return err
}
func (a *App) invite(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	email, err := normalizeEmail(v.Email)
	if err != nil {
		return nil, err
	}
	if v.Role == "" {
		v.Role = "user"
	}
	if v.Role != "admin" && v.Role != "user" {
		return nil, bad("Invalid role.")
	}
	s, err := a.settings()
	if err != nil {
		return nil, err
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRow("SELECT count(*) FROM users WHERE email=?", email).Scan(&count); err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, problem(409, "email_exists", "That email already has an account.")
	}
	id := randomID()
	if _, err = tx.Exec("INSERT INTO users(id,email,role,status,created_at) VALUES(?,?,?,'invited',?)", id, email, v.Role, now()); err != nil {
		return nil, err
	}
	if err = a.issueToken(tx, id, email, "invite", s); err != nil {
		return nil, err
	}
	return map[string]string{"id": id}, tx.Commit()
}
func (a *App) tokenForUser(r *http.Request, id, purpose string) error {
	s, err := a.settings()
	if err != nil {
		return err
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var email, status string
	if err = tx.QueryRow("SELECT email,status FROM users WHERE id=?", id).Scan(&email, &status); err != nil {
		return err
	}
	if purpose == "invite" && status != "invited" {
		return bad("This account has already been activated.")
	}
	if purpose == "reset" && status != "active" {
		return bad("Only active accounts can reset their password.")
	}
	if err = a.issueToken(tx, id, email, purpose, s); err != nil {
		return err
	}
	return tx.Commit()
}
func (a *App) resendInvite(w http.ResponseWriter, r *http.Request) (any, error) {
	return map[string]bool{"ok": true}, a.tokenForUser(r, r.PathValue("id"), "invite")
}
func (a *App) adminReset(w http.ResponseWriter, r *http.Request) (any, error) {
	return map[string]bool{"ok": true}, a.tokenForUser(r, r.PathValue("id"), "reset")
}
func (a *App) forgot(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.limited(r, "forgot") {
		return nil, problem(429, "rate_limit", "Please retry in a minute.")
	}
	var v struct {
		Email string `json:"email"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	var id string
	err := a.DB.QueryRow("SELECT id FROM users WHERE email=? AND status='active'", strings.ToLower(strings.TrimSpace(v.Email))).Scan(&id)
	if err == nil {
		if err = a.tokenForUser(r, id, "reset"); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return map[string]bool{"ok": true}, nil
}
func (a *App) redeem(w http.ResponseWriter, r *http.Request) (any, error) {
	if a.limited(r, "redeem") {
		return nil, problem(429, "rate_limit", "Please retry in a minute.")
	}
	var v struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	purpose := "reset"
	if strings.HasSuffix(r.URL.Path, "accept-invite") {
		purpose = "invite"
	}
	a.hashMu.Lock()
	hash, err := hashPassword(v.Password)
	a.hashMu.Unlock()
	if err != nil {
		return nil, err
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var uid, status string
	err = tx.QueryRow("SELECT t.user_id,u.status FROM account_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=? AND t.purpose=? AND t.expires_at>?", digest(v.Token), purpose, now()).Scan(&uid, &status)
	if err != nil || (purpose == "invite" && status != "invited") || (purpose == "reset" && status != "active") {
		return nil, bad("This link is invalid or expired. Request a new reset link or ask your administrator to resend the invitation.")
	}
	if _, err = tx.Exec("UPDATE users SET password_hash=?,status='active' WHERE id=?", hash, uid); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("DELETE FROM account_tokens WHERE user_id=?", uid); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("DELETE FROM sessions WHERE user_id=?", uid); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("DELETE FROM jobs WHERE user_id=? AND kind='mail'", uid); err != nil {
		return nil, err
	}
	return map[string]bool{"ok": true}, tx.Commit()
}
func guardLastAdmin(tx *sql.Tx, id, newRole, newStatus string) error {
	var role, status string
	if err := tx.QueryRow("SELECT role,status FROM users WHERE id=?", id).Scan(&role, &status); err != nil {
		return err
	}
	if role == "admin" && status == "active" && (newRole != "admin" || newStatus != "active") {
		var n int
		if err := tx.QueryRow("SELECT count(*) FROM users WHERE role='admin' AND status='active'").Scan(&n); err != nil {
			return err
		}
		if n <= 1 {
			return bad("Keep at least one active administrator.")
		}
	}
	return nil
}
func (a *App) updateUser(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		Role   string `json:"role"`
		Status string `json:"status"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	id := r.PathValue("id")
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var role, status string
	if err = tx.QueryRow("SELECT role,status FROM users WHERE id=?", id).Scan(&role, &status); err != nil {
		return nil, err
	}
	if v.Role == "" {
		v.Role = role
	}
	if v.Status == "" {
		v.Status = status
	}
	if v.Role != "user" && v.Role != "admin" {
		return nil, bad("Invalid role.")
	}
	if v.Status != "active" && v.Status != "inactive" && v.Status != "invited" {
		return nil, bad("Invalid status.")
	}
	if status == "invited" && v.Status != status || status != "invited" && v.Status == "invited" {
		return nil, bad("Invitation activation requires the email link.")
	}
	if err = guardLastAdmin(tx, id, v.Role, v.Status); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("UPDATE users SET role=?,status=? WHERE id=?", v.Role, v.Status, id); err != nil {
		return nil, err
	}
	if v.Status != "active" {
		if _, err = tx.Exec("DELETE FROM sessions WHERE user_id=?", id); err != nil {
			return nil, err
		}
		if _, err = tx.Exec("DELETE FROM account_tokens WHERE user_id=?", id); err != nil {
			return nil, err
		}
		if _, err = tx.Exec("DELETE FROM jobs WHERE user_id=? AND kind='mail'", id); err != nil {
			return nil, err
		}
	}
	return map[string]bool{"ok": true}, tx.Commit()
}
func (a *App) deleteUser(w http.ResponseWriter, r *http.Request) (any, error) {
	var v struct {
		Confirm bool `json:"confirm"`
	}
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	if !v.Confirm {
		return nil, bad("Confirm permanent user deletion.")
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	id := r.PathValue("id")
	if err = guardLastAdmin(tx, id, "", ""); err != nil {
		return nil, err
	}
	if _, err = tx.Exec("DELETE FROM users WHERE id=?", id); err != nil {
		return nil, err
	}
	return map[string]bool{"ok": true}, tx.Commit()
}
func (a *App) updateSettings(w http.ResponseWriter, r *http.Request) (any, error) {
	var v Settings
	if err := decode(w, r, &v); err != nil {
		return nil, err
	}
	old, err := a.settings()
	if err != nil {
		return nil, err
	}
	if v.SMTP.Password == "" {
		v.SMTP.Password = old.SMTP.Password
	}
	v.PublicURL = strings.TrimRight(v.PublicURL, "/")
	if err = validSettings(v, a.Config.Dev); err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(v)
	_, err = a.DB.Exec("UPDATE app_settings SET config=? WHERE id=1", string(raw))
	return map[string]bool{"ok": true}, err
}
