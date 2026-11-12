package domain

import (
	"strings"
	"unicode/utf8"
)

type Workspace struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Revision  int    `json:"revision"`
	UpdatedAt int64  `json:"updatedAt"`
	PageCount int    `json:"pageCount"`
}
type Page struct {
	ID               string `json:"id"`
	WorkspaceID      string `json:"workspaceId"`
	Name             string `json:"name"`
	Position         int    `json:"position"`
	Revision         int    `json:"revision"`
	MetadataRevision int    `json:"metadataRevision"`
}

func ValidateName(s string) error {
	if strings.TrimSpace(s) == "" || utf8.RuneCountInString(s) > 200 {
		return Invalid("Use a name between 1 and 200 characters.")
	}
	return nil
}
