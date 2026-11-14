package email

import (
	"bytes"
	_ "embed"
	"encoding/base64"
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

//go:embed assets/icon.png
var icon []byte

//go:embed assets/github.png
var githubIcon []byte

type messageLine struct {
	Label string
	Text  string
	Link  string
}

const contentTypeHeader = "Content-Type"

const messageHTML = `<!doctype html><html><body style="margin:0;background:#f5f5f7;color:#1d1d1f;font-family:Arial,sans-serif"><table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:40px 16px"><table role="presentation" width="560" style="width:100%;max-width:560px;background:#fff;border-radius:20px" cellpadding="32"><tr><td><table role="presentation"><tr><td><img src="cid:mindflow-icon" width="36" height="36" alt="" /></td><td style="padding-left:10px;color:#1b5e3b;font-size:24px;font-weight:bold">mindflow<span style="color:#4caf50">.</span></td><td valign="middle" style="padding-left:20px;vertical-align:middle"><a href="https://github.com/Neokil/mindflow" aria-label="mindflow on GitHub" style="display:inline-block;line-height:0"><img src="cid:github-icon" width="20" height="20" alt="GitHub" style="display:block;border:0" /></a></td></tr></table><h1 style="font-size:24px">{{.Subject}}</h1>{{range .Lines}}{{if .Link}}<p style="padding:12px 0"><a href="{{.Link}}" style="background:#1b5e3b;color:#fff;border-radius:10px;padding:14px 20px;display:inline-block;text-decoration:none">{{if .Label}}{{.Label}}{{else}}{{$.Action}}{{end}}</a></p>{{if not .Label}}<p style="font-size:12px;overflow-wrap:anywhere"><a href="{{.Link}}" style="color:#1b5e3b">{{.Link}}</a></p>{{end}}{{else}}<p style="line-height:1.6">{{.Text}}</p>{{end}}{{end}}<hr style="border:0;border-top:1px solid #eee;margin-top:32px"><p style="font-size:12px;color:#6e6e73">Your place for thoughts.</p></td></tr></table></td></tr></table></body></html>`

// FormatMessage builds escaped HTML and plain text alternatives for delivery.
func FormatMessage(config Config, recipient, subject, body string) (string, error) {
	var content bytes.Buffer
	alternative := multipart.NewWriter(&content)
	var html bytes.Buffer
	lines := make([]messageLine, 0, strings.Count(body, "\n")+1)
	for line := range strings.SplitSeq(body, "\n") {
		if strings.TrimSpace(line) == "" || strings.TrimSpace(line) == subject || strings.HasPrefix(line, "If you weren't expecting") {
			continue
		}
		item := messageLine{Text: line}
		label, address, labeled := strings.Cut(line, "\t")
		if !labeled {
			address = line
		} else {
			item.Label = label
		}
		parsed, err := url.Parse(strings.TrimSpace(address))
		if err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" {
			item.Link = parsed.String()
		}
		lines = append(lines, item)
	}
	messageTemplate := template.Must(template.New("email").Parse(messageHTML))
	action := "Continue to mindflow"
	if strings.Contains(body, "/#accept-invite/") {
		action = "Join mindflow"
	}
	if strings.Contains(body, "/#reset-password/") {
		action = "Reset password"
	}
	err := messageTemplate.Execute(&html, struct {
		Subject string
		Lines   []messageLine
		Action  string
	}{subject, lines, action})
	if err != nil {
		return "", wrap("render email", err)
	}
	for _, part := range []struct{ kind, text string }{{"text/plain", body}, {"text/html", html.String()}} {
		headers := textproto.MIMEHeader{contentTypeHeader: {part.kind + "; charset=UTF-8"}, "Content-Transfer-Encoding": {"quoted-printable"}}
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
	var related bytes.Buffer
	root := multipart.NewWriter(&related)
	text, err := root.CreatePart(textproto.MIMEHeader{contentTypeHeader: {fmt.Sprintf("multipart/alternative; boundary=%q", alternative.Boundary())}})
	if err != nil {
		return "", wrap("create alternatives", err)
	}
	_, err = text.Write(content.Bytes())
	if err != nil {
		return "", wrap("write alternatives", err)
	}
	for _, image := range []struct {
		id, filename string
		data         []byte
	}{
		{id: "mindflow-icon", filename: "icon.png", data: icon},
		{id: "github-icon", filename: "github.png", data: githubIcon},
	} {
		err = attachImage(root, image.id, image.filename, image.data)
		if err != nil {
			return "", err
		}
	}
	err = root.Close()
	if err != nil {
		return "", wrap("close email", err)
	}
	from := (&mail.Address{Name: config.Name, Address: config.From}).String()

	return fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/related; boundary=%q\r\n\r\n%s", from, recipient, mime.QEncoding.Encode("UTF-8", subject), root.Boundary(), related.String()), nil
}

func attachImage(root *multipart.Writer, id, filename string, data []byte) error {
	picture, err := root.CreatePart(textproto.MIMEHeader{contentTypeHeader: {"image/png"}, "Content-ID": {"<" + id + ">"}, "Content-Disposition": {"inline; filename=" + filename}, "Content-Transfer-Encoding": {"base64"}})
	if err != nil {
		return wrap("create email icon", err)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	const lineWidth = 76
	for len(encoded) > 0 {
		length := min(lineWidth, len(encoded))
		_, err = picture.Write([]byte(encoded[:length] + "\r\n"))
		if err != nil {
			return wrap("write email icon", err)
		}
		encoded = encoded[length:]
	}

	return nil
}
