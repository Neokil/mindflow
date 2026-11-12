package workspace

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

// ReorderPagesCommand supplies the complete desired page order and workspace revision.
type ReorderPagesCommand struct {
	IDs      []string `json:"ids"`
	Revision int      `json:"revision"`
}

// SaveDocumentCommand identifies a fenced, retryable document mutation.
type SaveDocumentCommand struct {
	BaseRevision int      `json:"baseRevision"`
	Epoch        int64    `json:"epoch"`
	ClientID     string   `json:"clientId"`
	MutationID   string   `json:"mutationId"`
	Document     Document `json:"document"`
}

// RecoveryCopyCommand preserves an unsaved document in a separate page.
type RecoveryCopyCommand struct {
	WorkspaceID string   `json:"workspaceId"`
	Name        string   `json:"name"`
	Document    Document `json:"document"`
}

// LeaseCommand requests acquisition, renewal, release or takeover of editor control.
type LeaseCommand struct {
	ClientID string `json:"clientId"`
	Action   string `json:"action"`
	Epoch    int64  `json:"epoch"`
}

// RequestVariantCommand supplies the desired canvas dimensions of an image.
type RequestVariantCommand struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
