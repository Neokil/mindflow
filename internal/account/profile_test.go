package account_test

import (
	"mindflow/internal/account"
	"strings"
	"testing"
)

func TestDisplayNamePolicy(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"", "   ", "Alice\nSmith", "Alice\x00Smith", strings.Repeat("\u674e", 81)} {
		_, err := account.NormalizeDisplayName(value)
		if err == nil {
			t.Errorf("accepted invalid name %q", value)
		}
	}
	for _, value := range []string{"  Ada Lovelace  ", "Élodie Chen", "\u674e \u660e", strings.Repeat("\u674e", 80)} {
		name, err := account.NormalizeDisplayName(value)
		if err != nil || name != strings.TrimSpace(value) {
			t.Errorf("valid display name %q: %q %v", value, name, err)
		}
	}
}
