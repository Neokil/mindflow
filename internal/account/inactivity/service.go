// Package inactivity manages reminders and permanent deletion of dormant accounts.
package inactivity

import (
	"context"
	"crypto/rand"
	"fmt"
	"mindflow/internal/account"
	"mindflow/internal/account/storage"
	"time"
)

// Mail supplies transactional notices and successful-delivery receipts.
type Mail interface {
	QueueWithin(ctx context.Context, transaction *storage.Transaction, userID, purpose, to, subject, body, tokenHash string) error
	DeliveredWithin(ctx context.Context, transaction *storage.Transaction, userID, purpose string) (int, error)
}

// Configuration supplies the public application address.
type Configuration interface {
	PublicAddress(ctx context.Context) (string, error)
}

// Service owns the inactivity timeline. Administrators are always exempt.
type Service struct {
	repository    *storage.Repository
	accounts      *account.Service
	mail          Mail
	configuration Configuration
	now           func() time.Time
}

// NewService supplies account deletion, delivery and a testable clock.
func NewService(repository *storage.Repository, accounts *account.Service, mail Mail, configuration Configuration, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}

	return &Service{repository: repository, accounts: accounts, mail: mail, configuration: configuration, now: now}
}

// Process advances pending notices and deletes only accounts that received both reminders.
func (a *Service) Process(ctx context.Context) error {
	now := int(a.now().Unix())
	candidates, err := a.repository.ActivityCandidates(ctx, now-inactiveDays*day)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return nil
	}
	address, err := a.configuration.PublicAddress(ctx)
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		err = a.processAccount(ctx, candidate.UserID, address, now)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *Service) processAccount(ctx context.Context, userID, address string, now int) error {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	row, err := queries.ActivityFind(ctx, userID)
	if storage.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if row.Role != account.RoleUser || row.Status != account.StatusActive || row.LastSeen > now-inactiveDays*day {
		return nil
	}
	switch {
	case row.Cycle == "":
		cycle := rand.Text()
		err = queries.ActivityWarning(ctx, storage.ActivityWarningParams{UserID: userID, Cycle: cycle, QueuedAt: now})
		if err != nil {
			return err
		}
		err = a.queue(ctx, transaction, userID, row.Email, address, "warning", cycle, graceDays)
	case row.Deadline == 0:
		delivered, deliveryErr := a.mail.DeliveredWithin(ctx, transaction, userID, "inactivity-warning-"+row.Cycle)
		if deliveryErr != nil {
			return deliveryErr
		}
		if delivered > 0 {
			err = queries.ActivityDeadline(ctx, storage.ActivityDeadlineParams{UserID: userID, Deadline: delivered + graceDays*day})
		} else if now-row.QueuedAt >= day {
			err = queries.ActivityWarning(ctx, storage.ActivityWarningParams{UserID: userID, Cycle: row.Cycle, QueuedAt: now})
			if err != nil {
				return err
			}
			err = a.queue(ctx, transaction, userID, row.Email, address, "warning", row.Cycle, graceDays)
		}
	case row.FinalQueuedAt == 0 && now >= row.Deadline-finalDays*day:
		err = queries.ActivityFinalQueued(ctx, storage.ActivityFinalQueuedParams{UserID: userID, FinalQueuedAt: now})
		if err != nil {
			return err
		}
		err = a.queue(ctx, transaction, userID, row.Email, address, "final", row.Cycle, finalDays)
	case row.FinalQueuedAt > 0 && row.FinalDeadline == 0:
		delivered, deliveryErr := a.mail.DeliveredWithin(ctx, transaction, userID, "inactivity-final-"+row.Cycle)
		if deliveryErr != nil {
			return deliveryErr
		}
		if delivered > 0 {
			err = queries.ActivityFinalDeadline(ctx, storage.ActivityFinalDeadlineParams{UserID: userID, FinalDeadline: max(row.Deadline, delivered+finalDays*day)})
		} else if now-row.FinalQueuedAt >= day {
			err = queries.ActivityFinalQueued(ctx, storage.ActivityFinalQueuedParams{UserID: userID, FinalQueuedAt: now})
			if err != nil {
				return err
			}
			err = a.queue(ctx, transaction, userID, row.Email, address, "final", row.Cycle, finalDays)
		}
	case row.FinalDeadline > 0 && now >= row.FinalDeadline:
		err = a.accounts.DeleteWithin(ctx, transaction, userID)
	}
	if err != nil {
		return err
	}

	err = transaction.Commit()

	return wrapCommit(err)
}

func (a *Service) queue(ctx context.Context, transaction *storage.Transaction, userID, to, address, stage, cycle string, days int) error {
	body := fmt.Sprintf("You haven't used mindflow for more than 90 days. Sign in within %d days to keep your account and workspaces. Otherwise your account and all owned content will be permanently deleted.\n\nSign in to mindflow\t%s/", days, address)

	return a.mail.QueueWithin(ctx, transaction, userID, "inactivity-"+stage+"-"+cycle, to, "Keep your mindflow account", body, "")
}

const (
	day          = 86400
	inactiveDays = 90
	graceDays    = 30
	finalDays    = 7
)

func wrapCommit(err error) error {
	if err != nil {
		return fmt.Errorf("advance inactivity timeline: %w", err)
	}

	return nil
}
