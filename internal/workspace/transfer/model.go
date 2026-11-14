// Package transfer owns portable mindflow workspaces and page bundles.
// Persistence belongs to the parent workspace storage package.
package transfer

import (
	"mindflow/internal/workspace"
	"mindflow/internal/workspace/page/canvas"
	"mindflow/internal/workspace/storage"
)

const (
	formatName          = "mindflow"
	formatVersion       = 1
	maxArchiveBytes     = 64 << 20
	maxExpandedBytes    = 128 << 20
	maxManifestBytes    = 32 << 20
	maxAssetBytes       = 50 << 20
	maxPreviewBytes     = 2 << 20
	maxPages            = 100
	maxAssets           = 1000
	maxTemplates        = 100
	maxTemplateElements = 500
	maxTemplateBytes    = 1 << 20
	maxPageBytes        = 16 << 20
	maxElements         = 100000
	maxFiles            = 1 + 2*maxAssets
	manifestPath        = "manifest.json"
	kindPages           = "pages"
	elementImage        = "image"
	elementPDF          = "pdf"
	entriesPerAsset     = 2
	multipartMemory     = 1 << 20
	templateBaseLength  = 180
	mediaPDF            = "application/pdf"
)

// Page is an ordered canvas with a bundle-local identity for element links.
type Page struct {
	ID       string          `json:"id,omitempty"`
	Name     string          `json:"name"`
	Document canvas.Document `json:"document"`
}

// Template is a reusable element snapshot without its server identity.
type Template struct {
	Name     string           `json:"name"`
	Elements []canvas.Element `json:"elements"`
}

// Asset identifies an embedded original and, for PDFs, its PNG preview.
type Asset struct {
	ID      string `json:"id"`
	MIME    string `json:"mime"`
	Preview bool   `json:"preview,omitempty"`
}

// Bundle is the versioned manifest inside a .mindflow ZIP file.
type Bundle struct {
	Format    string     `json:"format"`
	Version   int        `json:"version"`
	Kind      string     `json:"kind"`
	Name      string     `json:"name"`
	Pages     []Page     `json:"pages"`
	Templates []Template `json:"templates"`
	Assets    []Asset    `json:"assets"`
}

// Selection chooses pages in workspace order; an empty list means all pages.
type Selection struct {
	PageIDs          []string `json:"pageIds"`
	IncludeTemplates bool     `json:"includeTemplates"`
}

// Snapshot is a consistent saved-content view used for PDF and image rendering.
type Snapshot struct {
	Name  string `json:"name"`
	Pages []Page `json:"pages"`
}

// Result reports an atomic workspace creation or page append.
type Result struct {
	ID            string   `json:"id"`
	PageID        string   `json:"pageId"`
	PageCount     int      `json:"pageCount"`
	ElementCount  int      `json:"elementCount"`
	TemplateCount int      `json:"templateCount"`
	Warnings      []string `json:"warnings"`
}

func invalid(message string) error {
	return &workspace.Error{Code: "validation_failed", Message: message}
}

func repositoryError(err error) error {
	if storage.IsNotFound(err) {
		return &workspace.Error{Code: "not_found", Message: "This workspace or its content is unavailable."}
	}

	return err
}
func assetPath(resourceID string) string   { return "assets/" + resourceID + ".bin" }
func previewPath(resourceID string) string { return "assets/" + resourceID + ".png" }
