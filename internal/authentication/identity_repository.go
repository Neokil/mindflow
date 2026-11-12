package authentication

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mindflow/internal/authentication/dbquery"
	"mindflow/internal/fault"
	"mindflow/internal/identifier"
	"strings"

	outcome "mindflow/internal/result"

	"golang.org/x/crypto/argon2"
)

// Setup atomically creates the first administrator and application settings.
func (a *Service) Setup(ctx context.Context, _ Principal, command SetupCommand) (_ outcome.Completed, err error) {
	defer func() { err = repositoryError(err) }()

	if a.Config.Bootstrap == "" || subtle.ConstantTimeCompare([]byte(command.Credential), []byte(a.Config.Bootstrap)) != 1 {
		return outcome.Completed{}, fault.Fault("bootstrap", "Incorrect setup credential. Check the server startup log.")
	}
	email, err := NormalizeEmail(command.Email)
	if err != nil {
		return outcome.Completed{}, err
	}
	command.Settings.PublicURL = strings.TrimRight(command.Settings.PublicURL, "/")
	err = ValidateSettings(command.Settings, a.Config.Dev)
	if err != nil {
		return outcome.Completed{}, err
	}
	a.hashMu.Lock()
	hash, err := hashPassword(command.Password)
	a.hashMu.Unlock()
	if err != nil {
		return outcome.Completed{}, err
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("identity_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var init int
	setupStatusRow, err := dbquery.New(transaction).SetupStatus(ctx)
	init = setupStatusRow
	if err != nil {
		return outcome.Completed{}, err
	}
	if init != 0 {
		return outcome.Completed{}, fault.Fault("initialized", "Setup is already complete.")
	}
	_, err = dbquery.New(transaction).CreateAdministrator(ctx, dbquery.CreateAdministratorParams{ID: identifier.New(), Email: email, PasswordHash: hash, CreatedAt: identifier.Now()})
	if err != nil {
		return outcome.Completed{}, err
	}
	raw, err := json.Marshal(command.Settings)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("encode settings", err)
	}
	_, err = dbquery.New(transaction).CompleteSetup(ctx, string(raw))
	if err != nil {
		return outcome.Completed{}, err
	}

	return outcome.Completed{OK: true}, fault.Wrap("identity_repository.go", transaction.Commit())
}

// SetupTestEmail tests SMTP settings using the first-run credential.
func (a *Service) SetupTestEmail(ctx context.Context, _ Principal, command SetupTestEmailCommand) (_ outcome.Completed, err error) {
	defer func() { err = repositoryError(err) }()

	var init int
	setupStatusRow, err := dbquery.New(a.db).SetupStatus(ctx)
	init = setupStatusRow
	if err != nil {
		return outcome.Completed{}, err
	}
	if init != 0 || a.Config.Bootstrap == "" || subtle.ConstantTimeCompare([]byte(command.Credential), []byte(a.Config.Bootstrap)) != 1 {
		return outcome.Completed{}, fault.Fault("bootstrap", "Setup is unavailable or the credential is incorrect.")
	}
	validateSettingsErr := ValidateSettings(command.Settings, a.Config.Dev)
	if validateSettingsErr != nil {
		return outcome.Completed{}, validateSettingsErr
	}
	email, err := NormalizeEmail(command.Email)
	if err != nil {
		return outcome.Completed{}, err
	}
	err = sendMail(ctx, command.Settings.SMTP, email, "mindflow test email", "Your mindflow email settings are working.")
	if err != nil {
		return outcome.Completed{}, fault.Invalid("The SMTP test failed. Check your settings.")
	}

	return outcome.Completed{OK: true}, nil
}

// Login verifies credentials and creates a revocable session.
func (a *Service) Login(ctx context.Context, _ Principal, command LoginCommand) (_ LoginResult, err error) {
	defer func() { err = repositoryError(err) }()

	email := strings.ToLower(strings.TrimSpace(command.Email))
	var user User
	var hash string
	userByEmailRow, err := dbquery.New(a.db).UserByEmail(ctx, email)
	user.ID = userByEmailRow.ID
	user.Email = userByEmailRow.Email
	user.Role = userByEmailRow.Role
	user.Status = userByEmailRow.Status
	user.Appearance = userByEmailRow.Appearance
	hash = userByEmailRow.PasswordHash

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return LoginResult{}, err
	}
	if err != nil || hash == "" {
		hash = a.dummyHash
	}
	a.hashMu.Lock()
	valid := verifyPassword(hash, command.Password)
	a.hashMu.Unlock()
	if err != nil || !valid || user.Status != StatusActive {
		return LoginResult{}, fault.Fault("invalid_login", "Email or password is incorrect, or the account is unavailable.")
	}
	resourceID, secret, csrf := identifier.New(), identifier.New(), identifier.New()
	_, err = dbquery.New(a.db).CreateSession(ctx, dbquery.CreateSessionParams{ID: resourceID, UserID: user.ID, VerifierHash: identifier.Digest(secret), Csrf: csrf, ExpiresAt: identifier.Now() + 30*86400})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{User: user, CSRF: csrf, Cookie: resourceID + "." + secret}, nil
}

// ListUsers returns account and mail-delivery status to administrators.
func (a *Service) ListUsers(ctx context.Context, actor Principal) (_ []UserDelivery, err error) {
	requireAdministratorErr := requireAdministrator(actor)
	if requireAdministratorErr != nil {
		return nil, requireAdministratorErr
	}
	defer func() { err = repositoryError(err) }()

	rows, err := dbquery.New(a.db).ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]UserDelivery, 0, len(rows))
	for _, row := range rows {
		users = append(users, UserDelivery{User: User{ID: row.ID, Email: row.Email, Role: row.Role, Status: row.Status, Appearance: row.Appearance}, MailError: row.MailError})
	}

	return users, nil
}

// Invite creates an invited account and queues its activation email.
func (a *Service) Invite(ctx context.Context, actor Principal, command InviteCommand) (_ outcome.Created, err error) {
	requireAdministratorErr2 := requireAdministrator(actor)
	if requireAdministratorErr2 != nil {
		return outcome.Created{}, requireAdministratorErr2
	}
	defer func() { err = repositoryError(err) }()

	email, err := NormalizeEmail(command.Email)
	if err != nil {
		return outcome.Created{}, err
	}
	if command.Role == "" {
		command.Role = RoleUser
	}
	if command.Role != RoleAdmin && command.Role != RoleUser {
		return outcome.Created{}, fault.Invalid("Invalid role.")
	}
	settings, err := a.settings(ctx)
	if err != nil {
		return outcome.Created{}, err
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Created{}, fault.Wrap("identity_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var count int
	countEmailRow, err := dbquery.New(transaction).CountEmail(ctx, email)
	count = int(countEmailRow)
	if err != nil {
		return outcome.Created{}, err
	}
	if count > 0 {
		return outcome.Created{}, fault.Fault("email_exists", "That email already has an account.")
	}
	resourceID := identifier.New()
	_, err = dbquery.New(transaction).CreateInvitedUser(ctx, dbquery.CreateInvitedUserParams{ID: resourceID, Email: email, Role: command.Role, CreatedAt: identifier.Now()})
	if err != nil {
		return outcome.Created{}, err
	}
	err = a.issueToken(ctx, transaction, resourceID, email, purposeInvite, settings)
	if err != nil {
		return outcome.Created{}, err
	}

	return outcome.Created{ID: resourceID}, fault.Wrap("identity_repository.go", transaction.Commit())
}

// ResendInvite replaces an unused invitation and queues another email.
func (a *Service) ResendInvite(ctx context.Context, actor Principal, resourceID string) (_ outcome.Completed, err error) {
	requireAdministratorErr3 := requireAdministrator(actor)
	if requireAdministratorErr3 != nil {
		return outcome.Completed{}, requireAdministratorErr3
	}
	defer func() { err = repositoryError(err) }()

	return outcome.Completed{OK: true}, a.tokenForUser(ctx, resourceID, purposeInvite)
}

// AdminReset queues a password reset for an active account.
func (a *Service) AdminReset(ctx context.Context, actor Principal, resourceID string) (_ outcome.Completed, err error) {
	requireAdministratorErr4 := requireAdministrator(actor)
	if requireAdministratorErr4 != nil {
		return outcome.Completed{}, requireAdministratorErr4
	}
	defer func() { err = repositoryError(err) }()

	return outcome.Completed{OK: true}, a.tokenForUser(ctx, resourceID, purposeReset)
}

// Forgot queues a reset without revealing whether an account exists.
func (a *Service) Forgot(ctx context.Context, _ Principal, v ForgotCommand) (_ outcome.Completed, err error) {
	defer func() { err = repositoryError(err) }()

	var id string
	activeUserByEmailRow, err := dbquery.New(a.db).ActiveUserByEmail(ctx, strings.ToLower(strings.TrimSpace(v.Email)))
	id = activeUserByEmailRow

	if err == nil {
		err = a.tokenForUser(ctx, id, purposeReset)
		if err != nil {
			return outcome.Completed{}, err
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return outcome.Completed{}, err
	}

	return outcome.Completed{OK: true}, nil
}

// Redeem consumes an invitation or reset token and revokes existing credentials.
func (a *Service) Redeem(ctx context.Context, _ Principal, purpose string, command RedeemCommand) (_ outcome.Completed, err error) {
	defer func() { err = repositoryError(err) }()

	a.hashMu.Lock()
	hash, err := hashPassword(command.Password)
	a.hashMu.Unlock()
	if err != nil {
		return outcome.Completed{}, err
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("identity_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var uid, status string
	findAccountTokenRow, err := dbquery.New(transaction).FindAccountToken(ctx, dbquery.FindAccountTokenParams{TokenHash: identifier.Digest(command.Token), Purpose: purpose, ExpiresAt: identifier.Now()})
	uid = findAccountTokenRow.UserID
	status = findAccountTokenRow.Status

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return outcome.Completed{}, err
	}
	if err != nil || (purpose == purposeInvite && status != StatusInvited) || (purpose == purposeReset && status != StatusActive) {
		return outcome.Completed{}, fault.Invalid("This link is invalid or expired. Request a new reset link or ask your administrator to resend the invitation.")
	}
	_, err = dbquery.New(transaction).ActivateAccount(ctx, dbquery.ActivateAccountParams{PasswordHash: hash, ID: uid})
	if err != nil {
		return outcome.Completed{}, err
	}
	_, err = dbquery.New(transaction).DeleteAccountTokens(ctx, uid)
	if err != nil {
		return outcome.Completed{}, err
	}
	_, err = dbquery.New(transaction).DeleteUserSessions(ctx, uid)
	if err != nil {
		return outcome.Completed{}, err
	}
	_, err = dbquery.New(transaction).DeleteUserMail(ctx, sql.NullString{String: uid, Valid: true})
	if err != nil {
		return outcome.Completed{}, err
	}

	return outcome.Completed{OK: true}, fault.Wrap("identity_repository.go", transaction.Commit())
}

// UpdateUser updates an account while preserving the last active administrator.
func (a *Service) UpdateUser(ctx context.Context, actor Principal, resourceID string, command UpdateUserCommand) (_ outcome.Completed, err error) {
	requireAdministratorErr5 := requireAdministrator(actor)
	if requireAdministratorErr5 != nil {
		return outcome.Completed{}, requireAdministratorErr5
	}
	defer func() { err = repositoryError(err) }()

	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("identity_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var role, status string
	userRoleStatusRow, err := dbquery.New(transaction).UserRoleStatus(ctx, resourceID)
	role = userRoleStatusRow.Role
	status = userRoleStatusRow.Status
	if err != nil {
		return outcome.Completed{}, err
	}
	command.Role, command.Status, err = ChangeAccount(role, status, command.Role, command.Status)
	if err != nil {
		return outcome.Completed{}, err
	}
	err = guardLastAdmin(ctx, transaction, resourceID, command.Role, command.Status)
	if err != nil {
		return outcome.Completed{}, err
	}
	_, err = dbquery.New(transaction).UpdateUser(ctx, dbquery.UpdateUserParams{Role: command.Role, Status: command.Status, ID: resourceID})
	if err != nil {
		return outcome.Completed{}, err
	}
	if command.Status != StatusActive {
		_, err = dbquery.New(transaction).DeleteUserSessions(ctx, resourceID)
		if err != nil {
			return outcome.Completed{}, err
		}
		_, err = dbquery.New(transaction).DeleteAccountTokens(ctx, resourceID)
		if err != nil {
			return outcome.Completed{}, err
		}
		_, err = dbquery.New(transaction).DeleteUserMail(ctx, sql.NullString{String: resourceID, Valid: true})
		if err != nil {
			return outcome.Completed{}, err
		}
	}

	return outcome.Completed{OK: true}, fault.Wrap("identity_repository.go", transaction.Commit())
}

// DeleteUser permanently deletes an account and its cascading content.
func (a *Service) DeleteUser(ctx context.Context, actor Principal, resourceID string, command DeleteUserCommand) (_ outcome.Completed, err error) {
	requireAdministratorErr6 := requireAdministrator(actor)
	if requireAdministratorErr6 != nil {
		return outcome.Completed{}, requireAdministratorErr6
	}
	defer func() { err = repositoryError(err) }()

	if !command.Confirm {
		return outcome.Completed{}, fault.Invalid("Confirm permanent user deletion.")
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("identity_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	err = guardLastAdmin(ctx, transaction, resourceID, "", "")
	if err != nil {
		return outcome.Completed{}, err
	}
	_, err = dbquery.New(transaction).DeleteUser(ctx, resourceID)
	if err != nil {
		return outcome.Completed{}, err
	}

	return outcome.Completed{OK: true}, fault.Wrap("identity_repository.go", transaction.Commit())
}

// UpdateSettings validates application settings and preserves an omitted SMTP password.
func (a *Service) UpdateSettings(ctx context.Context, actor Principal, command UpdateSettingsCommand) (_ outcome.Completed, err error) {
	requireAdministratorErr7 := requireAdministrator(actor)
	if requireAdministratorErr7 != nil {
		return outcome.Completed{}, requireAdministratorErr7
	}
	defer func() { err = repositoryError(err) }()

	old, err := a.settings(ctx)
	if err != nil {
		return outcome.Completed{}, err
	}
	if command.SMTP.Password == "" {
		command.SMTP.Password = old.SMTP.Password
	}
	command.PublicURL = strings.TrimRight(command.PublicURL, "/")
	err = ValidateSettings(Settings(command), a.Config.Dev)
	if err != nil {
		return outcome.Completed{}, err
	}
	raw, err := json.Marshal(command)
	if err != nil {
		return outcome.Completed{}, fault.Wrap("encode settings", err)
	}
	_, err = dbquery.New(a.db).UpdateSettings(ctx, string(raw))

	return outcome.Completed{OK: true}, err
}

func (a *Service) settings(ctx context.Context) (Settings, error) {
	var raw string
	settingsRow, err := dbquery.New(a.db).Settings(ctx)
	raw = settingsRow

	var s Settings
	if err == nil {
		err = json.Unmarshal([]byte(raw), &s)
	}

	return s, fault.Wrap("identity_repository.go", err)
}

func hashPassword(password string) (string, error) {
	err := ValidatePassword(password)
	if err != nil {
		return "", err
	}
	salt := make([]byte, passwordSaltBytes)
	_, readErr := rand.Read(salt)
	if readErr != nil {
		return "", fault.Wrap("identity_repository.go", readErr)
	}
	h := argon2.IDKey([]byte(password), salt, passwordIterations, passwordMemoryKiB, 1, passwordKeyBytes)

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
	h := argon2.IDKey([]byte(password), salt, passwordIterations, passwordMemoryKiB, 1, passwordKeyBytes)

	return subtle.ConstantTimeCompare(h, expected) == 1
}

func (a *Service) issueToken(ctx context.Context, transaction *sql.Tx, userID, email, purpose string, settings Settings) error {
	_, deletePurposeTokensErr := dbquery.New(transaction).DeletePurposeTokens(ctx, dbquery.DeletePurposeTokensParams{UserID: userID, Purpose: purpose})
	if deletePurposeTokensErr != nil {
		return deletePurposeTokensErr
	}
	// Superseded secret-bearing jobs must not remain deliverable.
	_, deletePurposeMailErr := dbquery.New(transaction).DeletePurposeMail(ctx, dbquery.DeletePurposeMailParams{UserID: sql.NullString{String: userID, Valid: true}, DedupKey: sql.NullString{String: userID + ":" + purpose, Valid: true}})
	if deletePurposeMailErr != nil {
		return deletePurposeMailErr
	}
	token := identifier.New()
	ttl := int64(resetTokenSeconds)
	route := "reset-password"
	subject := "Reset your mindflow password"
	if purpose == purposeInvite {
		ttl = inviteLifetimeDays * secondsPerDay
		route = "accept-invite"
		subject = "You're invited to mindflow"
	}
	_, createAccountTokenErr := dbquery.New(transaction).CreateAccountToken(ctx, dbquery.CreateAccountTokenParams{ID: identifier.New(), UserID: userID, Purpose: purpose, TokenHash: identifier.Digest(token), ExpiresAt: identifier.Now() + ttl})
	if createAccountTokenErr != nil {
		return createAccountTokenErr
	}
	payload, err := json.Marshal(mailJob{To: email, Subject: subject, Body: subject + "\n\n" + settings.PublicURL + "/#" + route + "/" + token + "\n\nIf you weren't expecting this email, you can ignore it.", TokenHash: identifier.Digest(token)})
	if err != nil {
		return fault.Wrap("encode invitation email", err)
	}
	_, err = dbquery.New(transaction).QueueMail(ctx, dbquery.QueueMailParams{ID: identifier.New(), UserID: sql.NullString{String: userID, Valid: true}, DedupKey: sql.NullString{String: userID + ":" + purpose, Valid: true}, Payload: string(payload), RunAt: identifier.Now()})

	return err
}

func (a *Service) tokenForUser(ctx context.Context, resourceID, purpose string) error {
	settings, err := a.settings(ctx)
	if err != nil {
		return err
	}
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return fault.Wrap("identity_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var email, status string
	userEmailStatusRow, err := dbquery.New(transaction).UserEmailStatus(ctx, resourceID)
	email = userEmailStatusRow.Email
	status = userEmailStatusRow.Status
	if err != nil {
		return err
	}
	if purpose == purposeInvite && status != StatusInvited {
		return fault.Invalid("This account has already been activated.")
	}
	if purpose == purposeReset && status != StatusActive {
		return fault.Invalid("Only active accounts can reset their password.")
	}
	err = a.issueToken(ctx, transaction, resourceID, email, purpose, settings)
	if err != nil {
		return err
	}

	return fault.Wrap("identity_repository.go", transaction.Commit())
}

func guardLastAdmin(ctx context.Context, transaction *sql.Tx, id, newRole, newStatus string) error {
	var role, status string
	userRoleStatusRow, err := dbquery.New(transaction).UserRoleStatus(ctx, id)
	role = userRoleStatusRow.Role
	status = userRoleStatusRow.Status
	if err != nil {
		return err
	}
	var count int
	countAdministratorsRow, err := dbquery.New(transaction).CountAdministrators(ctx)
	count = int(countAdministratorsRow)
	if err != nil {
		return err
	}

	return CheckLastAdministrator(role, status, newRole, newStatus, count)
}
