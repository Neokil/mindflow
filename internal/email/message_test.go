package email_test

import (
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"mindflow/internal/email"
	"net/mail"
	"strings"
	"testing"
)

const htmlMIME = "text/html"

func TestMessageContainsEscapedHTMLAndClickableLinks(t *testing.T) {
	t.Parallel()
	const link = "https://mindflow.example/#accept-invite/secret?mode=setup&next=home"
	value, err := email.FormatMessage(email.Config{From: "mindflow@example.com", Name: "mindflow"}, "reader@example.com", "You're invited to mindflow", "You're invited to mindflow\n\nHello <script>alert(1)</script>\n\n"+link)
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(strings.NewReader(value))
	if err != nil {
		t.Fatal(err)
	}
	kind, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil || kind != "multipart/related" {
		t.Fatalf("invalid MIME: %s %v", kind, err)
	}
	root := multipart.NewReader(message.Body, params["boundary"])
	alternatives, partErr := root.NextPart()
	if partErr != nil {
		t.Fatal(partErr)
	}
	_, alternativeParams, parseErr := mime.ParseMediaType(alternatives.Header.Get("Content-Type"))
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	reader := multipart.NewReader(alternatives, alternativeParams["boundary"])
	for _, expected := range []string{"text/plain", htmlMIME} {
		part, partErr := reader.NextRawPart()
		if partErr != nil {
			t.Fatal(partErr)
		}
		body, readErr := io.ReadAll(quotedprintable.NewReader(part))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.HasPrefix(part.Header.Get("Content-Type"), expected) {
			t.Fatal("wrong alternative order")
		}
		if expected == "text/plain" && !strings.Contains(string(body), link) {
			t.Fatal("missing plaintext fallback")
		}
		if expected == htmlMIME && (strings.Count(string(body), "invited to mindflow") != 1 || !strings.Contains(string(body), "Join mindflow") || !strings.Contains(string(body), "cid:mindflow-icon") || !strings.Contains(string(body), "https://github.com/Neokil/mindflow")) {
			t.Fatal("missing or duplicated invitation content")
		}
		if expected == htmlMIME && (!strings.Contains(string(body), `href="https://mindflow.example/#accept-invite/secret?mode=setup&amp;next=home"`) || strings.Contains(string(body), "<script>")) {
			t.Fatal("unsafe or unlinked HTML", string(body))
		}
	}
	image, imageErr := root.NextPart()
	if imageErr != nil || image.Header.Get("Content-ID") != "<mindflow-icon>" {
		t.Fatal("missing inline icon")
	}
}
