package workspace

const (
	copyNamePrefixLength   = 150
	defaultGridSpacing     = 20
	elementImage           = "image"
	leaseRenew             = "renew"
	maxElements            = 10000
	maxIdentifierLength    = 128
	maxImageDimension      = 16384
	maxImagePixels         = 40000000
	previewJPEGQuality     = 88
	receiptLifetimeSeconds = 86400
	variantRetrySeconds    = 60
	variantScaleThreshold  = .8
)

const (
	jpegMarker         = 0xff
	jpegStart          = 0xd8
	jpegScan           = 0xda
	jpegEnd            = 0xd9
	jpegEXIF           = 0xe1
	jpegMarkerBytes    = 2
	jpegSegmentBytes   = 4
	exifSignatureBytes = 6
	exifMinimumBytes   = 14
	tiffHeaderBytes    = 8
	byteOrderBytes     = 2
	tiffOffsetPosition = 4
	tiffMagic          = 42
	tiffEntryBytes     = 12
	entryCountBytes    = 2
	entryTypePosition  = 2
	entryCountPosition = 4
	entryValuePosition = 8
	shortBytes         = 2
	orientationTag     = 0x112
	tiffShort          = 3
)

const (
	orientationNormal = iota + 1
	orientationFlipHorizontal
	orientationRotate180
	orientationFlipVertical
	orientationTranspose
	orientationRotate90
	orientationTransverse
	orientationRotate270
)

const workerIntervalSeconds = 2
