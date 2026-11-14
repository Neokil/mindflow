package email_test

import (
	"mindflow/internal/email"
	emailstorage "mindflow/internal/email/storage"
	"mindflow/internal/mindflow/storage"
	"path/filepath"
	"testing"
	"time"
)

func TestCancelledMailCannotRecreateAccountDeliveryReceipts(t *testing.T) {
	t.Parallel()
	data, err := storage.OpenData(t.Context(), filepath.Join(t.TempDir(), "mail.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = data.Close() })
	outbox := email.NewOutbox(data.Email)
	transaction, err := data.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = transaction.Rollback() }()
	err = outbox.QueueWithin(t.Context(), transaction, "deleted-account", "inactivity-warning-test", "user@example.com", "Reminder", "Sign in", "")
	if err != nil {
		t.Fatal(err)
	}
	err = transaction.Commit()
	if err != nil {
		t.Fatal(err)
	}
	job, err := data.Email.EmailDueMail(t.Context(), time.Now().Add(time.Second).Unix())
	if err != nil {
		t.Fatal(err)
	}
	// Simulate account deletion while SMTP delivery is in flight.
	_, err = data.Email.EmailDeleteUserMail(t.Context(), "deleted-account")
	if err != nil {
		t.Fatal(err)
	}
	err = data.Email.EmailRecordDelivery(t.Context(), emailstorage.EmailRecordDeliveryParams{JobID: job.ID, DeliveredAt: time.Now().Unix()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = data.Email.EmailDelivered(t.Context(), "deleted-account:inactivity-warning-test")
	if !emailstorage.IsNotFound(err) {
		t.Fatal("cancelled delivery recreated account-linked data", err)
	}
}
