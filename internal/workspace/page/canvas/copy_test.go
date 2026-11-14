package canvas_test

import (
	"mindflow/internal/workspace/page/canvas"
	"testing"
)

func TestCopiedPageLinksTargetCopiesAndPreserveOnlyLocalExternalLinks(t *testing.T) {
	t.Parallel()
	document := canvas.NewDocument()
	document.Elements = []canvas.Element{{ID: "a", Link: &canvas.ElementLink{PageID: sourcePage, ElementID: "b"}}, {ID: "b", Link: &canvas.ElementLink{PageID: "elsewhere", ElementID: "c"}}}
	copied := canvas.ClonePages([]canvas.PageCopy{{SourceID: sourcePage, TargetID: copiedPage, Document: document}}, true)[0]
	if copied.Elements[0].Link.PageID != copiedPage || copied.Elements[0].Link.ElementID != copied.Elements[1].ID || copied.Elements[1].Link.PageID != "elsewhere" {
		t.Fatal("page-copy links were not retargeted", copied.Elements)
	}
	detached := canvas.ClonePages([]canvas.PageCopy{{SourceID: sourcePage, TargetID: copiedPage, Document: document}}, false)[0]
	if detached.Elements[1].Link != nil || document.Elements[0].Link.PageID != sourcePage {
		t.Fatal("external link retained or source mutated")
	}
}

const (
	sourcePage = "source"
	copiedPage = "copy"
)
