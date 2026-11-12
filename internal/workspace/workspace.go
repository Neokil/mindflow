package workspace

import (
	"mindflow/internal/fault"
	"strings"
	"unicode/utf8"
)

// Workspace is an owned collection of ordered pages.
type Workspace struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Revision  int    `json:"revision"`
	UpdatedAt int64  `json:"updatedAt"`
	PageCount int    `json:"pageCount"`
}

// Page contains page metadata without its document body.
type Page struct {
	ID               string `json:"id"`
	WorkspaceID      string `json:"workspaceId"`
	Name             string `json:"name"`
	Position         int    `json:"position"`
	Revision         int    `json:"revision"`
	MetadataRevision int    `json:"metadataRevision"`
}

// ValidateName checks a nonempty workspace or page name.
func ValidateName(s string) error {
	if strings.TrimSpace(s) == "" || utf8.RuneCountInString(s) > 200 {
		return fault.Invalid("Use a name between 1 and 200 characters.")
	}

	return nil
}
