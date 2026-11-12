package application

import (
	"context"
	"errors"
	"testing"

	"mindflow/internal/domain"
)

func TestAdministrationRequiresPrivilegeBeforePersistence(t *testing.T) {
	service := New(nil)
	actor := domain.Principal{User: domain.User{Role: "user", Status: "active"}}
	_, err := service.Invite(context.Background(), actor, domain.InviteCommand{Email: "friend@example.com"})
	var fault *domain.Error
	if !errors.As(err, &fault) || fault.Code != "forbidden" {
		t.Fatal("application allowed non-administrator invitation")
	}
}
