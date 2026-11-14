package email_test

import (
	"encoding/base64"
	"image/png"
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
		if expected == htmlMIME && (strings.Count(string(body), `href="https://mindflow.example/#accept-invite/secret?mode=setup&amp;next=home"`) != 2 || strings.Contains(string(body), "<script>")) {
			t.Fatal("unsafe or unlinked HTML", string(body))
		}
	}
	image, imageErr := root.NextPart()
	if imageErr != nil || image.Header.Get("Content-ID") != "<mindflow-icon>" {
		t.Fatal("missing inline icon")
	}
	github, githubErr := root.NextPart()
	if githubErr != nil || github.Header.Get("Content-ID") != "<github-icon>" {
		t.Fatal("missing inline GitHub icon", githubErr)
	}
	config, decodeErr := png.DecodeConfig(base64.NewDecoder(base64.StdEncoding, github))
	if decodeErr != nil || config.Width != githubIconPixels || config.Height != githubIconPixels {
		t.Fatal("invalid GitHub icon", config, decodeErr)
	}
}

func TestLabeledWelcomeLinksAreEscapedAndDistinct(t *testing.T) {
	t.Parallel()
	const application = "https://mindflow.example/"
	const releases = "https://github.com/Neokil/mindflow/releases"
	const issues = "https://github.com/Neokil/mindflow/issues"
	value, err := email.FormatMessage(email.Config{From: "mindflow@example.com", Name: "mindflow"}, "friend@example.com", "Welcome <Simon>!", "Open mindflow\t"+application+"\nView patchnotes on GitHub\t"+releases+"\nReport an error or request a feature on GitHub\t"+issues)
	if err != nil {
		t.Fatal(err)
	}
	parts := alternativeBodies(t, value)
	html := parts[htmlMIME]
	for _, label := range []string{"Open mindflow", "View patchnotes on GitHub", "Report an error or request a feature on GitHub", "Welcome &lt;Simon&gt;!"} {
		if !strings.Contains(html, label) {
			t.Fatal("missing escaped welcome content", label)
		}
	}
	for _, address := range []string{application, releases, issues} {
		if strings.Count(html, address) != 1 || strings.Contains(html, ">"+address+"</a>") {
			t.Fatal("welcome link repeated as a raw URL", address)
		}
		if !strings.Contains(parts["text/plain"], address) {
			t.Fatal("plain-text welcome link missing", address)
		}
	}
	if !strings.Contains(html, `src="cid:github-icon"`) || strings.Contains(html, ">GitHub</a>") {
		t.Fatal("header does not use the GitHub icon")
	}
}

func alternativeBodies(t *testing.T, value string) map[string]string {
	t.Helper()
	message, err := mail.ReadMessage(strings.NewReader(value))
	if err != nil {
		t.Fatal(err)
	}
	_, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	root := multipart.NewReader(message.Body, params["boundary"])
	alternative, err := root.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	_, params, err = mime.ParseMediaType(alternative.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	reader := multipart.NewReader(alternative, params["boundary"])
	parts := map[string]string{}
	for {
		part, readErr := reader.NextPart()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			t.Fatal(readErr)
		}
		kind, _, parseErr := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		body, bodyErr := io.ReadAll(part)
		if bodyErr != nil {
			t.Fatal(bodyErr)
		}
		parts[kind] = string(body)
	}

	return parts
}

const githubIconPixels = 40
