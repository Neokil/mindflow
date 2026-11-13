// Package template owns reusable, workspace-scoped element snapshots.
package template

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/storage"
	"strings"
	"unicode/utf8"
)

// Service uses parent-domain storage and explicit notification delivery.
type Service struct {
	repository *storage.Repository
	publish    func(string, string, string)
}

// NewService constructs workspace template operations.
func NewService(repository *storage.Repository, publish func(string, string, string)) *Service {
	return &Service{repository: repository, publish: publish}
}

// Summary describes a reusable custom element without loading its content.
type Summary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ElementCount int    `json:"elementCount"`
}

// Snapshot contains independent editable elements, not references to source objects.
type Snapshot struct {
	Summary

	Elements []canvas.Element `json:"elements"`
}

// CreateCommand captures a named selection.
type CreateCommand struct {
	Name     string           `json:"name"`
	Elements []canvas.Element `json:"elements"`
}

// List checks current workspace access before returning template metadata.
func (a *Service) List(ctx context.Context, actor session.Principal, workspaceID string) (_ []Summary, err error) {
	defer func() { err = repositoryError(err) }()
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	_, err = queries.RequireWorkspace(ctx, workspaceID, actor.ID, false)
	if err != nil {
		return nil, err
	}
	rows, err := queries.TemplateList(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]Summary, 0, len(rows))
	for _, row := range rows {
		result = append(result, Summary{ID: row.ID, Name: row.Name, ElementCount: row.ElementCount})
	}

	return result, nil
}

// Get returns a snapshot only to a current workspace member.
func (a *Service) Get(ctx context.Context, actor session.Principal, workspaceID, resourceID string) (_ Snapshot, err error) {
	defer func() { err = repositoryError(err) }()
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	_, err = queries.RequireWorkspace(ctx, workspaceID, actor.ID, false)
	if err != nil {
		return Snapshot{}, err
	}
	row, err := queries.TemplateGet(ctx, storage.TemplateGetParams{ID: resourceID, WorkspaceID: workspaceID})
	if err != nil {
		return Snapshot{}, err
	}
	result := Snapshot{Summary: Summary{ID: row.ID, Name: row.Name, ElementCount: row.ElementCount}}
	err = json.Unmarshal([]byte(row.Elements), &result.Elements)

	return result, wrap("decode template", err)
}

// Create validates geometry and image access, then retains a selection atomically.
func (a *Service) Create(ctx context.Context, actor session.Principal, workspaceID string, command CreateCommand) (_ Summary, err error) {
	defer func() { err = repositoryError(err) }()
	name := strings.TrimSpace(command.Name)
	if name == "" || utf8.RuneCountInString(name) > maxNameLength || len(command.Elements) < 1 || len(command.Elements) > maxElements {
		return Summary{}, invalid("Use a name of 1–200 characters and select 1–500 elements.")
	}
	document := canvas.NewDocument()
	document.Elements = command.Elements
	// Inserted copies always start unlocked and independently editable.
	for index := range document.Elements {
		document.Elements[index].Locked = false
		document.Elements[index].GroupID = ""
	}
	err = canvas.ValidateDocument(document)
	if err != nil {
		return Summary{}, err
	}
	encoded, err := json.Marshal(document.Elements)
	if err != nil {
		return Summary{}, wrap("encode template", err)
	}
	if len(encoded) > maxSnapshotBytes {
		return Summary{}, invalid("A template can contain up to 1 MiB of element data.")
	}
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Summary{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	_, err = queries.RequireWorkspace(ctx, workspaceID, actor.ID, true)
	if err != nil {
		return Summary{}, err
	}
	summaries, err := queries.TemplateList(ctx, workspaceID)
	if err != nil {
		return Summary{}, err
	}
	if len(summaries) >= maxTemplates {
		return Summary{}, invalid("A workspace can contain up to 100 templates.")
	}
	count, err := queries.TemplateNameExists(ctx, storage.TemplateNameExistsParams{WorkspaceID: workspaceID, Name: name})
	if err != nil {
		return Summary{}, err
	}
	if count != 0 {
		return Summary{}, invalid("A template with that name already exists.")
	}
	result := Summary{ID: newID(), Name: name, ElementCount: len(document.Elements)}
	err = queries.TemplateCreate(ctx, storage.TemplateCreateParams{ID: result.ID, WorkspaceID: workspaceID, Name: name, Elements: string(encoded), ElementCount: result.ElementCount})
	if err != nil {
		return Summary{}, err
	}
	err = attachAssets(ctx, queries, actor.ID, workspaceID, result.ID, document.Elements)
	if err != nil {
		return Summary{}, err
	}
	err = wrap("commit template", transaction.Commit())
	if err != nil {
		return Summary{}, err
	}
	a.publish(actor.ID, "workspace", workspaceID)

	return result, nil
}

func attachAssets(ctx context.Context, queries *storage.Queries, userID, workspaceID, resourceID string, elements []canvas.Element) error {
	for _, element := range elements {
		if element.Type != "image" {
			continue
		}
		allowed, err := queries.TemplateAssetAllowed(ctx, storage.TemplateAssetAllowedParams{WorkspaceID: workspaceID, AssetID: element.AssetID, UserID: userID})
		if err != nil {
			return err
		}
		if allowed != 1 {
			return invalid("A template image is not available in this workspace. Wait for uploads to finish and retry.")
		}
		err = queries.TemplateAttachAsset(ctx, storage.TemplateAttachAssetParams{TemplateID: resourceID, AssetID: element.AssetID})
		if err != nil {
			return err
		}
	}

	return nil
}

// Remove deletes a reusable snapshot; already inserted instances remain unchanged.
func (a *Service) Remove(ctx context.Context, actor session.Principal, workspaceID, resourceID string) (_ Completed, err error) {
	defer func() { err = repositoryError(err) }()
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return Completed{}, err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	_, err = queries.RequireWorkspace(ctx, workspaceID, actor.ID, true)
	if err != nil {
		return Completed{}, err
	}
	_, err = queries.TemplateGet(ctx, storage.TemplateGetParams{ID: resourceID, WorkspaceID: workspaceID})
	if err != nil {
		return Completed{}, err
	}
	err = queries.TemplateDelete(ctx, storage.TemplateDeleteParams{ID: resourceID, WorkspaceID: workspaceID})
	if err != nil {
		return Completed{}, err
	}
	err = wrap("commit template", transaction.Commit())
	if err != nil {
		return Completed{}, err
	}
	a.publish(actor.ID, "workspace", workspaceID)

	return Completed{OK: true}, nil
}

// CopyWithin copies the private workspace's library and retains its images.
func (a *Service) CopyWithin(ctx context.Context, transaction *storage.Transaction, source, target string) error {
	queries := a.repository.Within(transaction)
	rows, err := queries.TemplateCopies(ctx, source)
	if err != nil {
		return err
	}
	for _, row := range rows {
		resourceID := newID()
		err = queries.TemplateCreate(ctx, storage.TemplateCreateParams{ID: resourceID, WorkspaceID: target, Name: row.Name, Elements: row.Elements, ElementCount: row.ElementCount})
		if err != nil {
			return err
		}
		err = queries.TemplateCopyAssets(ctx, storage.TemplateCopyAssetsParams{TargetID: resourceID, SourceID: row.ID})
		if err != nil {
			return err
		}
	}

	return nil
}

// Completed acknowledges a successful library mutation.
type Completed struct {
	OK bool `json:"ok"`
}

const (
	maxNameLength    = 200
	maxElements      = 500
	maxTemplates     = 100
	maxSnapshotBytes = 1 << 20
	identifierBytes  = 16
)

func newID() string {
	value := make([]byte, identifierBytes)
	_, _ = rand.Read(value)

	return hex.EncodeToString(value)
}
