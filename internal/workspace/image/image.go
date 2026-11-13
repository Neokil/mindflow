package image

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"mindflow/internal/account/session"
	"mindflow/internal/workspace/storage"
	"time"

	"golang.org/x/image/draw"

	// Register WebP decoding for original uploads.

	// Register WebP image decoding alongside standard PNG and JPEG support.
	_ "golang.org/x/image/webp"
)

// AssetMeta returns protected image metadata and available preview sizes.
func (a *Service) AssetMeta(ctx context.Context, actor session.Principal, id string) (_ AssetInfo, err error) {
	defer func() { err = repositoryError(err) }()
	var value AssetInfo
	assetMetadataRow, err := a.repository.ImageAssetMetadata(ctx, storage.ImageAssetMetadataParams{ID: id, OwnerID: actor.ID})
	value.ID = assetMetadataRow.ID
	value.Width = assetMetadataRow.Width
	value.Height = assetMetadataRow.Height
	value.MIME = assetMetadataRow.Mime
	if err != nil {
		return AssetInfo{}, err
	}
	value.Variants = []int{}
	rows, err := a.repository.ImageVariantBuckets(ctx, value.ID)
	if err != nil {
		return AssetInfo{}, err
	}
	value.Variants = rows

	return value, nil
}

// RequestVariant queues a bounded display preview for an owned image.
func (a *Service) RequestVariant(ctx context.Context, actor session.Principal, resourceID string, command RequestVariantCommand) (_ Completed, err error) {
	defer func() { err = repositoryError(err) }()

	if !finite(command.Width) || !finite(command.Height) || command.Width <= 0 || command.Height <= 0 || command.Width > 1e7 || command.Height > 1e7 {
		return Completed{}, invalid("Invalid image resolution.")
	}

	return Completed{OK: true}, a.QueueVariant(ctx, actor.ID, resourceID, command.Width, command.Height)
}

func jpegOrientation(data []byte) int {
	if len(data) < jpegSegmentBytes || data[0] != jpegMarker || data[1] != jpegStart {
		return 1
	}
	offset := jpegMarkerBytes
	for offset+jpegSegmentBytes <= len(data) {
		if data[offset] != jpegMarker {
			return 1
		}
		marker := data[offset+1]
		if marker == jpegScan || marker == jpegEnd {
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[offset+jpegMarkerBytes : offset+jpegSegmentBytes]))
		if size < jpegMarkerBytes || offset+jpegMarkerBytes+size > len(data) {
			return 1
		}
		segment := data[offset+jpegSegmentBytes : offset+jpegMarkerBytes+size]
		offset += jpegMarkerBytes + size
		if marker != jpegEXIF || len(segment) < exifMinimumBytes || string(segment[:exifSignatureBytes]) != "Exif\x00\x00" {
			continue
		}

		return tiffOrientation(segment[exifSignatureBytes:])
	}

	return orientationNormal
}

func tiffOrientation(tiff []byte) int {
	if len(tiff) < tiffHeaderBytes {
		return orientationNormal
	}
	var order binary.ByteOrder = binary.LittleEndian
	if string(tiff[:byteOrderBytes]) == "MM" {
		order = binary.BigEndian
	} else if string(tiff[:byteOrderBytes]) != "II" {
		return orientationNormal
	}
	if order.Uint16(tiff[byteOrderBytes:tiffOffsetPosition]) != tiffMagic {
		return orientationNormal
	}
	position := int(order.Uint32(tiff[tiffOffsetPosition:tiffHeaderBytes]))
	if position < tiffHeaderBytes || position+entryCountBytes > len(tiff) {
		return orientationNormal
	}
	count := int(order.Uint16(tiff[position : position+entryCountBytes]))
	for index := range count {
		offset := position + entryCountBytes + index*tiffEntryBytes
		if offset+tiffEntryBytes > len(tiff) {
			return orientationNormal
		}
		tag := order.Uint16(tiff[offset : offset+entryTypePosition])
		fieldType := order.Uint16(tiff[offset+entryTypePosition : offset+entryCountPosition])
		fieldCount := order.Uint32(tiff[offset+entryCountPosition : offset+entryValuePosition])
		if tag != orientationTag || fieldType != tiffShort || fieldCount != 1 {
			continue
		}
		orientation := int(order.Uint16(tiff[offset+entryValuePosition : offset+entryValuePosition+shortBytes]))
		if orientation >= orientationNormal && orientation <= orientationRotate270 {
			return orientation
		}
	}

	return orientationNormal
}

func orient(src image.Image, orientation int) image.Image {
	if orientation == 1 {
		return src
	}
	buffer := src.Bounds()
	writer, height := buffer.Dx(), buffer.Dy()
	ow, oh := writer, height
	if orientation >= orientationTranspose {
		ow, oh = height, writer
	}
	dst := image.NewNRGBA(image.Rect(0, 0, ow, oh))
	for yValue := range height {
		for xValue := range writer {
			dxValue, targetY := xValue, yValue
			switch orientation {
			case orientationFlipHorizontal:
				dxValue = writer - 1 - xValue
			case orientationRotate180:
				dxValue, targetY = writer-1-xValue, height-1-yValue
			case orientationFlipVertical:
				targetY = height - 1 - yValue
			case orientationTranspose:
				dxValue, targetY = yValue, xValue
			case orientationRotate90:
				dxValue, targetY = height-1-yValue, xValue
			case orientationTransverse:
				dxValue, targetY = height-1-yValue, writer-1-xValue
			case orientationRotate270:
				dxValue, targetY = yValue, writer-1-xValue
			}
			dst.Set(dxValue, targetY, src.At(buffer.Min.X+xValue, buffer.Min.Y+yValue))
		}
	}

	return dst
}

// QueueVariant requests a display preview for an owned image.
func (a *Service) QueueVariant(ctx context.Context, owner, resourceID string, width, height float64) error {
	var originalWidth, originalHeight int
	assetDimensionsRow, err := a.repository.ImageAssetDimensions(ctx, storage.ImageAssetDimensionsParams{ID: resourceID, OwnerID: owner})
	if assetDimensionsRow.Mime == "application/pdf" {
		return invalid("PDF previews are created during import.")
	}
	originalWidth = assetDimensionsRow.Width
	originalHeight = assetDimensionsRow.Height
	if err != nil {
		return err
	}
	bucket := VariantBucket(originalWidth, originalHeight, width, height)
	if bucket == 0 {
		return nil
	}
	var count int
	countVariantsRow, err := a.repository.ImageCountVariants(ctx, storage.ImageCountVariantsParams{AssetID: resourceID, Bucket: bucket})
	count = int(countVariantsRow)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	payload, err := json.Marshal(map[string]int{"bucket": bucket})
	if err != nil {
		return wrap("encode variant request", err)
	}
	_, err = a.repository.ImageQueueVariant(ctx, storage.ImageQueueVariantParams{ID: newID(), UserID: assetDimensionsRow.OwnerID, AssetID: resourceID, DedupKey: fmt.Sprintf("variant:%s:%d", resourceID, bucket), Payload: string(payload), RunAt: time.Now().Unix()})

	return err
}

// RunVariant processes one queued image preview.
func (a *Service) RunVariant(ctx context.Context) {
	a.mediaMu.Lock()
	defer a.mediaMu.Unlock()
	job, err := a.repository.ImageDueVariant(ctx, time.Now().Unix())
	if err != nil {
		return
	}
	var request struct {
		Bucket int `json:"bucket"`
	}
	err = json.Unmarshal([]byte(job.Payload), &request)
	if err != nil {
		_, _ = a.repository.ImageDeleteJob(ctx, job.ID)

		return
	}
	original, err := a.repository.ImageOriginalImage(ctx, storage.ImageOriginalImageParams{ID: job.AssetID, OwnerID: job.UserID})
	if err != nil {
		_, _ = a.repository.ImageDeleteJob(ctx, job.ID)

		return
	}
	preview, err := renderPreview(original.Original, original.Width, original.Height, request.Bucket)
	if err == nil {
		err = a.storePreview(ctx, job, request.Bucket, preview)
	}
	if err == nil {
		a.publish(job.UserID, "asset", job.AssetID)

		return
	}
	_, _ = a.repository.ImageRetryVariant(ctx, storage.ImageRetryVariantParams{RunAt: time.Now().Unix() + variantRetrySeconds, ID: job.ID})
}

func renderPreview(raw []byte, width, height, bucket int) (Blob, error) {
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Blob{}, wrap("decode preview source", err)
	}
	source = orient(source, jpegOrientation(raw))
	scale := float64(bucket) / float64(max(width, height))
	targetWidth := max(1, int(math.Ceil(float64(width)*scale)))
	targetHeight := max(1, int(math.Ceil(float64(height)*scale)))
	target := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), draw.Over, nil)
	var buffer bytes.Buffer
	mime := "image/png"
	if target.Opaque() {
		mime = "image/jpeg"
		err = jpeg.Encode(&buffer, target, &jpeg.Options{Quality: previewJPEGQuality})
	} else {
		err = png.Encode(&buffer, target)
	}
	if err != nil {
		return Blob{}, wrap("encode preview", err)
	}

	return Blob{Data: buffer.Bytes(), MIME: mime, Width: targetWidth, Height: targetHeight}, nil
}

func (a *Service) storePreview(ctx context.Context, job storage.ImageDueVariantRow, bucket int, preview Blob) error {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return wrap("begin preview transaction", err)
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	exists, err := queries.ImageCountOwnedAsset(ctx, storage.ImageCountOwnedAssetParams{ID: job.AssetID, OwnerID: job.UserID})
	if err != nil {
		return err
	}
	if exists == 1 {
		_, err = queries.ImageCreateVariant(ctx, storage.ImageCreateVariantParams{AssetID: job.AssetID, Bucket: bucket, Mime: preview.MIME, Data: preview.Data, Width: preview.Width, Height: preview.Height})
		if err != nil {
			return err
		}
	}
	_, err = queries.ImageDeleteJob(ctx, job.ID)
	if err != nil {
		return err
	}

	return wrap("commit preview", transaction.Commit())
}

// UploadAsset validates and stores the original image for the caller.
func (a *Service) UploadAsset(ctx context.Context, actor session.Principal, raw []byte) (AssetInfo, error) {
	return a.UploadWorkspaceAsset(ctx, actor, "", raw)
}

// UploadWorkspaceAsset retains originals under the target workspace owner's lifetime.
func (a *Service) UploadWorkspaceAsset(ctx context.Context, actor session.Principal, workspaceID string, raw []byte) (_ AssetInfo, err error) {
	defer func() { err = repositoryError(err) }()
	original, err := a.Prepare(raw)
	if err != nil {
		return AssetInfo{}, err
	}

	return a.storePrepared(ctx, actor, workspaceID, original)
}

// PreparedImage contains validated original bytes, ready for an atomic workspace import.
type PreparedImage struct {
	raw           []byte
	width, height int
	mime          string
}

// Prepare validates an original image without writing it to storage.
func (a *Service) Prepare(raw []byte) (PreparedImage, error) {
	a.mediaMu.Lock()
	defer a.mediaMu.Unlock()
	if len(raw) > maxOriginalBytes {
		return PreparedImage{}, invalid("Choose an image up to 20 MiB.")
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "png" && format != "jpeg" && format != "webp") {
		return PreparedImage{}, invalid("Supported image formats are PNG, JPEG, and WebP.")
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxImageDimension || config.Height > maxImageDimension || int64(config.Width)*int64(config.Height) > maxImagePixels {
		return PreparedImage{}, invalid("Image dimensions exceed the 40-megapixel or 16,384-pixel limit.")
	}
	if format == "png" && bytes.Contains(raw, []byte("acTL")) || format == "webp" && bytes.Contains(raw, []byte("ANIM")) {
		return PreparedImage{}, invalid("Animated images are not supported yet.")
	}
	// Decode once to reject truncated or malformed files before storing them.
	_, _, err = image.Decode(bytes.NewReader(raw))
	if err != nil {
		return PreparedImage{}, invalid("The image could not be decoded.")
	}
	width, height := config.Width, config.Height
	if jpegOrientation(raw) >= orientationTranspose {
		width, height = height, width
	}

	return PreparedImage{raw: raw, width: width, height: height, mime: "image/" + format}, nil
}

func (a *Service) storePrepared(ctx context.Context, actor session.Principal, workspaceID string, original PreparedImage) (AssetInfo, error) {
	raw, width, height, mime := original.raw, original.width, original.height, original.mime

	hash := sha256.Sum256(raw)
	resourceID := newID()
	err := a.repository.CreateScopedAsset(ctx, actor.ID, workspaceID, storage.ImageCreateAssetParams{ID: resourceID, OwnerID: actor.ID, Original: raw, Mime: mime, Width: width, Height: height, Digest: hex.EncodeToString(hash[:]), CreatedAt: time.Now().Unix()})
	if err != nil {
		return AssetInfo{}, err
	}

	return AssetInfo{ID: resourceID, Width: width, Height: height, MIME: mime, Variants: []int{}}, nil
}

// GetAsset loads an owned preview or falls back to its original.
func (a *Service) GetAsset(ctx context.Context, actor session.Principal, resourceID string, bucket int) (_ Blob, err error) {
	defer func() { err = repositoryError(err) }()
	var raw []byte
	var mime string
	var width, height int
	if bucket > 0 {
		imageVariantRow, _ := a.repository.ImageImageVariant(ctx, storage.ImageImageVariantParams{ID: resourceID, OwnerID: actor.ID, Bucket: bucket})
		raw = imageVariantRow.Data
		mime = imageVariantRow.Mime
		width = imageVariantRow.Width
		height = imageVariantRow.Height
	}
	if len(raw) == 0 {
		getImageRow, err := a.repository.ImageGetImage(ctx, storage.ImageGetImageParams{ID: resourceID, OwnerID: actor.ID})
		raw = getImageRow.Original
		mime = getImageRow.Mime
		width = getImageRow.Width
		height = getImageRow.Height
		if err != nil {
			return Blob{}, err
		}
	}

	return Blob{Data: raw, MIME: mime, Width: width, Height: height}, nil
}

// StoreWithin stores a validated original in the caller's workspace transaction.
func (a *Service) StoreWithin(ctx context.Context, transaction *storage.Transaction, ownerID, workspaceID string, original PreparedImage) (AssetInfo, error) {
	resourceID := newID()
	hash := sha256.Sum256(original.raw)
	err := a.repository.CreateImportedAssetWithin(ctx, transaction, workspaceID, storage.ImageCreateAssetParams{ID: resourceID, OwnerID: ownerID, Original: original.raw, Mime: original.mime, Width: original.width, Height: original.height, Digest: hex.EncodeToString(hash[:]), CreatedAt: time.Now().Unix()})
	if err != nil {
		return AssetInfo{}, err
	}

	return AssetInfo{ID: resourceID, Width: original.width, Height: original.height, MIME: original.mime, Variants: []int{}}, nil
}

const maxOriginalBytes = 20 << 20
