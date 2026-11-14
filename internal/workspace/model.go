package workspace

import (
	"strings"
	"unicode/utf8"
)

// Workspace is an owned collection of ordered pages.
type Workspace struct {
	FirstPageID     string `json:"firstPageId"`
	PreviewRevision int    `json:"previewRevision"`
	Role            string `json:"role"`
	Shared          bool   `json:"shared"`
	ID              string `json:"id"`
	Name            string `json:"name"`
	Revision        int    `json:"revision"`
	UpdatedAt       int64  `json:"updatedAt"`
	PageCount       int    `json:"pageCount"`
}

// CreateWorkspaceCommand names a new private workspace.
type CreateWorkspaceCommand struct {
	Name string `json:"name"`
}

// RenameWorkspaceCommand changes a name using a metadata revision.
type RenameWorkspaceCommand struct {
	Name     string `json:"name"`
	Revision int    `json:"revision"`
}

// DeleteWorkspaceCommand confirms deletion using a metadata revision.
type DeleteWorkspaceCommand struct {
	Revision int  `json:"revision"`
	Confirm  bool `json:"confirm"`
}

// ReorderPagesCommand supplies the complete page order and parent metadata revision.
type ReorderPagesCommand struct {
	IDs      []string `json:"ids"`
	Revision int      `json:"revision"`
}

// Created identifies a newly created domain resource.
type Created struct {
	ID     string `json:"id"`
	PageID string `json:"pageId"`
}

// ValidateName checks a nonempty workspace or page name.
func ValidateName(s string) error {
	if strings.TrimSpace(s) == "" || utf8.RuneCountInString(s) > 200 {
		return invalid("Use a name between 1 and 200 characters.")
	}

	return nil
}
