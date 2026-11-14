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
	now        func() time.Time
}

// NewService supplies private report storage and public account metadata.
func NewService(repository *storage.Repository, accounts Accounts, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}

	return &Service{repository: repository, accounts: accounts, now: now}
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
	ClosedAt      int64     `json:"closedAt"`
	PurgeAt       int64     `json:"purgeAt"`
	Comments      []Message `json:"comments"`
	Unread        int64     `json:"unread"`
	ReadThrough   int64     `json:"readThrough"`
	Actions       []string  `json:"actions"`
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

// Create validates bounded text and normalizes optional screenshot content.
func (a *Service) Create(ctx context.Context, actor account.User, command CreateCommand) (Report, error) {
	if command.Kind != KindBug && command.Kind != KindFeature {
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
	now := a.now().Unix()
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Report{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	err = a.repository.Within(transaction).FeedbackCreate(ctx, storage.FeedbackCreateParams{ID: resourceID, AuthorID: actor.ID, Kind: command.Kind, Title: title, Body: body, Screenshot: screenshot, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		return Report{}, err
	}

	err = a.repository.EventWithin(ctx, transaction, resourceID, actor.ID, "report", "")
	if err != nil {
		return Report{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Report{}, wrap("create feedback report", err)
	}

	return a.Find(ctx, actor, resourceID)
}

// List returns only reports the actor is authorized to read.
func (a *Service) List(ctx context.Context, actor account.User, all bool, search string) ([]Report, error) {
	search = strings.TrimSpace(search)
	if utf8.RuneCountInString(search) > searchLimit {
		return nil, invalid("Use a search query up to 200 characters.")
	}
	admin := int64(0)
	if all {
		err := account.RequireAdministrator(actor)
		if err != nil {
			return nil, err
		}
		admin = 1
	}
	rows, err := a.repository.FeedbackList(ctx, storage.FeedbackListParams{UserID: actor.ID, Administrator: admin, Search: search})
	if err != nil {
		return nil, err
	}
	result := make([]Report, 0, len(rows))
	for _, row := range rows {
		result = append(result, Report{ID: row.ID, AuthorID: row.AuthorID, AuthorName: a.name(ctx, row.AuthorID), Kind: row.Kind, Title: row.Title, Status: row.Status, Release: row.ReleaseVersion, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, ClosedAt: row.ClosedAt, PurgeAt: purgeAt(row.Status, row.ClosedAt), Unread: row.Unread})
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
	through, err := a.repository.FeedbackThrough(ctx, resourceID)
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

	actions := []string{}
	if administrator(actor) == 1 {
		actions = availableActions(row.Status)
	}

	return Report{Actions: actions, ID: row.ID, AuthorID: row.AuthorID, AuthorName: a.name(ctx, row.AuthorID), Kind: row.Kind, Title: row.Title, Body: row.Body, Status: row.Status, Release: row.ReleaseVersion, ReleaseURL: notes, HasScreenshot: row.HasScreenshot != 0, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, ClosedAt: row.ClosedAt, PurgeAt: purgeAt(row.Status, row.ClosedAt), Comments: comments, ReadThrough: through}, nil
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
	now := a.now().Unix()
	err = a.appendComment(ctx, transaction, resourceID, actor.ID, body, now)
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

// Review enforces state transitions and records the decision and comment atomically.
func (a *Service) Review(ctx context.Context, actor account.User, resourceID string, command ReviewCommand) (Report, error) {
	err := account.RequireAdministrator(actor)
	if err != nil {
		return Report{}, err
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Report{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	report, err := queries.FeedbackFind(ctx, storage.FeedbackFindParams{ID: resourceID, AuthorID: actor.ID, Administrator: 1})
	if storage.IsNotFound(err) {
		return Report{}, unavailable()
	}
	if err != nil {
		return Report{}, err
	}
	decision, err := Decide(report.Status, command)
	if err != nil {
		return Report{}, err
	}
	now := a.now().Unix()
	closedAt := int64(0)
	if decision.Status == StateClosed {
		closedAt = now
	}
	err = queries.FeedbackReview(ctx, storage.FeedbackReviewParams{ID: resourceID, Status: decision.Status, ReleaseVersion: decision.Release, UpdatedAt: now, ClosedAt: closedAt})
	if err != nil {
		return Report{}, err
	}
	err = a.appendComment(ctx, transaction, resourceID, actor.ID, decision.Comment, now)
	if err != nil {
		return Report{}, err
	}
	err = transaction.Commit()
	if err != nil {
		return Report{}, wrap("review feedback report", err)
	}

	return a.Find(ctx, actor, resourceID)
}

// UnreadCounts separates administrative activity from replies on the actor's own reports.
type UnreadCounts struct {
	Reports int64 `json:"reports"`
	Replies int64 `json:"replies"`
}

// Unread returns counts without granting access to other authors' reports.
func (a *Service) Unread(ctx context.Context, actor account.User) (UnreadCounts, error) {
	counts, err := a.repository.FeedbackUnread(ctx, storage.FeedbackUnreadParams{UserID: actor.ID, Administrator: administrator(actor)})

	return UnreadCounts(counts), err
}

// MarkRead advances only this reader's cursor through the snapshot displayed by their client.
func (a *Service) MarkRead(ctx context.Context, actor account.User, resourceID string, through int64) error {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	_, err = queries.FeedbackFind(ctx, storage.FeedbackFindParams{ID: resourceID, AuthorID: actor.ID, Administrator: administrator(actor)})
	if storage.IsNotFound(err) {
		return unavailable()
	}
	if err != nil {
		return err
	}
	latest, err := queries.FeedbackThrough(ctx, resourceID)
	if err != nil {
		return err
	}
	if through < 0 || through > latest {
		return invalid("Invalid read position.")
	}
	err = queries.FeedbackMarkRead(ctx, storage.FeedbackMarkReadParams{ReportID: resourceID, UserID: actor.ID, ThroughEvent: through})
	if err != nil {
		return err
	}

	return wrap("mark feedback read", transaction.Commit())
}

// Screenshot returns a normalized PNG only to the author or an administrator.
func (a *Service) Screenshot(ctx context.Context, actor account.User, resourceID string) ([]byte, error) {
	content, err := a.repository.FeedbackScreenshot(ctx, storage.FeedbackScreenshotParams{ID: resourceID, AuthorID: actor.ID, Administrator: administrator(actor)})
	if storage.IsNotFound(err) || len(content) == 0 {
		return nil, unavailable()
	}

	return content, err
}

// Cleanup removes expired closed reports, including their media and dependent records.
func (a *Service) Cleanup(ctx context.Context) error {
	return a.repository.FeedbackDeleteExpired(ctx, a.now().Add(-reportRetention).Unix())
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

func (a *Service) appendComment(ctx context.Context, transaction *storage.Transaction, reportID, actorID, body string, now int64) error {
	resourceID := rand.Text()
	err := a.repository.Within(transaction).FeedbackComment(ctx, storage.FeedbackCommentParams{ID: resourceID, ReportID: reportID, AuthorID: actorID, Body: body, CreatedAt: now})
	if err != nil {
		return err
	}

	return a.repository.EventWithin(ctx, transaction, reportID, actorID, "comment", resourceID)
}

func purgeAt(state string, closedAt int64) int64 {
	if state != StateClosed {
		return 0
	}

	return closedAt + int64(reportRetention/time.Second)
}

const (
	reportRetention = 30 * 24 * time.Hour
	searchLimit     = 200
)

// Supported report kinds.
const (
	KindBug     = "bug"
	KindFeature = "feature"
)
