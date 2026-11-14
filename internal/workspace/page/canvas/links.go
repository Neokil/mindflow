package canvas

import (
	"context"
	"encoding/json"
	"mindflow/internal/workspace/storage"
)

// ElementLink names an element on the same canvas or another page of its workspace.
type ElementLink struct {
	PageID    string `json:"pageId"`
	ElementID string `json:"elementId"`
}

func (a *Service) validateLinksWithin(ctx context.Context, transaction *storage.Transaction, pageID string, document Document) error {
	hasLinks := false
	for _, element := range document.Elements {
		if element.Link != nil {
			hasLinks = true

			break
		}
	}
	if !hasLinks {
		return nil
	}
	queries := a.repository.Within(transaction)
	workspaceID, err := queries.PageWorkspace(ctx, pageID)
	if err != nil {
		return err
	}
	raw, err := queries.PagePageDocument(ctx, pageID)
	if err != nil {
		return err
	}
	var previous Document
	err = json.Unmarshal([]byte(raw), &previous)
	if err != nil {
		return wrap("decode previous element links", err)
	}
	old := map[string]*ElementLink{}
	for _, element := range previous.Elements {
		old[element.ID] = element.Link
	}
	local := map[string]bool{}
	for _, element := range document.Elements {
		local[element.ID] = true
	}
	targets := map[string]map[string]bool{pageID: local}
	for _, element := range document.Elements {
		if element.Link == nil {
			continue
		}
		if old[element.ID] != nil && *old[element.ID] == *element.Link {
			continue
		} // Retain links to deleted targets so restoring them repairs navigation.
		if element.Link.PageID == pageID && element.Link.ElementID == element.ID {
			return invalid("Choose a different target element.")
		}
		ids, loaded := targets[element.Link.PageID]
		if !loaded {
			entries, readErr := queries.LinkTargetElements(ctx, storage.LinkTargetElementsParams{PageID: element.Link.PageID, WorkspaceID: workspaceID})
			if readErr != nil {
				return readErr
			}
			ids = map[string]bool{}
			for _, id := range entries {
				ids[id] = true
			}
			targets[element.Link.PageID] = ids
		}
		if !ids[element.Link.ElementID] {
			return invalid("Choose an existing target element in this workspace.")
		}
	}

	return nil
}
