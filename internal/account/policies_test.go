package account_test

import (
	"mindflow/internal/account"
	"testing"
)

func TestAdministratorAndInvitationRules(t *testing.T) {
	t.Parallel()

	if account.CheckLastAdministrator("admin", "active", "user", "active", 1) == nil {
		t.Fatal("last administrator demoted")
	}
	if account.CheckLastAdministrator("admin", "active", "user", "active", 2) != nil {
		t.Fatal("valid demotion refused")
	}
	_, _, changeAccountErr := account.ChangeAccount("user", "invited", "", "active")
	if changeAccountErr == nil {
		t.Fatal("invitation bypassed")
	}
	role, status, err := account.ChangeAccount("user", "inactive", "admin", "active")
	if err != nil || role != "admin" || status != "active" {
		t.Fatal("valid account update refused")
	}
}
