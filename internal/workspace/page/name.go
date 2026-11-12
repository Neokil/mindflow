package page

import (
	"strings"
	"unicode/utf8"
)

// ValidateName checks a nonempty page or workspace name.
func ValidateName(s string) error {
	if strings.TrimSpace(s) == "" || utf8.RuneCountInString(s) > 200 {
		return invalid("Use a name between 1 and 200 characters.")
	}

	return nil
}
