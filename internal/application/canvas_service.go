package application

import (
	"context"

	"mindflow/internal/domain"
)

func (s *Service) GetDocument(ctx context.Context, actor domain.Principal, id string) (PageSnapshot, error) {
	return s.repo.GetDocument(ctx, actor, id)
}

func (s *Service) SaveDocument(ctx context.Context, actor domain.Principal, id string, v domain.SaveDocumentCommand) (SaveReceipt, error) {
	if err := domain.ValidateDocument(v.Document); err != nil {
		return SaveReceipt{}, err
	}
	return s.repo.SaveDocument(ctx, actor, id, v)
}

func (s *Service) RecoveryCopy(ctx context.Context, actor domain.Principal, v domain.RecoveryCopyCommand) (Created, error) {
	if err := domain.ValidateDocument(v.Document); err != nil {
		return Created{}, err
	}
	return s.repo.RecoveryCopy(ctx, actor, v)
}

func (s *Service) Lease(ctx context.Context, actor domain.Principal, id string, v domain.LeaseCommand) (domain.LeaseResult, error) {
	return s.repo.Lease(ctx, actor, id, v)
}

func (s *Service) LeaseStatus(ctx context.Context, actor domain.Principal, id string, clientID string) (domain.LeaseResult, error) {
	return s.repo.LeaseStatus(ctx, actor, id, clientID)
}
