package release_test

import (
	"mindflow/internal/release"
	"testing"
)

func TestSemanticVersionOrdering(t *testing.T) {
	t.Parallel()
	versions := []string{"v1.0.0-alpha", "v1.0.0-alpha.1", "v1.0.0-alpha.2", "v1.0.0-alpha.10", "v1.0.0-beta", "v1.0.0", "v1.2.0", "v1.10.0", "v2.0.0"}
	for index, left := range versions {
		if !release.Valid(left) {
			t.Fatal("invalid release", left)
		}
		for other, right := range versions {
			want := 0
			if index < other {
				want = -1
			} else if index > other {
				want = 1
			}
			if release.Compare(left, right) != want {
				t.Fatal("incorrect comparison", left, right)
			}
		}
	}
	for _, invalid := range []string{"dev", "v01.0.0", "v1.0.0-01", "v1.0.0-alpha..1", "v1.0.0/other", "v1.0.0\n"} {
		if release.Valid(invalid) {
			t.Fatal("accepted invalid version", invalid)
		}
	}
}
