package comment

import (
	"context"
	"math"
	"mindflow/internal/account"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/storage"
	"strings"
	"time"
	"unicode"
)

// Accounts supplies author profile metadata without cross-domain table access.
type Accounts interface {
	FindWithin(ctx context.Context, transaction *storage.Transaction, resourceID string) (account.User, error)
}

// Service owns page discussions independently of the canvas editor lease.
type Service struct {
	accounts   Accounts
	repository *storage.Repository
	publish    func(string, string, string)
}

// NewService supplies parent-domain storage and event delivery.
func NewService(repository *storage.Repository, accounts Accounts, publish func(string, string, string)) *Service {
	return &Service{repository: repository, accounts: accounts, publish: publish}
}

// Message is an attributed plain-text discussion entry.
type (
	Message struct {
		ID            string `json:"id"`
		AvatarVersion string `json:"avatarVersion"`
		AuthorID      string `json:"authorId"`
		AuthorEmail   string `json:"authorEmail"`
		Body          string `json:"body"`
		CreatedAt     int64  `json:"createdAt"`
	}
	// Thread is a positioned discussion with its caller-specific management permission.
	Thread struct {
		ID            string    `json:"id"`
		AvatarVersion string    `json:"avatarVersion"`
		AuthorID      string    `json:"authorId"`
		AuthorEmail   string    `json:"authorEmail"`
		Initials      string    `json:"initials"`
		X             float64   `json:"x"`
		Y             float64   `json:"y"`
		Resolved      bool      `json:"resolved"`
		CanManage     bool      `json:"canManage"`
		Messages      []Message `json:"messages"`
	}
	// CreateCommand starts a comment at a canvas point.
	CreateCommand struct {
		X    float64 `json:"x"`
		Y    float64 `json:"y"`
		Body string  `json:"body"`
	}
	// ReplyCommand adds plain text to an existing thread.
	ReplyCommand struct {
		Body string `json:"body"`
	}
	// ResolveCommand resolves or reopens an attributed thread.
	ResolveCommand struct {
		Resolved bool `json:"resolved"`
	}
	// Created identifies a newly created comment thread.
	Created struct {
		ID string `json:"id"`
	}
)

func initials(email string) string {
	local, _, _ := strings.Cut(email, "@")
	words := strings.FieldsFunc(local, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	value := []rune{}
	for _, word := range words {
		value = append(value, []rune(word)[0])
		if len(value) == initialsLength {
			break
		}
	}
	if len(value) == 1 {
		word := []rune(words[0])
		if len(word) > 1 {
			value = append(value, word[1])
		}
	}

	if len(value) == 0 {
		return "?"
	}

	return strings.ToUpper(string(value))
}

func text(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 10000 {
		return "", invalid("Write a comment of up to 10,000 characters.")
	}

	return value, nil
}

func accessPage(ctx context.Context, queries *storage.Queries, resourceID, userID string, write bool) (storage.Access, error) {
	access, err := queries.RequirePage(ctx, resourceID, userID, false)
	if storage.IsNotFound(err) {
		return storage.Access{}, notFound()
	}
	if err != nil {
		return storage.Access{}, err
	}
	if write && !storage.CanComment(access.Role) {
		return storage.Access{}, &storage.AccessError{}
	}

	return access, nil
}

// List loads all readable threads with reply authors and management permissions.
func (a *Service) List(ctx context.Context, actor session.Principal, pageID string) ([]Thread, error) {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	access, err := accessPage(ctx, queries, pageID, actor.ID, false)
	if err != nil {
		return nil, err
	}
	rows, err := queries.CommentThreads(ctx, pageID)
	if err != nil {
		return nil, err
	}
	out := make([]Thread, 0, len(rows))
	profiles := make(map[string]string)
	profile := func(resourceID string) (string, error) {
		if version, ok := profiles[resourceID]; ok {
			return version, nil
		}
		user, readErr := a.accounts.FindWithin(ctx, transaction, resourceID)
		if readErr != nil {
			return "", readErr
		}
		profiles[resourceID] = user.AvatarVersion

		return user.AvatarVersion, nil
	}
	for _, row := range rows {
		messages, readErr := queries.CommentMessages(ctx, row.ID)
		if readErr != nil {
			return nil, readErr
		}
		version, readErr := profile(row.AuthorID)
		if readErr != nil {
			return nil, readErr
		}
		thread := Thread{AvatarVersion: version, ID: row.ID, AuthorID: row.AuthorID, AuthorEmail: row.AuthorEmail, Initials: initials(row.AuthorEmail), X: row.X, Y: row.Y, Resolved: row.Resolved != 0, CanManage: actor.ID == row.AuthorID || access.Role == "owner", Messages: make([]Message, 0, len(messages))}
		for _, entry := range messages {
			version, readErr := profile(entry.AuthorID)
			if readErr != nil {
				return nil, readErr
			}
			thread.Messages = append(thread.Messages, Message{AvatarVersion: version, ID: entry.ID, AuthorID: entry.AuthorID, AuthorEmail: entry.AuthorEmail, Body: entry.Body, CreatedAt: entry.CreatedAt})
		}
		out = append(out, thread)
	}

	return out, nil
}

// Create adds a positioned thread without acquiring document editing control.
func (a *Service) Create(ctx context.Context, actor session.Principal, pageID string, command CreateCommand) (Created, error) {
	body, err := text(command.Body)
	if err != nil {
		return Created{}, err
	}
	if math.IsNaN(command.X) || math.IsNaN(command.Y) || math.Abs(command.X) > 1e9 || math.Abs(command.Y) > 1e9 {
		return Created{}, invalid("Invalid comment position.")
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Created{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	_, err = accessPage(ctx, queries, pageID, actor.ID, true)
	if err != nil {
		return Created{}, err
	}
	count, err := queries.CommentCountThreads(ctx, pageID)
	if err != nil {
		return Created{}, err
	}
	if count >= maxThreads {
		return Created{}, invalid("A page can contain at most 1,000 comment threads.")
	}
	resourceID := newID()
	created := time.Now().UnixMilli()
	err = queries.CommentCreateThread(ctx, storage.CommentCreateThreadParams{ID: resourceID, PageID: pageID, AuthorID: actor.ID, AuthorEmail: actor.Email, X: command.X, Y: command.Y, CreatedAt: created})
	if err != nil {
		return Created{}, err
	}
	err = queries.CommentCreateMessage(ctx, storage.CommentCreateMessageParams{ID: resourceID, ThreadID: resourceID, AuthorID: actor.ID, AuthorEmail: actor.Email, Body: body, CreatedAt: created})
	if err != nil {
		return Created{}, err
	}
	err = wrap("commit comment", transaction.Commit())
	if err != nil {
		return Created{}, err
	}
	a.publish(actor.ID, "comment", pageID)

	return Created{ID: resourceID}, nil
}

// Reply adds a discussion entry for comment/full members on an unresolved thread.
func (a *Service) Reply(ctx context.Context, actor session.Principal, resourceID string, command ReplyCommand) (Completed, error) {
	body, err := text(command.Body)
	if err != nil {
		return Completed{}, err
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	thread, err := queries.CommentThread(ctx, resourceID)
	if storage.IsNotFound(err) {
		return Completed{}, notFound()
	}
	if err != nil {
		return Completed{}, err
	}
	_, err = accessPage(ctx, queries, thread.PageID, actor.ID, true)
	if err != nil {
		return Completed{}, err
	}
	if thread.Resolved != 0 {
		return Completed{}, invalid("Reopen this comment before replying.")
	}
	count, err := queries.CommentCountMessages(ctx, resourceID)
	if err != nil {
		return Completed{}, err
	}
	if count >= maxMessages {
		return Completed{}, invalid("This thread has reached its reply limit.")
	}
	err = queries.CommentCreateMessage(ctx, storage.CommentCreateMessageParams{ID: newID(), ThreadID: resourceID, AuthorID: actor.ID, AuthorEmail: actor.Email, Body: body, CreatedAt: time.Now().UnixMilli()})
	if err != nil {
		return Completed{}, err
	}
	err = wrap("commit comment", transaction.Commit())
	if err != nil {
		return Completed{}, err
	}
	a.publish(actor.ID, "comment", thread.PageID)

	return Completed{OK: true}, nil
}

// Manage resolves, reopens or deletes a thread only for its creator or workspace owner.
func (a *Service) Manage(ctx context.Context, actor session.Principal, resourceID string, resolved, remove bool) (Completed, error) {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	thread, err := queries.CommentThread(ctx, resourceID)
	if storage.IsNotFound(err) {
		return Completed{}, notFound()
	}
	if err != nil {
		return Completed{}, err
	}
	access, err := accessPage(ctx, queries, thread.PageID, actor.ID, false)
	if err != nil {
		return Completed{}, err
	}
	if access.Role != "owner" && thread.AuthorID != actor.ID {
		return Completed{}, &storage.AccessError{}
	}
	if remove {
		err = queries.CommentDelete(ctx, resourceID)
	} else {
		value := 0
		if resolved {
			value = 1
		}
		err = queries.CommentResolve(ctx, storage.CommentResolveParams{ID: resourceID, Resolved: value})
	}
	if err != nil {
		return Completed{}, err
	}
	err = wrap("commit comment", transaction.Commit())
	if err != nil {
		return Completed{}, err
	}
	a.publish(actor.ID, "comment", thread.PageID)

	return Completed{OK: true}, nil
}

const (
	initialsLength = 2
	maxThreads     = 1000
	maxMessages    = 500
)
