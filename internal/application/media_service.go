package application

import (
	"context"

	"mindflow/internal/domain"
)

func (s *Service) AssetMeta(ctx context.Context, actor domain.Principal, id string) (domain.AssetInfo, error) {
	return s.repo.AssetMeta(ctx, actor, id)
}

func (s *Service) RequestVariant(ctx context.Context, actor domain.Principal, id string, v domain.RequestVariantCommand) (Completed, error) {
	return s.repo.RequestVariant(ctx, actor, id, v)
}

func (s *Service) UploadAsset(ctx context.Context, actor domain.Principal, raw []byte) (domain.AssetInfo, error) {
	return s.repo.UploadAsset(ctx, actor, raw)
}

func (s *Service) GetAsset(ctx context.Context, actor domain.Principal, id string, bucket int) (domain.Blob, error) {
	return s.repo.GetAsset(ctx, actor, id, bucket)
}
