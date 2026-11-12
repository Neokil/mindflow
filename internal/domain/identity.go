package domain

import (
	"net/mail"
	"net/url"
	"strings"
	"unicode/utf8"
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
type Principal struct {
	User
	SessionID string
	CSRF      string
}

func NormalizeEmail(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s || len(s) > 254 {
		return "", Invalid("Enter a valid email address.")
	}
	return s, nil
}
func ValidateSettings(s Settings, dev bool) error {
	u, err := url.Parse(s.PublicURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && !(dev && u.Scheme == "http")) {
		return Invalid("Use the public HTTPS application URL without a path.")
	}
	if s.SMTP.Host == "" || strings.ContainsAny(s.SMTP.Host, "\r\n") || s.SMTP.Port < 1 || s.SMTP.Port > 65535 {
		return Invalid("Enter the SMTP host and port.")
	}
	if s.SMTP.Mode != "starttls" && s.SMTP.Mode != "tls" {
		return Invalid("Choose STARTTLS or TLS for SMTP.")
	}
	if _, err = NormalizeEmail(s.SMTP.From); err != nil {
		return Invalid("Enter a valid sender email address.")
	}
	if strings.ContainsAny(s.SMTP.Name, "\r\n") {
		return Invalid("Invalid sender name.")
	}
	return nil
}
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < 15 || utf8.RuneCountInString(password) > 128 {
		return Invalid("Use a password between 15 and 128 characters.")
	}
	return nil
}
func CheckLastAdministrator(role, status, newRole, newStatus string, activeAdministrators int) error {
	if role == "admin" && status == "active" && (newRole != "admin" || newStatus != "active") && activeAdministrators <= 1 {
		return Invalid("Keep at least one active administrator.")
	}
	return nil
}
func ChangeAccount(role, status, newRole, newStatus string) (string, string, error) {
	if newRole == "" {
		newRole = role
	}
	if newStatus == "" {
		newStatus = status
	}
	if newRole != "user" && newRole != "admin" {
		return "", "", Invalid("Invalid role.")
	}
	if newStatus != "active" && newStatus != "inactive" && newStatus != "invited" {
		return "", "", Invalid("Invalid status.")
	}
	if status == "invited" && newStatus != status || status != "invited" && newStatus == "invited" {
		return "", "", Invalid("Invitation activation requires the email link.")
	}
	return newRole, newStatus, nil
}
