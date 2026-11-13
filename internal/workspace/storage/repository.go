package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mindflow/internal/mindflow/storage/transaction"
	"time"
)

// Transaction is the shared transaction handle used by collaborating storage adapters.
// Transaction is the handle shared by collaborating domain storage adapters.
type Transaction = transaction.Transaction

// Repository owns this domain's database queries and transaction binding.
type Repository struct {
	*Queries

	connection *sql.DB
}

// NewRepository binds queries to the supplied SQLite connection.
func NewRepository(connection *sql.DB) *Repository {
	return &Repository{Queries: New(connection), connection: connection}
}

// Begin starts a caller-owned transaction for an atomic domain operation.
func (a *Repository) Begin(ctx context.Context) (*Transaction, error) {
	transaction, err := a.connection.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin workspace transaction: %w", err)
	}

	return transaction, nil
}

// Within binds queries to a transaction shared by cooperating domains.
func (a *Repository) Within(transaction *Transaction) *Queries { return New(transaction) }

// IsNotFound classifies a missing row without exposing driver errors to policies.
func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }

// TouchPage updates parent workspace recency in the save transaction.
func (a *Repository) TouchPage(ctx context.Context, transaction *Transaction, pageID string) error {
	_, err := a.Within(transaction).WorkspaceTouchWorkspace(ctx, WorkspaceTouchWorkspaceParams{UpdatedAt: time.Now().Unix(), ID: pageID})

	return err
}

// PositionWithin updates a page position and reports the number of matching pages.
func (a *Repository) PositionWithin(ctx context.Context, transaction *Transaction, index int, id, workspaceID string) (int64, error) {
	result, err := a.Within(transaction).PagePositionPage(ctx, PagePositionPageParams{Position: index, ID: id, WorkspaceID: workspaceID})

	return affectedRows(result, err)
}

func affectedRows(result sql.Result, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read affected row count: %w", err)
	}

	return count, nil
}

// LeaseRecord represents persisted editor control without SQL null values.
type LeaseRecord struct {
	SessionID, ClientID string
	Epoch, ExpiresAt    int64
}

// Lease reads the current persisted editor control.
func (a *Repository) Lease(ctx context.Context, pageID string) (LeaseRecord, error) {
	row, err := a.EditorPageLease(ctx, pageID)

	return LeaseRecord{SessionID: row.SessionID.String, ClientID: row.ClientID, Epoch: row.Epoch, ExpiresAt: row.ExpiresAt}, err
}

// LeaseWithin reads editor control inside the document transaction.
func (a *Repository) LeaseWithin(ctx context.Context, transaction *Transaction, pageID string) (LeaseRecord, error) {
	row, err := a.Within(transaction).EditorPageLease(ctx, pageID)

	return LeaseRecord{SessionID: row.SessionID.String, ClientID: row.ClientID, Epoch: row.Epoch, ExpiresAt: row.ExpiresAt}, err
}

// SaveLeaseWithin persists the policy result using SQL null for unowned control.
func (a *Repository) SaveLeaseWithin(ctx context.Context, transaction *Transaction, pageID, sessionID, clientID, userID string, epoch, expires int64) error {
	_, err := a.Within(transaction).EditorUpdateLease(ctx, EditorUpdateLeaseParams{HolderUserID: userID, PageID: pageID, SessionID: sql.NullString{String: sessionID, Valid: sessionID != ""}, ClientID: clientID, Epoch: epoch, ExpiresAt: expires})

	return err
}

// RevokeWithin invalidates all editor control in a stopped-instance restore transaction.
func (a *Repository) RevokeWithin(ctx context.Context, transaction *Transaction) error {
	_, err := a.Within(transaction).EditorExpireLeases(ctx)

	return err
}

// OwnedPageWithin reads page ownership in the editor transaction.
func (a *Repository) OwnedPageWithin(ctx context.Context, transaction *Transaction, id, owner string) (EditorOwnedPageRow, error) {
	return a.Within(transaction).EditorOwnedPage(ctx, EditorOwnedPageParams{ID: id, OwnerID: owner})
}

// DeleteWorkspaceAssetsWithin deletes originals referenced only by this workspace.
func (a *Repository) DeleteWorkspaceAssetsWithin(ctx context.Context, transaction *Transaction, owner, id string) (int64, error) {
	result, err := a.Within(transaction).ImageDeleteWorkspaceAssets(ctx, ImageDeleteWorkspaceAssetsParams{OwnerID: owner, WorkspaceID: id, WorkspaceID_2: id})

	return affectedRows(result, err)
}

// DeletePageAssetsWithin deletes originals referenced only by this page.
func (a *Repository) DeletePageAssetsWithin(ctx context.Context, transaction *Transaction, id string) (int64, error) {
	owner, ownerErr := a.Within(transaction).SharingPageOwner(ctx, id)
	if ownerErr != nil {
		return 0, ownerErr
	}
	result, err := a.Within(transaction).ImageDeletePageAssets(ctx, ImageDeletePageAssetsParams{OwnerID: owner, PageID: id, PageID_2: id})

	return affectedRows(result, err)
}

// DeleteOwnerWithin deletes this owner's complete workspace data in the caller transaction.
func (a *Repository) DeleteOwnerWithin(ctx context.Context, transaction *Transaction, ownerID string) error {
	queries := a.Within(transaction)
	err := queries.EditorDeleteUserRequests(ctx, EditorDeleteUserRequestsParams{RequesterID: ownerID, HolderUserID: ownerID})
	if err != nil {
		return err
	}
	err = queries.SharingInvalidateUserLeases(ctx, ownerID)
	if err != nil {
		return err
	}
	err = queries.SharingRemoveUserMemberships(ctx, ownerID)
	if err != nil {
		return err
	}
	err = queries.CommentDeleteUserThreads(ctx, ownerID)
	if err != nil {
		return err
	}
	err = queries.CommentDeleteUserMessages(ctx, ownerID)
	if err != nil {
		return err
	}
	err = queries.WorkspaceDeleteOwnerWorkspaces(ctx, ownerID)
	if err != nil {
		return err
	}

	return queries.WorkspaceDeleteOwnerImages(ctx, ownerID)
}
