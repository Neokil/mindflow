package workspace

import (
	"mindflow/internal/workspace/page"
	"mindflow/internal/workspace/storage"
	"mindflow/internal/workspace/template"

	imageasset "mindflow/internal/workspace/image"
)

// NewService constructs workspace operations with explicit page and image collaborators.
// Activity records workspace recency without depending on page or canvas services.
// TouchPage updates the parent recency in a document-save transaction.

// Service owns this domain's behavior and persistence.
type Service struct {
	repository *storage.Repository
	pages      *page.Service
	images     *imageasset.Service
	templates  *template.Service
	publish    func(string, string, string)
}

// NewService constructs the domain service with explicit collaborators.
func NewService(repository *storage.Repository, pages *page.Service, images *imageasset.Service, templates *template.Service, publish func(string, string, string)) *Service {
	return &Service{repository: repository, pages: pages, images: images, templates: templates, publish: publish}
}
