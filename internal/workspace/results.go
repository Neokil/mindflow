package workspace

import "encoding/json"

// Created identifies a newly created resource.
type Created struct {
	ID     string `json:"id"`
	PageID string `json:"pageId"`
}

// PageSnapshot pairs persisted page content with its revisions and metadata.
type PageSnapshot struct {
	Document         json.RawMessage `json:"document"`
	Revision         int             `json:"revision"`
	MetadataRevision int             `json:"metadataRevision"`
	Name             string          `json:"name"`
	WorkspaceID      string          `json:"workspaceId"`
}

// SaveReceipt acknowledges a mutation and its resulting revision.
type SaveReceipt struct {
	Revision   int    `json:"revision"`
	MutationID string `json:"mutationId"`
}
