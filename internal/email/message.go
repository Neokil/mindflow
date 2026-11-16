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

//go:embed assets/message.html
var messageHTML string

type messageView struct {
	Subject   string
	Lines     []messageLine
	Action    string
	Update    *ProductUpdate
	Changelog template.HTML
}

// FormatMessage builds escaped HTML and plain text alternatives for delivery.
func FormatMessage(config Config, recipient, subject, body string) (string, error) {
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
	err := messageTemplate.Execute(&html, messageView{Subject: subject, Lines: lines, Action: action})
	if err != nil {
		return "", wrap("render email", err)
	}

	return formatAlternatives(config, recipient, subject, body, html.String())
}

func formatAlternatives(config Config, recipient, subject, body, html string) (string, error) {
	var content bytes.Buffer
	alternative := multipart.NewWriter(&content)

	for _, part := range []struct{ kind, text string }{{"text/plain", body}, {"text/html", html}} {
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
	err := alternative.Close()
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
