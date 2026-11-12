package application

import (
	"context"

	"mindflow/internal/domain"
)

func (s *Service) ListWorkspaces(ctx context.Context, actor domain.Principal) ([]domain.Workspace, error) {
	return s.repo.ListWorkspaces(ctx, actor)
}

func (s *Service) CreateWorkspace(ctx context.Context, actor domain.Principal, v domain.CreateWorkspaceCommand) (WorkspaceCreated, error) {
	if err := domain.ValidateName(v.Name); err != nil {
		return WorkspaceCreated{}, err
	}
	return s.repo.CreateWorkspace(ctx, actor, v)
}

func (s *Service) RenameWorkspace(ctx context.Context, actor domain.Principal, id string, v domain.RenameWorkspaceCommand) (Completed, error) {
	if err := domain.ValidateName(v.Name); err != nil {
		return Completed{}, err
	}
	return s.repo.RenameWorkspace(ctx, actor, id, v)
}

func (s *Service) DeleteWorkspace(ctx context.Context, actor domain.Principal, id string, v domain.DeleteWorkspaceCommand) (Completed, error) {
	return s.repo.DeleteWorkspace(ctx, actor, id, v)
}

func (s *Service) DuplicateWorkspace(ctx context.Context, actor domain.Principal, id string) (Created, error) {
	return s.repo.DuplicateWorkspace(ctx, actor, id)
}

func (s *Service) ListPages(ctx context.Context, actor domain.Principal, id string) ([]domain.Page, error) {
	return s.repo.ListPages(ctx, actor, id)
}

func (s *Service) CreatePage(ctx context.Context, actor domain.Principal, id string, v domain.CreatePageCommand) (Created, error) {
	if err := domain.ValidateName(v.Name); err != nil {
		return Created{}, err
	}
	return s.repo.CreatePage(ctx, actor, id, v)
}

func (s *Service) RenamePage(ctx context.Context, actor domain.Principal, id string, v domain.RenamePageCommand) (Completed, error) {
	if err := domain.ValidateName(v.Name); err != nil {
		return Completed{}, err
	}
	return s.repo.RenamePage(ctx, actor, id, v)
}

func (s *Service) DeletePage(ctx context.Context, actor domain.Principal, id string, v domain.DeletePageCommand) (Completed, error) {
	return s.repo.DeletePage(ctx, actor, id, v)
}

func (s *Service) DuplicatePage(ctx context.Context, actor domain.Principal, id string) (Created, error) {
	return s.repo.DuplicatePage(ctx, actor, id)
}

func (s *Service) ReorderPages(ctx context.Context, actor domain.Principal, id string, v domain.ReorderPagesCommand) (Completed, error) {
	return s.repo.ReorderPages(ctx, actor, id, v)
}
