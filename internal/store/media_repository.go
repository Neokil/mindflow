package store

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

	"mindflow/internal/application"
	"mindflow/internal/domain"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func (a *Repositories) AssetMeta(ctx context.Context, actor domain.Principal, id string) (result domain.AssetInfo, err error) {
	defer mapRepositoryError(&err)

	var v assetInfo
	if err := a.DB.QueryRowContext(ctx, "SELECT id,width,height,mime FROM assets WHERE id=? AND owner_id=?", id, actor.ID).Scan(&v.ID, &v.Width, &v.Height, &v.MIME); err != nil {
		return domain.AssetInfo{}, err
	}
	v.Variants = []int{}
	rows, err := a.DB.QueryContext(ctx, "SELECT bucket FROM asset_variants WHERE asset_id=? ORDER BY bucket", v.ID)
	if err != nil {
		return domain.AssetInfo{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var b int
		if err = rows.Scan(&b); err != nil {
			return domain.AssetInfo{}, err
		}
		v.Variants = append(v.Variants, b)
	}
	return v, rows.Err()

}
func (a *Repositories) RequestVariant(ctx context.Context, actor domain.Principal, id string, v domain.RequestVariantCommand) (result application.Completed, err error) {
	defer mapRepositoryError(&err)

	if !finite(v.Width) || !finite(v.Height) || v.Width <= 0 || v.Height <= 0 || v.Width > 1e7 || v.Height > 1e7 {
		return application.Completed{}, bad("Invalid image resolution.")
	}
	return application.Completed{OK: true}, a.queueVariant(ctx, actor.ID, id, v.Width, v.Height)

}
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return 1
	}
	offset := 2
	for offset+4 <= len(data) {
		if data[offset] != 0xff {
			return 1
		}
		marker := data[offset+1]
		if marker == 0xda || marker == 0xd9 {
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
		if size < 2 || offset+2+size > len(data) {
			return 1
		}
		segment := data[offset+4 : offset+2+size]
		offset += 2 + size
		if marker != 0xe1 || len(segment) < 14 || string(segment[:6]) != "Exif\x00\x00" {
			continue
		}
		tiff := segment[6:]
		var order binary.ByteOrder = binary.LittleEndian
		if string(tiff[:2]) == "MM" {
			order = binary.BigEndian
		} else if string(tiff[:2]) != "II" {
			return 1
		}
		if order.Uint16(tiff[2:4]) != 42 {
			return 1
		}
		pos := int(order.Uint32(tiff[4:8]))
		if pos < 8 || pos+2 > len(tiff) {
			return 1
		}
		count := int(order.Uint16(tiff[pos : pos+2]))
		for i := 0; i < count; i++ {
			j := pos + 2 + i*12
			if j+12 > len(tiff) {
				return 1
			}
			if order.Uint16(tiff[j:j+2]) == 0x112 && order.Uint16(tiff[j+2:j+4]) == 3 && order.Uint32(tiff[j+4:j+8]) == 1 {
				v := int(order.Uint16(tiff[j+8 : j+10]))
				if v >= 1 && v <= 8 {
					return v
				}
			}
		}
		return 1
	}
	return 1
}
func orient(src image.Image, o int) image.Image {
	if o == 1 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	ow, oh := w, h
	if o >= 5 {
		ow, oh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, ow, oh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := x, y
			switch o {
			case 2:
				dx = w - 1 - x
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dy = h - 1 - y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
func (a *Repositories) queueVariant(ctx context.Context, owner, id string, width, height float64) error {
	var ow, oh int
	if err := a.DB.QueryRowContext(ctx, "SELECT width,height FROM assets WHERE id=? AND owner_id=?", id, owner).Scan(&ow, &oh); err != nil {
		return err
	}
	bucket := variantBucket(ow, oh, width, height)
	if bucket == 0 {
		return nil
	}
	var n int
	if err := a.DB.QueryRowContext(ctx, "SELECT count(*) FROM asset_variants WHERE asset_id=? AND bucket=?", id, bucket).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	payload, _ := json.Marshal(map[string]int{"bucket": bucket})
	_, err := a.DB.ExecContext(ctx, "INSERT OR IGNORE INTO jobs(id,user_id,asset_id,kind,dedup_key,payload,run_at) VALUES(?,?,?,'variant',?,?,?)", randomID(), owner, id, fmt.Sprintf("variant:%s:%d", id, bucket), string(payload), now())
	return err
}
func (a *Repositories) enqueueImageVariants(ctx context.Context, owner string, d Document) {
	targets := map[string][2]float64{}
	for _, e := range d.Elements {
		if e.Type == "image" {
			target := targets[e.AssetID]
			target[0] = math.Max(target[0], e.Width)
			target[1] = math.Max(target[1], e.Height)
			targets[e.AssetID] = target
		}
	}
	for id, target := range targets {
		_ = a.queueVariant(ctx, owner, id, target[0], target[1])
	}
}
func (a *Repositories) RunVariant(ctx context.Context) {
	a.mediaMu.Lock()
	defer a.mediaMu.Unlock()
	var id, owner, asset, payload string
	var attempts int
	if err := a.DB.QueryRowContext(ctx, "SELECT id,user_id,asset_id,payload,attempts FROM jobs WHERE kind='variant' AND attempts<3 AND run_at<=? ORDER BY run_at LIMIT 1", now()).Scan(&id, &owner, &asset, &payload, &attempts); err != nil {
		return
	}
	var v struct {
		Bucket int `json:"bucket"`
	}
	_ = json.Unmarshal([]byte(payload), &v)
	var raw []byte
	var width, height int
	if err := a.DB.QueryRowContext(ctx, "SELECT original,width,height FROM assets WHERE id=? AND owner_id=?", asset, owner).Scan(&raw, &width, &height); err != nil {
		_, _ = a.DB.ExecContext(ctx, "DELETE FROM jobs WHERE id=?", id)
		return
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err == nil {
		src = orient(src, jpegOrientation(raw))
		scale := float64(v.Bucket) / float64(max(width, height))
		nw, nh := max(1, int(math.Ceil(float64(width)*scale))), max(1, int(math.Ceil(float64(height)*scale)))
		dst := image.NewNRGBA(image.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
		var buffer bytes.Buffer
		mime := "image/png"
		opaque := dst.Opaque()
		if opaque {
			mime = "image/jpeg"
			err = jpeg.Encode(&buffer, dst, &jpeg.Options{Quality: 88})
		} else {
			err = png.Encode(&buffer, dst)
		}
		if err == nil {
			tx, e := a.DB.BeginTx(ctx, nil)
			err = e
			if e == nil {
				defer tx.Rollback()
				var exists int
				e = tx.QueryRow("SELECT count(*) FROM assets WHERE id=? AND owner_id=?", asset, owner).Scan(&exists)
				if e == nil && exists == 1 {
					_, e = tx.Exec("INSERT OR IGNORE INTO asset_variants(asset_id,bucket,mime,data,width,height) VALUES(?,?,?,?,?,?)", asset, v.Bucket, mime, buffer.Bytes(), nw, nh)
				}
				if e == nil {
					_, e = tx.Exec("DELETE FROM jobs WHERE id=?", id)
				}
				if e == nil {
					e = tx.Commit()
				}
				err = e
			}
			if err == nil {
				a.publish(owner, "asset", asset)
				return
			}
		}
	}
	_, _ = a.DB.ExecContext(ctx, "UPDATE jobs SET attempts=attempts+1,run_at=?,last_error='Image preview generation failed.' WHERE id=?", now()+60, id)
}
func (a *Repositories) UploadAsset(ctx context.Context, actor domain.Principal, raw []byte) (domain.AssetInfo, error) {
	a.mediaMu.Lock()
	defer a.mediaMu.Unlock()
	config, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "png" && format != "jpeg" && format != "webp") {
		return domain.AssetInfo{}, bad("Supported image formats are PNG, JPEG, and WebP.")
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > 16384 || config.Height > 16384 || int64(config.Width)*int64(config.Height) > 40000000 {
		return domain.AssetInfo{}, bad("Image dimensions exceed the 40-megapixel or 16,384-pixel limit.")
	}
	if format == "png" && bytes.Contains(raw, []byte("acTL")) || format == "webp" && bytes.Contains(raw, []byte("ANIM")) {
		return domain.AssetInfo{}, bad("Animated images are not supported yet.")
	}
	// Decode once to reject truncated or malformed files before storing them.
	if _, _, err = image.Decode(bytes.NewReader(raw)); err != nil {
		return domain.AssetInfo{}, bad("The image could not be decoded.")
	}
	width, height := config.Width, config.Height
	if jpegOrientation(raw) >= 5 {
		width, height = height, width
	}
	mime := "image/" + format
	hash := sha256.Sum256(raw)
	id := randomID()
	_, err = a.DB.ExecContext(ctx, "INSERT INTO assets(id,owner_id,original,mime,width,height,digest,created_at) VALUES(?,?,?,?,?,?,?,?)", id, actor.ID, raw, mime, width, height, hex.EncodeToString(hash[:]), now())
	if err != nil {
		return domain.AssetInfo{}, err
	}
	return assetInfo{id, width, height, mime, []int{}}, nil

}
func (a *Repositories) GetAsset(ctx context.Context, actor domain.Principal, id string, bucket int) (blob domain.Blob, err error) {
	defer mapRepositoryError(&err)
	var raw []byte
	var mime string
	var width, height int
	if bucket > 0 {
		_ = a.DB.QueryRowContext(ctx, `SELECT v.data,v.mime,v.width,v.height FROM asset_variants v JOIN assets a ON a.id=v.asset_id WHERE a.id=? AND a.owner_id=? AND v.bucket=?`, id, actor.ID, bucket).Scan(&raw, &mime, &width, &height)
	}
	if len(raw) == 0 {
		if err := a.DB.QueryRowContext(ctx, "SELECT original,mime,width,height FROM assets WHERE id=? AND owner_id=?", id, actor.ID).Scan(&raw, &mime, &width, &height); err != nil {
			return blob, err
		}
	}

	return domain.Blob{Data: raw, MIME: mime, Width: width, Height: height}, nil
}
