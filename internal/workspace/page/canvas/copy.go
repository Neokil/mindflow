package canvas

// PageCopy identifies source and destination canvases for a coordinated copy.
type PageCopy struct {
	SourceID string
	TargetID string
	Document Document
}

// ClonePages assigns fresh element/group identities and retargets links among copied pages.
// External targets survive only when copying within the same workspace.
func ClonePages(pages []PageCopy, keepExternal bool) []Document {
	targets := map[ElementLink]ElementLink{}
	result := make([]Document, len(pages))
	for index, entry := range pages {
		result[index] = entry.Document
		result[index].Elements = append([]Element{}, entry.Document.Elements...)
		groups := map[string]string{}
		for elementIndex, element := range result[index].Elements {
			id := newID()
			targets[ElementLink{PageID: entry.SourceID, ElementID: element.ID}] = ElementLink{PageID: entry.TargetID, ElementID: id}
			result[index].Elements[elementIndex].ID = id
			if element.GroupID != "" {
				if groups[element.GroupID] == "" {
					groups[element.GroupID] = newID()
				}
				result[index].Elements[elementIndex].GroupID = groups[element.GroupID]
			}
		}
	}
	for _, document := range result {
		RemapLinks(document.Elements, targets, keepExternal)
	}

	return result
}

// RemapLinks replaces copied targets and optionally removes links outside the copy.
func RemapLinks(elements []Element, targets map[ElementLink]ElementLink, keepExternal bool) {
	for elementIndex, element := range elements {
		if element.Link == nil {
			continue
		}
		if target, ok := targets[*element.Link]; ok {
			elements[elementIndex].Link = &target
		} else if !keepExternal {
			elements[elementIndex].Link = nil
		}
	}
}
