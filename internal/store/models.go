package store

import "mindflow/internal/domain"

type SMTPConfig = domain.SMTPConfig
type Settings = domain.Settings
type User = domain.User
type Document = domain.Document
type Element = domain.Element
type CanvasSettings = domain.CanvasSettings
type Workspace = domain.Workspace
type Page = domain.Page
type assetInfo = domain.AssetInfo
type principal = domain.Principal
type leaseResult = domain.LeaseResult

var bad = domain.Invalid
var notFound = domain.NotFound
var normalizeEmail = domain.NormalizeEmail
var validSettings = domain.ValidateSettings
var finite = domain.Finite
var validateDocument = domain.ValidateDocument
var emptyDocument = domain.NewDocument
var validName = domain.ValidateName
var variantBucket = domain.VariantBucket
