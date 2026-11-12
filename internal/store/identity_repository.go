package store

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"mindflow/internal/application"
	"mindflow/internal/domain"

	"golang.org/x/crypto/argon2"
)

func (a *Repositories) Setup(ctx context.Context, actor domain.Principal, v domain.SetupCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	if a.Config.Bootstrap == "" || subtle.ConstantTimeCompare([]byte(v.Credential), []byte(a.Config.Bootstrap)) != 1 {
		return application.Completed{}, domain.Fault("bootstrap", "Incorrect setup credential. Check the server startup log.")
	}
	email, err := normalizeEmail(v.Email)
	if err != nil {
		return application.Completed{}, err
	}
	v.Settings.PublicURL = strings.TrimRight(v.Settings.PublicURL, "/")
	if err = validSettings(v.Settings, a.Config.Dev); err != nil {
		return application.Completed{}, err
	}
	a.hashMu.Lock()
	hash, err := hashPassword(v.Password)
	a.hashMu.Unlock()
	if err != nil {
		return application.Completed{}, err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return application.Completed{}, err
	}
	defer tx.Rollback()
	var init int
	if err = tx.QueryRow("SELECT initialized FROM app_settings WHERE id=1").Scan(&init); err != nil {
		return application.Completed{}, err
	}
	if init != 0 {
		return application.Completed{}, domain.Fault("initialized", "Setup is already complete.")
	}
	if _, err = tx.Exec("INSERT INTO users(id,email,password_hash,role,status,created_at) VALUES(?,?,?,'admin','active',?)", randomID(), email, hash, now()); err != nil {
		return application.Completed{}, err
	}
	raw, _ := json.Marshal(v.Settings)
	if _, err = tx.Exec("UPDATE app_settings SET initialized=1,config=? WHERE id=1", string(raw)); err != nil {
		return application.Completed{}, err
	}
	return application.Completed{OK: true}, tx.Commit()

}
func (a *Repositories) SetupTestEmail(ctx context.Context, actor domain.Principal, v domain.SetupTestEmailCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	var init int
	if err := a.db.QueryRowContext(ctx, "SELECT initialized FROM app_settings WHERE id=1").Scan(&init); err != nil {
		return application.Completed{}, err
	}
	if init != 0 || a.Config.Bootstrap == "" || subtle.ConstantTimeCompare([]byte(v.Credential), []byte(a.Config.Bootstrap)) != 1 {
		return application.Completed{}, domain.Fault("bootstrap", "Setup is unavailable or the credential is incorrect.")
	}
	if err := validSettings(v.Settings, a.Config.Dev); err != nil {
		return application.Completed{}, err
	}
	email, err := normalizeEmail(v.Email)
	if err != nil {
		return application.Completed{}, err
	}
	if err = sendMail(ctx, v.Settings.SMTP, email, "mindflow test email", "Your mindflow email settings are working."); err != nil {
		return application.Completed{}, bad("The SMTP test failed. Check your settings.")
	}
	return application.Completed{OK: true}, nil

}
func (a *Repositories) Login(ctx context.Context, actor domain.Principal, v domain.LoginCommand) (result domain.LoginResult, err error) {
	defer mapRepositoryError(&err)

	email := strings.ToLower(strings.TrimSpace(v.Email))
	var u User
	var hash string
	err = a.db.QueryRowContext(ctx, "SELECT id,email,role,status,appearance,password_hash FROM users WHERE email=?", email).Scan(&u.ID, &u.Email, &u.Role, &u.Status, &u.Appearance, &hash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.LoginResult{}, err
	}
	if err != nil || hash == "" {
		hash = a.dummyHash
	}
	a.hashMu.Lock()
	valid := verifyPassword(hash, v.Password)
	a.hashMu.Unlock()
	if err != nil || !valid || u.Status != "active" {
		return domain.LoginResult{}, domain.Fault("invalid_login", "Email or password is incorrect, or the account is unavailable.")
	}
	id, secret, csrf := randomID(), randomID(), randomID()
	_, err = a.db.ExecContext(ctx, "INSERT INTO sessions(id,user_id,verifier_hash,csrf,expires_at) VALUES(?,?,?,?,?)", id, u.ID, digest(secret), csrf, now()+30*86400)
	if err != nil {
		return domain.LoginResult{}, err
	}
	return domain.LoginResult{User: u, CSRF: csrf, Cookie: id + "." + secret}, nil

}
func (a *Repositories) ListUsers(ctx context.Context, actor domain.Principal) (result []application.UserDelivery, err error) {
	defer mapRepositoryError(&err)

	rows, err := a.db.QueryContext(ctx, `SELECT u.id,u.email,u.role,u.status,u.appearance, COALESCE((SELECT last_error FROM jobs WHERE user_id=u.id AND kind='mail' ORDER BY run_at DESC LIMIT 1),'') FROM users u ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []application.UserDelivery{}
	for rows.Next() {
		var u User
		var mailError string
		if err = rows.Scan(&u.ID, &u.Email, &u.Role, &u.Status, &u.Appearance, &mailError); err != nil {
			return nil, err
		}
		users = append(users, application.UserDelivery{User: u, MailError: mailError})
	}
	return users, rows.Err()

}
func (a *Repositories) Invite(ctx context.Context, actor domain.Principal, v domain.InviteCommand) (result application.Created, err error) {
	defer mapRepositoryError(&err)

	email, err := normalizeEmail(v.Email)
	if err != nil {
		return application.Created{}, err
	}
	if v.Role == "" {
		v.Role = "user"
	}
	if v.Role != "admin" && v.Role != "user" {
		return application.Created{}, bad("Invalid role.")
	}
	s, err := a.settings(ctx)
	if err != nil {
		return application.Created{}, err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return application.Created{}, err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRow("SELECT count(*) FROM users WHERE email=?", email).Scan(&count); err != nil {
		return application.Created{}, err
	}
	if count > 0 {
		return application.Created{}, domain.Fault("email_exists", "That email already has an account.")
	}
	id := randomID()
	if _, err = tx.Exec("INSERT INTO users(id,email,role,status,created_at) VALUES(?,?,?,'invited',?)", id, email, v.Role, now()); err != nil {
		return application.Created{}, err
	}
	if err = a.issueToken(tx, id, email, "invite", s); err != nil {
		return application.Created{}, err
	}
	return application.Created{ID: id}, tx.Commit()

}
func (a *Repositories) ResendInvite(ctx context.Context, actor domain.Principal, id string) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	return application.Completed{OK: true}, a.tokenForUser(ctx, id, "invite")

}
func (a *Repositories) AdminReset(ctx context.Context, actor domain.Principal, id string) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	return application.Completed{OK: true}, a.tokenForUser(ctx, id, "reset")

}
func (a *Repositories) Forgot(ctx context.Context, actor domain.Principal, v domain.ForgotCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	var id string
	err = a.db.QueryRowContext(ctx, "SELECT id FROM users WHERE email=? AND status='active'", strings.ToLower(strings.TrimSpace(v.Email))).Scan(&id)
	if err == nil {
		if err = a.tokenForUser(ctx, id, "reset"); err != nil {
			return application.Completed{}, err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return application.Completed{}, err
	}
	return application.Completed{OK: true}, nil

}
func (a *Repositories) Redeem(ctx context.Context, actor domain.Principal, purpose string, v domain.RedeemCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	a.hashMu.Lock()
	hash, err := hashPassword(v.Password)
	a.hashMu.Unlock()
	if err != nil {
		return application.Completed{}, err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return application.Completed{}, err
	}
	defer tx.Rollback()
	var uid, status string
	err = tx.QueryRow("SELECT t.user_id,u.status FROM account_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=? AND t.purpose=? AND t.expires_at>?", digest(v.Token), purpose, now()).Scan(&uid, &status)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return application.Completed{}, err
	}
	if err != nil || (purpose == "invite" && status != "invited") || (purpose == "reset" && status != "active") {
		return application.Completed{}, bad("This link is invalid or expired. Request a new reset link or ask your administrator to resend the invitation.")
	}
	if _, err = tx.Exec("UPDATE users SET password_hash=?,status='active' WHERE id=?", hash, uid); err != nil {
		return application.Completed{}, err
	}
	if _, err = tx.Exec("DELETE FROM account_tokens WHERE user_id=?", uid); err != nil {
		return application.Completed{}, err
	}
	if _, err = tx.Exec("DELETE FROM sessions WHERE user_id=?", uid); err != nil {
		return application.Completed{}, err
	}
	if _, err = tx.Exec("DELETE FROM jobs WHERE user_id=? AND kind='mail'", uid); err != nil {
		return application.Completed{}, err
	}
	return application.Completed{OK: true}, tx.Commit()

}
func (a *Repositories) UpdateUser(ctx context.Context, actor domain.Principal, id string, v domain.UpdateUserCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return application.Completed{}, err
	}
	defer tx.Rollback()
	var role, status string
	if err = tx.QueryRow("SELECT role,status FROM users WHERE id=?", id).Scan(&role, &status); err != nil {
		return application.Completed{}, err
	}
	v.Role, v.Status, err = domain.ChangeAccount(role, status, v.Role, v.Status)
	if err != nil {
		return application.Completed{}, err
	}
	if err = guardLastAdmin(tx, id, v.Role, v.Status); err != nil {
		return application.Completed{}, err
	}
	if _, err = tx.Exec("UPDATE users SET role=?,status=? WHERE id=?", v.Role, v.Status, id); err != nil {
		return application.Completed{}, err
	}
	if v.Status != "active" {
		if _, err = tx.Exec("DELETE FROM sessions WHERE user_id=?", id); err != nil {
			return application.Completed{}, err
		}
		if _, err = tx.Exec("DELETE FROM account_tokens WHERE user_id=?", id); err != nil {
			return application.Completed{}, err
		}
		if _, err = tx.Exec("DELETE FROM jobs WHERE user_id=? AND kind='mail'", id); err != nil {
			return application.Completed{}, err
		}
	}
	return application.Completed{OK: true}, tx.Commit()

}
func (a *Repositories) DeleteUser(ctx context.Context, actor domain.Principal, id string, v domain.DeleteUserCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	if !v.Confirm {
		return application.Completed{}, bad("Confirm permanent user deletion.")
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return application.Completed{}, err
	}
	defer tx.Rollback()
	if err = guardLastAdmin(tx, id, "", ""); err != nil {
		return application.Completed{}, err
	}
	if _, err = tx.Exec("DELETE FROM users WHERE id=?", id); err != nil {
		return application.Completed{}, err
	}
	return application.Completed{OK: true}, tx.Commit()

}
func (a *Repositories) UpdateSettings(ctx context.Context, actor domain.Principal, v domain.UpdateSettingsCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	old, err := a.settings(ctx)
	if err != nil {
		return application.Completed{}, err
	}
	if v.SMTP.Password == "" {
		v.SMTP.Password = old.SMTP.Password
	}
	v.PublicURL = strings.TrimRight(v.PublicURL, "/")
	if err = validSettings(Settings(v), a.Config.Dev); err != nil {
		return application.Completed{}, err
	}
	raw, _ := json.Marshal(v)
	_, err = a.db.ExecContext(ctx, "UPDATE app_settings SET config=? WHERE id=1", string(raw))
	return application.Completed{OK: true}, err

}
func (a *Repositories) settings(ctx context.Context) (Settings, error) {
	var raw string
	err := a.db.QueryRowContext(ctx, "SELECT config FROM app_settings WHERE id=1").Scan(&raw)
	var s Settings
	if err == nil {
		err = json.Unmarshal([]byte(raw), &s)
	}
	return s, err
}
func hashPassword(password string) (string, error) {
	if err := domain.ValidatePassword(password); err != nil {
		return "", err
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
func (a *Repositories) issueToken(tx *sql.Tx, userID, email, purpose string, s Settings) error {
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
func (a *Repositories) tokenForUser(ctx context.Context, id, purpose string) error {
	s, err := a.settings(ctx)
	if err != nil {
		return err
	}
	tx, err := a.db.BeginTx(ctx, nil)
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
func guardLastAdmin(tx *sql.Tx, id, newRole, newStatus string) error {
	var role, status string
	if err := tx.QueryRow("SELECT role,status FROM users WHERE id=?", id).Scan(&role, &status); err != nil {
		return err
	}
	var count int
	if err := tx.QueryRow("SELECT count(*) FROM users WHERE role='admin' AND status='active'").Scan(&count); err != nil {
		return err
	}
	return domain.CheckLastAdministrator(role, status, newRole, newStatus, count)
}
