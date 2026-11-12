package authentication_test

import (
	"context"
	"errors"
	"mindflow/internal/authentication"
	"mindflow/internal/fault"
	"testing"
)

func TestAdministrationRequiresPrivilegeBeforePersistence(t *testing.T) {
	t.Parallel()

	service := &authentication.Service{}
	actor := authentication.Principal{User: authentication.User{Role: "user", Status: "active"}}
	_, err := service.Invite(context.Background(), actor, authentication.InviteCommand{Email: "friend@example.com"})
	var f *fault.Error
	if !errors.As(err, &f) || f.Code != "forbidden" {
		t.Fatal("non-administrator invitation allowed")
	}
}
