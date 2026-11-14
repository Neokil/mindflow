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
	_, err := a.repository.Within(transaction).EmailDeletePurposeMail(ctx, storage.EmailDeletePurposeMailParams{UserID: userID, DedupKey: userID + ":" + purpose})
	if err != nil {
		return err
	}
	payload, err := json.Marshal(mailJob{To: to, Subject: subject, Body: body, TokenHash: tokenHash, UserID: userID, Purpose: purpose})
	if err != nil {
		return wrap("encode credential email", err)
	}
	_, err = a.repository.Within(transaction).EmailQueueMail(ctx, storage.EmailQueueMailParams{ID: newID(), UserID: userID, DedupKey: userID + ":" + purpose, Payload: string(payload), RunAt: time.Now().Unix()})

	return err
}

// QueueWelcomeWithin queues onboarding once account creation succeeds.
func (a *Outbox) QueueWelcomeWithin(ctx context.Context, transaction *storage.Transaction, user account.User, publicURL string) error {
	subject := "Welcome " + user.DisplayName + "!"
	body := "We're happy to see you on board. Here are the most important links to get started:\n\nOpen mindflow\t" + publicURL + "/\nView patch notes\thttps://github.com/Neokil/mindflow/releases\nReport an error\thttps://github.com/Neokil/mindflow/issues"

	return a.QueueWithin(ctx, transaction, user.ID, "welcome", user.Email, subject, body, "")
}
