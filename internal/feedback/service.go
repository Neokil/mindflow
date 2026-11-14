// Package feedback manages reports private to their authors and administrators.
package feedback

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"image/png"
	"mindflow/internal/account"
	"mindflow/internal/feedback/storage"
	"mindflow/internal/release"
	"strings"
	"time"
	"unicode/utf8"
)

// Accounts supplies display names without access to account tables.
type Accounts interface {
	Find(ctx context.Context, userID string) (account.User, error)
}

// Service owns feedback and author/admin authorization.
type Service struct {
	repository *storage.Repository
	accounts   Accounts
}

// NewService supplies private report storage and public account metadata.
func NewService(repository *storage.Repository, accounts Accounts) *Service {
	return &Service{repository: repository, accounts: accounts}
}

// Report contains user-visible feedback metadata and discussion.
type Report struct {
	ID            string    `json:"id"`
	AuthorID      string    `json:"authorId"`
	AuthorName    string    `json:"authorName"`
	Kind          string    `json:"kind"`
	Title         string    `json:"title"`
	Body          string    `json:"body,omitempty"`
	Status        string    `json:"status"`
	Release       string    `json:"release"`
	ReleaseURL    string    `json:"releaseURL,omitempty"`
	HasScreenshot bool      `json:"hasScreenshot"`
	CreatedAt     int64     `json:"createdAt"`
	UpdatedAt     int64     `json:"updatedAt"`
	Comments      []Message `json:"comments"`
}

// Message is a discussion reply.
type Message struct {
	ID         string `json:"id"`
	AuthorID   string `json:"authorId"`
	AuthorName string `json:"authorName"`
	Body       string `json:"body"`
	CreatedAt  int64  `json:"createdAt"`
}

// CreateCommand supplies a user-authored report and an optional PNG preview.
type CreateCommand struct {
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Screenshot string `json:"screenshot"`
}

// ReviewCommand records an administrative decision and the release fixing a bug.
type ReviewCommand struct {
	Status  string `json:"status"`
	Release string `json:"release"`
}

// Create validates bounded text and normalizes optional screenshot content.
func (a *Service) Create(ctx context.Context, actor account.User, command CreateCommand) (Report, error) {
	if command.Kind != "bug" && command.Kind != "feature" {
		return Report{}, invalid("Choose bug or feature request.")
	}
	title, err := text(command.Title, titleLimit)
	if err != nil {
		return Report{}, err
	}
	body, err := text(command.Body, bodyLimit)
	if err != nil {
		return Report{}, err
	}
	screenshot, err := normalizeScreenshot(command.Screenshot)
	if err != nil {
		return Report{}, err
	}
	resourceID := rand.Text()
	now := time.Now().Unix()
	err = a.repository.FeedbackCreate(ctx, storage.FeedbackCreateParams{ID: resourceID, AuthorID: actor.ID, Kind: command.Kind, Title: title, Body: body, Screenshot: screenshot, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		return Report{}, err
	}

	return a.Find(ctx, actor, resourceID)
}

// List returns only reports the actor is authorized to read.
func (a *Service) List(ctx context.Context, actor account.User) ([]Report, error) {
	rows, err := a.repository.FeedbackList(ctx, storage.FeedbackListParams{AuthorID: actor.ID, Administrator: administrator(actor)})
	if err != nil {
		return nil, err
	}
	result := make([]Report, 0, len(rows))
	for _, row := range rows {
		result = append(result, Report{ID: row.ID, AuthorID: row.AuthorID, AuthorName: a.name(ctx, row.AuthorID), Kind: row.Kind, Title: row.Title, Status: row.Status, Release: row.ReleaseVersion, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}

	return result, nil
}

// Find checks report access before loading its discussion.
func (a *Service) Find(ctx context.Context, actor account.User, resourceID string) (Report, error) {
	row, err := a.repository.FeedbackFind(ctx, storage.FeedbackFindParams{ID: resourceID, AuthorID: actor.ID, Administrator: administrator(actor)})
	if storage.IsNotFound(err) {
		return Report{}, unavailable()
	}
	if err != nil {
		return Report{}, err
	}
	rows, err := a.repository.FeedbackComments(ctx, resourceID)
	if err != nil {
		return Report{}, err
	}
	comments := make([]Message, 0, len(rows))
	for _, entry := range rows {
		comments = append(comments, Message{ID: entry.ID, AuthorID: entry.AuthorID, AuthorName: a.name(ctx, entry.AuthorID), Body: entry.Body, CreatedAt: entry.CreatedAt})
	}
	notes := ""
	if row.ReleaseVersion != "" {
		notes = release.NotesURL(row.ReleaseVersion)
	}

	return Report{ID: row.ID, AuthorID: row.AuthorID, AuthorName: a.name(ctx, row.AuthorID), Kind: row.Kind, Title: row.Title, Body: row.Body, Status: row.Status, Release: row.ReleaseVersion, ReleaseURL: notes, HasScreenshot: row.HasScreenshot != 0, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Comments: comments}, nil
}

// Comment allows an author or administrator to continue a report's discussion.
func (a *Service) Comment(ctx context.Context, actor account.User, resourceID, body string) (Report, error) {
	_, err := a.Find(ctx, actor, resourceID)
	if err != nil {
		return Report{}, err
	}
	body, err = text(body, commentLimit)
	if err != nil {
		return Report{}, err
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Report{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	now := time.Now().Unix()
	err = queries.FeedbackComment(ctx, storage.FeedbackCommentParams{ID: rand.Text(), ReportID: resourceID, AuthorID: actor.ID, Body: body, CreatedAt: now})
	if err != nil {
		return Report{}, err
	}
	err = queries.FeedbackTouch(ctx, storage.FeedbackTouchParams{ID: resourceID, UpdatedAt: now})
	if err != nil {
		return Report{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Report{}, wrap("save feedback comment", err)
	}

	return a.Find(ctx, actor, resourceID)
}

// Review permits administrative decisions; closing a bug requires a tagged release.
func (a *Service) Review(ctx context.Context, actor account.User, resourceID string, command ReviewCommand) (Report, error) {
	err := account.RequireAdministrator(actor)
	if err != nil {
		return Report{}, err
	}
	report, err := a.Find(ctx, actor, resourceID)
	if err != nil {
		return Report{}, err
	}
	switch command.Status {
	case "submitted", "accepted", "denied":
		command.Release = ""
	case "closed":
		if report.Kind != "bug" || !release.Valid(command.Release) {
			return Report{}, invalid("Closing a bug requires its release version, such as v1.2.0.")
		}
	default:
		return Report{}, invalid("Invalid report status.")
	}
	err = a.repository.FeedbackReview(ctx, storage.FeedbackReviewParams{ID: resourceID, Status: command.Status, ReleaseVersion: command.Release, UpdatedAt: time.Now().Unix()})
	if err != nil {
		return Report{}, err
	}

	return a.Find(ctx, actor, resourceID)
}

// Screenshot returns a normalized PNG only to the author or an administrator.
func (a *Service) Screenshot(ctx context.Context, actor account.User, resourceID string) ([]byte, error) {
	content, err := a.repository.FeedbackScreenshot(ctx, storage.FeedbackScreenshotParams{ID: resourceID, AuthorID: actor.ID, Administrator: administrator(actor)})
	if storage.IsNotFound(err) || len(content) == 0 {
		return nil, unavailable()
	}

	return content, err
}

func (a *Service) name(ctx context.Context, userID string) string {
	user, err := a.accounts.Find(ctx, userID)
	if err != nil {
		return "Deleted user"
	}

	return user.DisplayName
}

func administrator(actor account.User) int64 {
	if actor.Role == account.RoleAdmin && actor.Status == account.StatusActive {
		return 1
	}

	return 0
}

func text(value string, limit int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > limit {
		return "", invalid("Enter text within the allowed length.")
	}

	return value, nil
}

func normalizeScreenshot(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	if len(value) > screenshotBytes*4/3+4 {
		return nil, invalid("Screenshot must be a PNG up to 5 MB.")
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(data) > screenshotBytes {
		return nil, invalid("Invalid PNG screenshot.")
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > maxDimension || config.Height > maxDimension || config.Width*config.Height > maxPixels {
		return nil, invalid("Screenshot dimensions are too large or invalid.")
	}
	image, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, invalid("Invalid PNG screenshot.")
	}
	var output bytes.Buffer
	err = png.Encode(&output, image)
	if err != nil {
		return nil, wrap("normalize screenshot", err)
	}
	if output.Len() > screenshotBytes {
		return nil, invalid("Screenshot must be a PNG up to 5 MB.")
	}

	return output.Bytes(), nil
}

const (
	titleLimit      = 150
	bodyLimit       = 10000
	commentLimit    = 5000
	screenshotBytes = 5 << 20
	maxDimension    = 8192
	maxPixels       = 16_000_000
)
