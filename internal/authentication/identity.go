package authentication

import (
	"mindflow/internal/fault"
	"net/mail"
	"net/url"
	"strings"
	"unicode/utf8"
)

// SMTPConfig defines the encrypted SMTP connection and sender identity.
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Mode     string `json:"mode"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	From     string `json:"from"`
	Name     string `json:"name"`
}

// Settings contains the public application URL and SMTP configuration.
type Settings struct {
	PublicURL string     `json:"publicURL"`
	SMTP      SMTPConfig `json:"smtp"`
}

// User contains public account metadata without credentials.
type User struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	Appearance string `json:"appearance"`
}

// Principal identifies the authenticated account and server-side session.
type Principal struct {
	User

	SessionID string `json:"-"`
	CSRF      string `json:"-"`
}

// NormalizeEmail validates and normalizes a plain email address.
func NormalizeEmail(settings string) (string, error) {
	settings = strings.ToLower(strings.TrimSpace(settings))
	addr, err := mail.ParseAddress(settings)
	if err != nil || addr.Address != settings || len(settings) > 254 {
		return "", fault.Invalid("Enter a valid email address.")
	}

	return settings, nil
}

// ValidateSettings checks the public URL and encrypted SMTP configuration.
func ValidateSettings(settings Settings, dev bool) error {
	u, err := url.Parse(settings.PublicURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && (!dev || u.Scheme != "http")) {
		return fault.Invalid("Use the public HTTPS application URL without a path.")
	}
	if settings.SMTP.Host == "" || strings.ContainsAny(settings.SMTP.Host, "\r\n") || settings.SMTP.Port < 1 || settings.SMTP.Port > 65535 {
		return fault.Invalid("Enter the SMTP host and port.")
	}
	if settings.SMTP.Mode != "starttls" && settings.SMTP.Mode != "tls" {
		return fault.Invalid("Choose STARTTLS or TLS for SMTP.")
	}
	_, err = NormalizeEmail(settings.SMTP.From)
	if err != nil {
		return fault.Invalid("Enter a valid sender email address.")
	}
	if strings.ContainsAny(settings.SMTP.Name, "\r\n") {
		return fault.Invalid("Invalid sender name.")
	}

	return nil
}

// ValidatePassword checks the supported password length.
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < retryBaseSeconds || utf8.RuneCountInString(password) > 128 {
		return fault.Invalid("Use a password between 15 and 128 characters.")
	}

	return nil
}

// CheckLastAdministrator prevents removing the last active administrator.
func CheckLastAdministrator(role, status, newRole, newStatus string, activeAdministrators int) error {
	if role == RoleAdmin && status == StatusActive && (newRole != RoleAdmin || newStatus != StatusActive) && activeAdministrators <= 1 {
		return fault.Invalid("Keep at least one active administrator.")
	}

	return nil
}

// ChangeAccount validates role and status transitions.
func ChangeAccount(role, status, newRole, newStatus string) (string, string, error) {
	if newRole == "" {
		newRole = role
	}
	if newStatus == "" {
		newStatus = status
	}
	if newRole != RoleUser && newRole != RoleAdmin {
		return "", "", fault.Invalid("Invalid role.")
	}
	if newStatus != StatusActive && newStatus != StatusInactive && newStatus != StatusInvited {
		return "", "", fault.Invalid("Invalid status.")
	}
	if status == StatusInvited && newStatus != status || status != StatusInvited && newStatus == StatusInvited {
		return "", "", fault.Invalid("Invitation activation requires the email link.")
	}

	return newRole, newStatus, nil
}
