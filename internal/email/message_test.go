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

func TestMessageContainsEscapedHTMLAndClickableLinks(t *testing.T) {
	t.Parallel()
	const link = "https://mindflow.example/invite?token=secret&mode=setup"
	value, err := email.FormatMessage(email.Config{From: "mindflow@example.com", Name: "mindflow"}, "reader@example.com", "Welcome <reader>", "Hello <script>alert(1)</script>\n\n"+link)
	if err != nil {
		t.Fatal(err)
	}
	message, err := mail.ReadMessage(strings.NewReader(value))
	if err != nil {
		t.Fatal(err)
	}
	kind, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil || kind != "multipart/alternative" {
		t.Fatalf("invalid MIME: %s %v", kind, err)
	}
	reader := multipart.NewReader(message.Body, params["boundary"])
	for _, expected := range []string{"text/plain", "text/html"} {
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
		if expected == "text/html" && (!strings.Contains(string(body), `href="https://mindflow.example/invite?token=secret&amp;mode=setup"`) || strings.Contains(string(body), "<script>")) {
			t.Fatal("unsafe or unlinked HTML", string(body))
		}
	}
}
