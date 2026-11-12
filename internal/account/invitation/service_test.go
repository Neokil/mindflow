package invitation_test

import (
	"context"
	"errors"
	"mindflow/internal/account"
	"mindflow/internal/account/invitation"
	"testing"
)

func TestInvitationRequiresAdministrator(t *testing.T) {
	t.Parallel()
	service := &invitation.Service{}
	actor := account.User{Role: account.RoleUser, Status: account.StatusActive}
	_, err := service.Invite(context.Background(), actor, invitation.InviteCommand{Email: "friend@example.com"})
	problem, ok := errors.AsType[*account.Error](err)
	if !ok || problem.Code != "forbidden" {
		t.Fatal("unprivileged invitation reached persistence")
	}
}
