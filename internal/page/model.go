package page

import (
	"mindflow/internal/canvas"
)

// Page contains page metadata without its document body.
type Page struct {
	ID               string `json:"id"`
	WorkspaceID      string `json:"workspaceId"`
	Name             string `json:"name"`
	Position         int    `json:"position"`
	Revision         int    `json:"revision"`
	MetadataRevision int    `json:"metadataRevision"`
}

// CreatePageCommand names a page to append to a workspace.
type CreatePageCommand struct {
	Name string `json:"name"`
}

// RenamePageCommand changes page metadata using its current revision.
type RenamePageCommand struct {
	Name     string `json:"name"`
	Revision int    `json:"revision"`
}

// DeletePageCommand confirms permanent page deletion.
type DeletePageCommand struct {
	Revision int  `json:"revision"`
	Confirm  bool `json:"confirm"`
}

// RecoveryCopyCommand preserves unsaved content in a separate page.
type RecoveryCopyCommand struct {
	WorkspaceID string          `json:"workspaceId"`
	Name        string          `json:"name"`
	Document    canvas.Document `json:"document"`
}
