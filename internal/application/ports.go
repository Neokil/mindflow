package application

import (
	"context"

	"mindflow/internal/domain"
)

type WorkspaceRepository interface {
	ListWorkspaces(ctx context.Context, actor domain.Principal) ([]domain.Workspace, error)
	CreateWorkspace(ctx context.Context, actor domain.Principal, v domain.CreateWorkspaceCommand) (WorkspaceCreated, error)
	RenameWorkspace(ctx context.Context, actor domain.Principal, id string, v domain.RenameWorkspaceCommand) (Completed, error)
	DeleteWorkspace(ctx context.Context, actor domain.Principal, id string, v domain.DeleteWorkspaceCommand) (Completed, error)
	DuplicateWorkspace(ctx context.Context, actor domain.Principal, id string) (Created, error)
	ListPages(ctx context.Context, actor domain.Principal, id string) ([]domain.Page, error)
	CreatePage(ctx context.Context, actor domain.Principal, id string, v domain.CreatePageCommand) (Created, error)
	RenamePage(ctx context.Context, actor domain.Principal, id string, v domain.RenamePageCommand) (Completed, error)
	DeletePage(ctx context.Context, actor domain.Principal, id string, v domain.DeletePageCommand) (Completed, error)
	DuplicatePage(ctx context.Context, actor domain.Principal, id string) (Created, error)
	ReorderPages(ctx context.Context, actor domain.Principal, id string, v domain.ReorderPagesCommand) (Completed, error)
}
type CanvasRepository interface {
	GetDocument(ctx context.Context, actor domain.Principal, id string) (PageSnapshot, error)
	SaveDocument(ctx context.Context, actor domain.Principal, id string, v domain.SaveDocumentCommand) (SaveReceipt, error)
	RecoveryCopy(ctx context.Context, actor domain.Principal, v domain.RecoveryCopyCommand) (Created, error)
	Lease(ctx context.Context, actor domain.Principal, id string, v domain.LeaseCommand) (domain.LeaseResult, error)
	LeaseStatus(ctx context.Context, actor domain.Principal, id string, clientID string) (domain.LeaseResult, error)
}
type IdentityRepository interface {
	Setup(ctx context.Context, actor domain.Principal, v domain.SetupCommand) (Completed, error)
	SetupTestEmail(ctx context.Context, actor domain.Principal, v domain.SetupTestEmailCommand) (Completed, error)
	Login(ctx context.Context, actor domain.Principal, v domain.LoginCommand) (domain.LoginResult, error)
	ListUsers(ctx context.Context, actor domain.Principal) ([]UserDelivery, error)
	Invite(ctx context.Context, actor domain.Principal, v domain.InviteCommand) (Created, error)
	ResendInvite(ctx context.Context, actor domain.Principal, id string) (Completed, error)
	AdminReset(ctx context.Context, actor domain.Principal, id string) (Completed, error)
	Forgot(ctx context.Context, actor domain.Principal, v domain.ForgotCommand) (Completed, error)
	Redeem(ctx context.Context, actor domain.Principal, purpose string, v domain.RedeemCommand) (Completed, error)
	UpdateUser(ctx context.Context, actor domain.Principal, id string, v domain.UpdateUserCommand) (Completed, error)
	DeleteUser(ctx context.Context, actor domain.Principal, id string, v domain.DeleteUserCommand) (Completed, error)
	UpdateSettings(ctx context.Context, actor domain.Principal, v domain.UpdateSettingsCommand) (Completed, error)
}
type MediaRepository interface {
	AssetMeta(ctx context.Context, actor domain.Principal, id string) (domain.AssetInfo, error)
	RequestVariant(ctx context.Context, actor domain.Principal, id string, v domain.RequestVariantCommand) (Completed, error)
}
type IdentityInfrastructure interface {
	SendMail(context.Context, domain.SMTPConfig, string, string, string) error
	Initialized(context.Context) (bool, error)
	Settings(context.Context) (domain.Settings, error)
	Authenticate(context.Context, string) (domain.Principal, error)
	Logout(context.Context, domain.Principal) error
	SetAppearance(context.Context, domain.Principal, string) error
}
type MediaStorage interface {
	UploadAsset(context.Context, domain.Principal, []byte) (domain.AssetInfo, error)
	GetAsset(context.Context, domain.Principal, string, int) (domain.Blob, error)
}
type Repository interface {
	IdentityRepository
	WorkspaceRepository
	CanvasRepository
	MediaRepository
	IdentityInfrastructure
	MediaStorage
}
