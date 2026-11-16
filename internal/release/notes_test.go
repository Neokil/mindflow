package release_test

import (
	"mindflow/internal/release"
	"testing"
)

func TestBundledNotesMatchExactlyTheRunningVersion(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"version":"v1.3.5","body":"## Faster images\n- Reuse cached pictures."}`)
	if release.ParseNotes(raw, "v1.3.5") != "## Faster images\n- Reuse cached pictures." {
		t.Fatal("release text lost")
	}
	for _, version := range []string{"v1.3.4", "v1.4.0", "dev"} {
		if release.ParseNotes(raw, version) != "" {
			t.Fatal("notes from another build were used", version)
		}
	}
	if release.ParseNotes([]byte("invalid JSON"), "v1.3.5") != "" {
		t.Fatal("invalid notes accepted")
	}
}
