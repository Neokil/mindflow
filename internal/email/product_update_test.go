package email_test

import (
	"mindflow/internal/email"
	"strings"
	"testing"
)

const (
	testBrand  = "mindflow"
	testSender = "mindflow@example.com"
	testReader = "reader@example.com"
)

func TestProductUpdateShowsChangelogBeforeOneButtonAndQuietFooter(t *testing.T) {
	t.Parallel()
	notice := email.ProductUpdate{Version: "v1.3.5", Changelog: "## Improvements\n\n- **Faster images** when switching pages.\n- Improved email layout.", ApplicationURL: "https://mindflow.example/", ReleaseURL: "https://github.com/Neokil/mindflow/releases/tag/v1.3.5", UnsubscribeURL: "https://mindflow.example/api/v1/auth/product-updates/unsubscribe?token=example&mode=test"}
	message, err := email.FormatProductUpdate(email.Config{From: testSender, Name: testBrand}, testReader, notice)
	if err != nil {
		t.Fatal(err)
	}
	bodies := alternativeBodies(t, message)
	html := bodies[htmlMIME]
	previous := -1
	for _, text := range []string{"New features are available on mindflow", "Version v1.3.5 now available", "<h2>Improvements</h2>", "<strong>Faster images</strong>", "View this release on GitHub", "Open mindflow", "Your place for thoughts.", "If you want to unsubscribe from these updates click"} {
		position := strings.Index(html, text)
		if position <= previous {
			t.Fatal("missing or misplaced update content", text, html)
		}
		previous = position
	}
	if strings.Count(html, "background:#1b5e3b") != 1 {
		t.Fatal("only Open mindflow should be a button", html)
	}
	if !strings.Contains(html, `href="https://mindflow.example/api/v1/auth/product-updates/unsubscribe?token=example&amp;mode=test" style="color:#6e6e73;text-decoration:underline">here</a>`) {
		t.Fatal("unsubscribe is not a quiet escaped footer link", html)
	}
	for _, text := range []string{notice.Changelog, notice.ApplicationURL, notice.ReleaseURL, notice.UnsubscribeURL} {
		if !strings.Contains(strings.ReplaceAll(bodies["text/plain"], "\r\n", "\n"), text) {
			t.Fatal("missing plain text fallback", text)
		}
	}
}

func TestProductUpdateRejectsUnsafeChangelogHTMLAndLinks(t *testing.T) {
	t.Parallel()
	notice := email.ProductUpdate{Version: "v1.3.5", Changelog: "<script>alert(1)</script>\n\n[unsafe](javascript:alert(1))\n\n**Useful update**", ApplicationURL: "https://mindflow.example/", ReleaseURL: "https://github.com/Neokil/mindflow/releases/tag/v1.3.5", UnsubscribeURL: "https://mindflow.example/unsubscribe"}
	message, err := email.FormatProductUpdate(email.Config{From: testSender, Name: testBrand}, testReader, notice)
	if err != nil {
		t.Fatal(err)
	}
	html := alternativeBodies(t, message)[htmlMIME]
	if strings.Contains(html, "<script>") || strings.Contains(html, `href="javascript:`) || !strings.Contains(html, "<strong>Useful update</strong>") {
		t.Fatal("unsafe Markdown rendering", html)
	}
}
