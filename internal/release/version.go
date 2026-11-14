// Package release identifies tagged builds and schedules upgrade announcements.
package release

import (
	"cmp"
	"regexp"
	"strings"
)

// Valid reports whether a version is a canonical release tag.
func Valid(version string) bool {
	if !tag.MatchString(version) {
		return false
	}
	_, pre, _ := strings.Cut(version, "-")
	for part := range strings.SplitSeq(pre, ".") {
		if numeric(part) && len(part) > 1 && part[0] == '0' {
			return false
		}
	}

	return true
}

// NotesURL returns the trusted GitHub release page for a tagged build.
func NotesURL(version string) string {
	if Valid(version) {
		return "https://github.com/Neokil/mindflow/releases/tag/" + version
	}

	return "https://github.com/Neokil/mindflow/releases"
}

// Compare compares valid semantic versions, including numeric prerelease identifiers.
func Compare(left, right string) int {
	leftMain, leftPre, _ := strings.Cut(strings.TrimPrefix(left, "v"), "-")
	rightMain, rightPre, _ := strings.Cut(strings.TrimPrefix(right, "v"), "-")
	leftParts, rightParts := strings.Split(leftMain, "."), strings.Split(rightMain, ".")
	for index, part := range leftParts {
		if comparison := compareNumber(part, rightParts[index]); comparison != 0 {
			return comparison
		}
	}
	if leftPre == rightPre {
		return 0
	}
	if leftPre == "" {
		return 1
	}
	if rightPre == "" {
		return -1
	}
	leftParts, rightParts = strings.Split(leftPre, "."), strings.Split(rightPre, ".")
	for index := range min(len(leftParts), len(rightParts)) {
		leftPart, rightPart := leftParts[index], rightParts[index]
		comparison := cmp.Compare(leftPart, rightPart)
		switch {
		case numeric(leftPart) && numeric(rightPart):
			comparison = compareNumber(leftPart, rightPart)
		case numeric(leftPart):
			comparison = -1
		case numeric(rightPart):
			comparison = 1
		}
		if comparison != 0 {
			return comparison
		}
	}

	return cmp.Compare(len(leftParts), len(rightParts))
}

func compareNumber(left, right string) int {
	if len(left) != len(right) {
		return cmp.Compare(len(left), len(right))
	}

	return cmp.Compare(left, right)
}

func numeric(value string) bool {
	if value == "" {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}

	return true
}

var tag = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
