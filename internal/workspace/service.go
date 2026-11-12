package workspace

import (
	"mindflow/internal/page"
	"mindflow/internal/workspace/storage"

	imageasset "mindflow/internal/image"
)

// NewService constructs workspace operations with explicit page and image collaborators.
// Activity records workspace recency without depending on page or canvas services.
// TouchPage updates the parent recency in a document-save transaction.

// Service owns this domain's behavior and persistence.
type Service struct {
	repository *storage.Repository
	pages      *page.Service
	images     *imageasset.Service
	publish    func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(repository *storage.Repository, pages *page.Service, images *imageasset.Service, publish func(string, string, string)) *Service {
	return &Service{repository: repository, pages: pages, images: images, publish: publish}
}
