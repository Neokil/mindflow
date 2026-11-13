package email

import (
	"bytes"
	"fmt"
	"html/template"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"net/url"
	"strings"
)

type messageLine struct {
	Text string
	Link string
}

const messageHTML = `<!doctype html><html><body style="margin:0;background:#f5f5f7;color:#1d1d1f;font-family:Arial,sans-serif"><table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:40px 16px"><table role="presentation" width="560" style="max-width:100%;background:#fff;border-radius:20px" cellpadding="32"><tr><td><p style="color:#5963dc;font-size:24px;font-weight:bold">mindflow</p><h1 style="font-size:24px">{{.Subject}}</h1>{{range .Lines}}{{if .Link}}<p style="padding:12px 0"><a href="{{.Link}}" style="background:#5963dc;color:#fff;border-radius:10px;padding:14px 20px;display:inline-block;text-decoration:none">Continue to mindflow</a></p><p style="font-size:12px;overflow-wrap:anywhere"><a href="{{.Link}}">{{.Link}}</a></p>{{else}}<p style="line-height:1.6">{{.Text}}</p>{{end}}{{end}}<hr style="border:0;border-top:1px solid #eee;margin-top:32px"><p style="font-size:12px;color:#6e6e73">Your space for thoughts. If you did not expect this email, you can ignore it.</p></td></tr></table></td></tr></table></body></html>`

// FormatMessage builds escaped HTML and plain text alternatives for delivery.
func FormatMessage(config Config, recipient, subject, body string) (string, error) {
	var content bytes.Buffer
	alternative := multipart.NewWriter(&content)
	var html bytes.Buffer
	lines := make([]messageLine, 0, strings.Count(body, "\n")+1)
	for line := range strings.SplitSeq(body, "\n") {
		item := messageLine{Text: line}
		parsed, err := url.Parse(strings.TrimSpace(line))
		if err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" {
			item.Link = parsed.String()
		}
		lines = append(lines, item)
	}
	messageTemplate := template.Must(template.New("email").Parse(messageHTML))
	err := messageTemplate.Execute(&html, struct {
		Subject string
		Lines   []messageLine
	}{subject, lines})
	if err != nil {
		return "", wrap("render email", err)
	}
	for _, part := range []struct{ kind, text string }{{"text/plain", body}, {"text/html", html.String()}} {
		headers := textproto.MIMEHeader{"Content-Type": {part.kind + "; charset=UTF-8"}, "Content-Transfer-Encoding": {"quoted-printable"}}
		writer, err := alternative.CreatePart(headers)
		if err != nil {
			return "", wrap("email part", err)
		}
		encoded := quotedprintable.NewWriter(writer)
		_, err = encoded.Write([]byte(part.text))
		if err != nil {
			return "", wrap("encode email", err)
		}
		err = encoded.Close()
		if err != nil {
			return "", wrap("close email part", err)
		}
	}
	err = alternative.Close()
	if err != nil {
		return "", wrap("close email", err)
	}
	from := (&mail.Address{Name: config.Name, Address: config.From}).String()

	return fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n%s", from, recipient, mime.QEncoding.Encode("UTF-8", subject), alternative.Boundary(), content.String()), nil
}
