package inactivity_test

import (
	"context"
	"mindflow/internal/account"
	"mindflow/internal/account/inactivity"
	accountstorage "mindflow/internal/account/storage"
	"mindflow/internal/mindflow/storage"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mailbox struct {
	queued    map[string]int
	delivered map[string]int
}

func (m *mailbox) QueueWithin(_ context.Context, _ *accountstorage.Transaction, userID, purpose, _, _, _, _ string) error {
	m.queued[userID+":"+purpose]++

	return nil
}

func (m *mailbox) DeliveredWithin(_ context.Context, _ *accountstorage.Transaction, userID, purpose string) (int, error) {
	return m.delivered[userID+":"+purpose], nil
}

type configuration struct{}

func (configuration) PublicAddress(context.Context) (string, error) {
	return "https://mindflow.example", nil
}

func TestRemindersRequireDeliveryBeforeDeletionAndAdministratorsAreExempt(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	data, err := storage.OpenData(ctx, filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = data.Close() })
	accounts, err := account.NewService(data.Accounts, data.Email, data.Workspaces)
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := data.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = transaction.Rollback() }()
	admin, err := accounts.CreateAdministratorWithin(ctx, transaction, "admin@example.com", "Administrator", "unused")
	if err != nil {
		t.Fatal(err)
	}
	user, err := accounts.CreateInvitedWithin(ctx, transaction, "user@example.com", account.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	err = accounts.SetPasswordWithin(ctx, transaction, user.ID, "unused")
	if err != nil {
		t.Fatal(err)
	}
	err = transaction.Commit()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(91 * 24 * time.Hour)
	mail := &mailbox{queued: map[string]int{}, delivered: map[string]int{}}
	service := inactivity.NewService(data.Accounts, accounts, mail, configuration{}, func() time.Time { return now })
	process := func() {
		t.Helper()
		processErr := service.Process(ctx)
		if processErr != nil {
			t.Fatal(processErr)
		}
	}
	process()
	if len(mail.queued) != 1 {
		t.Fatalf("expected one warning, got %v", mail.queued)
	}
	var warning string
	for key := range mail.queued {
		warning = key
	}
	if !strings.HasPrefix(warning, user.ID+":inactivity-warning-") {
		t.Fatal("wrong warning recipient", warning)
	}
	now = now.Add(100 * 24 * time.Hour)
	process()
	_, err = accounts.Find(ctx, user.ID)
	if err != nil {
		t.Fatal("undelivered warning must never cause deletion", err)
	}
	mail.delivered[warning] = int(now.Unix())
	process()
	now = now.Add(23 * 24 * time.Hour)
	process()
	var final string
	for key := range mail.queued {
		if strings.Contains(key, ":inactivity-final-") {
			final = key
		}
	}
	if final == "" {
		t.Fatal("missing final warning")
	}
	now = now.Add(50 * 24 * time.Hour)
	process()
	_, err = accounts.Find(ctx, user.ID)
	if err != nil {
		t.Fatal("undelivered final warning must prevent deletion", err)
	}
	mail.delivered[final] = int(now.Unix())
	process()
	now = now.Add(6 * 24 * time.Hour)
	process()
	_, err = accounts.Find(ctx, user.ID)
	if err != nil {
		t.Fatal("late final delivery must extend grace", err)
	}
	now = now.Add(24 * time.Hour)
	process()
	_, err = accounts.Find(ctx, user.ID)
	if err == nil {
		t.Fatal("account was not deleted")
	}
	_, err = accounts.Find(ctx, admin.ID)
	if err != nil {
		t.Fatal("administrator must be exempt", err)
	}
}
