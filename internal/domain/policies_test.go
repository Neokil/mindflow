package domain

import (
	"errors"
	"testing"
)

func TestLeaseTakeoverFencesPreviousWriter(t *testing.T) {
	first, _, err := (LeaseState{}).Advance("session-a", LeaseCommand{ClientID: "client-a", Action: "acquire"}, 100)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, changed, err := first.Advance("session-b", LeaseCommand{ClientID: "client-b", Action: "acquire"}, 110)
	if err != nil || changed || unchanged != first {
		t.Fatal("second editor acquired a live lease")
	}
	second, _, err := first.Advance("session-b", LeaseCommand{ClientID: "client-b", Action: "takeover"}, 110)
	if err != nil {
		t.Fatal(err)
	}
	var fault *Error
	if !errors.As(second.CheckSave("session-a", "client-a", first.Epoch, 1, 1, 111), &fault) || fault.Code != "lease_lost" {
		t.Fatal("stale editor was not fenced")
	}
	if err = second.CheckSave("session-b", "client-b", second.Epoch, 1, 1, 111); err != nil {
		t.Fatal(err)
	}
	if err = second.CheckSave("session-b", "client-b", second.Epoch, 1, 2, 111); err == nil {
		t.Fatal("stale revision accepted")
	}
	if err = second.CheckSave("session-b", "client-b", second.Epoch, 1, 1, 171); err == nil {
		t.Fatal("expired lease accepted")
	}
}
func TestAdministratorAndInvitationRules(t *testing.T) {
	if CheckLastAdministrator("admin", "active", "user", "active", 1) == nil {
		t.Fatal("last administrator demoted")
	}
	if CheckLastAdministrator("admin", "active", "user", "active", 2) != nil {
		t.Fatal("valid demotion refused")
	}
	if _, _, err := ChangeAccount("user", "invited", "", "active"); err == nil {
		t.Fatal("invitation bypassed")
	}
	role, status, err := ChangeAccount("user", "inactive", "admin", "active")
	if err != nil || role != "admin" || status != "active" {
		t.Fatal("valid account update refused")
	}
}
