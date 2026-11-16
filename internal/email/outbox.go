package email

import (
	"context"
	"encoding/json"
	"mindflow/internal/account"
	"mindflow/internal/email/storage"
	"time"
)

// Outbox queues transactional emails without requiring delivery configuration.
type Outbox struct{ repository *storage.Repository }

// NewOutbox supplies the email-owned persistence boundary.
func NewOutbox(repository *storage.Repository) *Outbox { return &Outbox{repository: repository} }

// QueueWithin replaces a superseded message and queues the new credential link atomically.
func (a *Outbox) QueueWithin(ctx context.Context, transaction *storage.Transaction, userID, purpose, to, subject, body, tokenHash string) error {
	return a.queueWithin(ctx, transaction, mailJob{To: to, Subject: subject, Body: body, TokenHash: tokenHash, UserID: userID, Purpose: purpose})
}

// QueueProductUpdateWithin persists explicit release content with its delivery identity.
func (a *Outbox) QueueProductUpdateWithin(ctx context.Context, transaction *storage.Transaction, userID, to string, notice ProductUpdate) error {
	return a.queueWithin(ctx, transaction, mailJob{To: to, Subject: notice.subject(), Body: notice.plainText(), UserID: userID, Purpose: "release-" + notice.Version, Update: &notice})
}

// QueueWelcomeWithin queues onboarding once account creation succeeds.
func (a *Outbox) QueueWelcomeWithin(ctx context.Context, transaction *storage.Transaction, user account.User, publicURL string) error {
	subject := "Welcome " + user.DisplayName + "!"
	body := "We're happy to see you on board. Here are the most important links to get started:\n\nOpen mindflow\t" + publicURL + "/\nLearn how to use mindflow\thttps://github.com/Neokil/mindflow/wiki/user-docs-using-mindflow\nView patchnotes on GitHub\thttps://github.com/Neokil/mindflow/releases\nReport an error or request a feature on GitHub\thttps://github.com/Neokil/mindflow/issues"

	return a.QueueWithin(ctx, transaction, user.ID, "welcome", user.Email, subject, body, "")
}

// DeliveredWithin returns the successful SMTP delivery time, or zero while pending.
func (a *Outbox) DeliveredWithin(ctx context.Context, transaction *storage.Transaction, userID, purpose string) (int, error) {
	value, err := a.repository.Within(transaction).EmailDelivered(ctx, userID+":"+purpose)
	if storage.IsNotFound(err) {
		return 0, nil
	}

	return value, err
}

// CancelUpdatesWithin removes pending release messages after consent is withdrawn.
func (a *Outbox) CancelUpdatesWithin(ctx context.Context, tx *storage.Transaction, userID string) error {
	return a.repository.Within(tx).EmailCancelUpdates(ctx, storage.EmailCancelUpdatesParams{UserID: userID, DedupKey: userID + ":release-%"})
}

func (a *Outbox) queueWithin(ctx context.Context, transaction *storage.Transaction, job mailJob) error {
	userID, purpose := job.UserID, job.Purpose
	_, err := a.repository.Within(transaction).EmailDeletePurposeMail(ctx, storage.EmailDeletePurposeMailParams{UserID: userID, DedupKey: userID + ":" + purpose})
	if err != nil {
		return err
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return wrap("encode credential email", err)
	}
	_, err = a.repository.Within(transaction).EmailQueueMail(ctx, storage.EmailQueueMailParams{ID: newID(), UserID: userID, DedupKey: userID + ":" + purpose, Payload: string(payload), RunAt: time.Now().Unix()})

	return err
}
