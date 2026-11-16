package email

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/yuin/goldmark"
)

// ProductUpdate describes an opted-in release announcement independently of its layout.
type ProductUpdate struct {
	Version        string `json:"version"`
	Changelog      string `json:"changelog"`
	ApplicationURL string `json:"applicationURL"`
	ReleaseURL     string `json:"releaseURL"`
	UnsubscribeURL string `json:"unsubscribeURL"`
}

func (notice ProductUpdate) subject() string { return "mindflow " + notice.Version + " is available" }

func (notice ProductUpdate) plainText() string {
	return "New features are available on mindflow\nVersion " + notice.Version + " now available\n\n" + notice.notes() + "\n\nView this release on GitHub: " + notice.ReleaseURL + "\n\nOpen mindflow: " + notice.ApplicationURL + "\n\nYour place for thoughts.\nIf you want to unsubscribe from these updates click here: " + notice.UnsubscribeURL
}

func (notice ProductUpdate) notes() string {
	if strings.TrimSpace(notice.Changelog) == "" {
		return "See GitHub for the full changelog."
	}

	return notice.Changelog
}

// FormatProductUpdate shares the branded header and MIME delivery with other mail.
func FormatProductUpdate(config Config, recipient string, notice ProductUpdate) (string, error) {
	var changelog bytes.Buffer
	// Goldmark's default renderer rejects raw HTML and dangerous link schemes.
	err := goldmark.Convert([]byte(notice.notes()), &changelog)
	if err != nil {
		return "", wrap("render release changelog", err)
	}
	view := messageView{Update: &notice, Changelog: template.HTML(changelog.String())} //nolint:gosec // G203: only Goldmark output with raw HTML and unsafe links disabled is trusted.
	var html bytes.Buffer
	err = template.Must(template.New("email").Parse(messageHTML)).Execute(&html, view)
	if err != nil {
		return "", wrap("render product update", err)
	}

	return formatAlternatives(config, recipient, notice.subject(), notice.plainText(), html.String())
}
