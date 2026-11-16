package release

import (
	_ "embed"
	"encoding/json"
)

//go:embed notes.json
var bundledNotes []byte

// ParseNotes returns release text only when its embedded tag matches the running version.
func ParseNotes(raw []byte, version string) string {
	var notice struct {
		Version string `json:"version"`
		Body    string `json:"body"`
	}
	if json.Unmarshal(raw, &notice) != nil || notice.Version != version || !Valid(version) {
		return ""
	}

	return notice.Body
}

// BundledNotes reads the release text captured during the tagged image build.
func BundledNotes(version string) string { return ParseNotes(bundledNotes, version) }
