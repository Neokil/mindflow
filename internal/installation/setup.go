package installation

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"mindflow/internal/account"
	"mindflow/internal/email"
	"net/url"
	"strings"
)

// Settings loads operator-configured application settings.
type Settings struct {
	PublicURL string       `json:"publicURL"`
	SMTP      email.Config `json:"smtp"`
}

// SetupCommand supplies the one-time credential, first administrator and operator settings.
type SetupCommand struct {
	Credential  string   `json:"credential"`
	Email       string   `json:"email"`
	DisplayName string   `json:"displayName"`
	Password    string   `json:"password"`
	Settings    Settings `json:"settings"`
}

// SetupTestEmailCommand tests SMTP before the installation is committed.
type SetupTestEmailCommand struct {
	Credential string   `json:"credential"`
	Email      string   `json:"email"`
	Settings   Settings `json:"settings"`
}

// UpdateSettingsCommand updates the public URL and SMTP settings.
type UpdateSettingsCommand Settings

// ValidateSettings checks the public URL and encrypted SMTP configuration.
func ValidateSettings(settings Settings, dev bool) error {
	u, err := url.Parse(settings.PublicURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && (!dev || u.Scheme != "http")) {
		return invalid("Use the public HTTPS application URL without a path.")
	}
	if settings.SMTP.Host == "" || strings.ContainsAny(settings.SMTP.Host, "\r\n") || settings.SMTP.Port < 1 || settings.SMTP.Port > 65535 {
		return invalid("Enter the SMTP host and port.")
	}
	if settings.SMTP.Mode != "starttls" && settings.SMTP.Mode != "tls" {
		return invalid("Choose STARTTLS or TLS for SMTP.")
	}
	_, err = account.NormalizeEmail(settings.SMTP.From)
	if err != nil {
		return invalid("Enter a valid sender email address.")
	}
	if strings.ContainsAny(settings.SMTP.Name, "\r\n") {
		return invalid("Invalid sender name.")
	}

	return nil
}

// Setup atomically creates the first administrator and application settings.
func (a *Service) Setup(ctx context.Context, _ account.User, command SetupCommand) (_ Completed, err error) {
	defer func() { err = repositoryError(err) }()

	if a.Config.Bootstrap == "" || subtle.ConstantTimeCompare([]byte(command.Credential), []byte(a.Config.Bootstrap)) != 1 {
		return Completed{}, failure("bootstrap", "Incorrect setup credential. Check the server startup log.")
	}
	address, err := account.NormalizeEmail(command.Email)
	if err != nil {
		return Completed{}, err
	}
	command.Settings.PublicURL = strings.TrimRight(command.Settings.PublicURL, "/")
	err = ValidateSettings(command.Settings, a.Config.Dev)
	if err != nil {
		return Completed{}, err
	}
	hash, err := a.accounts.HashPassword(command.Password)
	if err != nil {
		return Completed{}, err
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, wrap("identity_repository.go", err)
	}
	defer func() { _ = transaction.Rollback() }()
	var init int
	setupStatusRow, err := a.repository.Within(transaction).InstallationSetupStatus(ctx)
	init = setupStatusRow
	if err != nil {
		return Completed{}, err
	}
	if init != 0 {
		return Completed{}, failure("initialized", "Setup is already complete.")
	}
	administrator, err := a.accounts.CreateAdministratorWithin(ctx, transaction, address, command.DisplayName, hash)
	if err != nil {
		return Completed{}, err
	}
	raw, err := json.Marshal(command.Settings)
	if err != nil {
		return Completed{}, wrap("encode settings", err)
	}
	_, err = a.repository.Within(transaction).InstallationCompleteSetup(ctx, string(raw))
	if err != nil {
		return Completed{}, err
	}

	err = a.welcome.QueueWelcomeWithin(ctx, transaction, administrator, command.Settings.PublicURL)
	if err != nil {
		return Completed{}, err
	}

	return Completed{OK: true}, wrap("identity_repository.go", transaction.Commit())
}

// SetupTestEmail tests SMTP settings using the first-run credential.
func (a *Service) SetupTestEmail(ctx context.Context, _ account.User, command SetupTestEmailCommand) (_ Completed, err error) {
	defer func() { err = repositoryError(err) }()

	var init int
	setupStatusRow, err := a.repository.InstallationSetupStatus(ctx)
	init = setupStatusRow
	if err != nil {
		return Completed{}, err
	}
	if init != 0 || a.Config.Bootstrap == "" || subtle.ConstantTimeCompare([]byte(command.Credential), []byte(a.Config.Bootstrap)) != 1 {
		return Completed{}, failure("bootstrap", "Setup is unavailable or the credential is incorrect.")
	}
	validateSettingsErr := ValidateSettings(command.Settings, a.Config.Dev)
	if validateSettingsErr != nil {
		return Completed{}, validateSettingsErr
	}
	address, err := account.NormalizeEmail(command.Email)
	if err != nil {
		return Completed{}, err
	}
	err = email.Send(ctx, command.Settings.SMTP, address, "mindflow test email", "Your mindflow email settings are working.")
	if err != nil {
		return Completed{}, invalid("The SMTP test failed. Check your settings.")
	}

	return Completed{OK: true}, nil
}

// UpdateSettings validates application settings and preserves an omitted SMTP password.
func (a *Service) UpdateSettings(ctx context.Context, actor account.User, command UpdateSettingsCommand) (_ Completed, err error) {
	requireAdministratorErr7 := account.RequireAdministrator(actor)
	if requireAdministratorErr7 != nil {
		return Completed{}, requireAdministratorErr7
	}
	defer func() { err = repositoryError(err) }()

	old, err := a.settings(ctx)
	if err != nil {
		return Completed{}, err
	}
	if command.SMTP.Password == "" {
		command.SMTP.Password = old.SMTP.Password
	}
	command.PublicURL = strings.TrimRight(command.PublicURL, "/")
	err = ValidateSettings(Settings(command), a.Config.Dev)
	if err != nil {
		return Completed{}, err
	}
	raw, err := json.Marshal(command)
	if err != nil {
		return Completed{}, wrap("encode settings", err)
	}
	_, err = a.repository.InstallationUpdateSettings(ctx, string(raw))

	return Completed{OK: true}, err
}

func (a *Service) settings(ctx context.Context) (Settings, error) {
	var raw string
	settingsRow, err := a.repository.InstallationSettings(ctx)
	raw = settingsRow

	var s Settings
	if err == nil {
		err = json.Unmarshal([]byte(raw), &s)
	}

	return s, wrap("identity_repository.go", err)
}

// Initialized reports whether first-run setup is complete.
func (a *Service) Initialized(ctx context.Context) (bool, error) {
	var n int
	setupStatusRow, err := a.repository.InstallationSetupStatus(ctx)
	n = setupStatusRow

	return n == 1, err
}
