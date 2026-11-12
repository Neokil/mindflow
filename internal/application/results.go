package application

import (
	"encoding/json"

	"mindflow/internal/domain"
)

// Result types are explicit application contracts, shared by adapters. They do
// not contain HTTP status codes or depend on a database driver.
type Completed struct {
	OK bool `json:"ok"`
}
type Created struct {
	ID string `json:"id"`
}
type WorkspaceCreated struct {
	ID     string `json:"id"`
	PageID string `json:"pageId"`
}
type PageSnapshot struct {
	Document         json.RawMessage `json:"document"`
	Revision         int             `json:"revision"`
	MetadataRevision int             `json:"metadataRevision"`
	Name             string          `json:"name"`
	WorkspaceID      string          `json:"workspaceId"`
}
type SaveReceipt struct {
	Revision   int    `json:"revision"`
	MutationID string `json:"mutationId"`
}
type UserDelivery struct {
	User      domain.User `json:"user"`
	MailError string      `json:"mailError"`
}
