package images

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"micro-front/internal/validate"
)

var errImageBlogNotFound = errors.New("blog not found")
var errImageUploadFailed = errors.New("image upload failed")

// Upload は設計書 3.8 の画像アップロード処理を行います。
func (uc Usecase) Upload(ctx context.Context, blogID int64, altText string, file multipart.File) (ImagesUploadResponse, string, map[string]string, error) {
	if _, err := uc.Store.GetBlog(ctx, blogID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ImagesUploadResponse{}, "", nil, errImageBlogNotFound
		}
		return ImagesUploadResponse{}, "", nil, err
	}
	if validate.Length(altText) > 200 {
		return ImagesUploadResponse{}, "VALIDATION_ERROR", map[string]string{
			"alt_text": "代替テキストは200文字以内で入力してください。",
		}, nil
	}

	src, _, err := image.Decode(file)
	if err != nil {
		return ImagesUploadResponse{}, "INVALID_IMAGE_FILE", map[string]string{
			"file": "画像ファイルを選択してください。",
		}, nil
	}

	record, err := uc.Store.CreateImage(ctx, blogID, altText)
	if err != nil {
		return ImagesUploadResponse{}, "", nil, err
	}

	if err := savePNG(uc.DataDir, blogID, record.ID, src); err != nil {
		_ = uc.Store.DeleteImage(ctx, blogID, record.ID)
		return ImagesUploadResponse{}, "", nil, errImageUploadFailed
	}

	return ImagesUploadResponse{
		Result:  "success",
		URL:     fmt.Sprintf("/admin/images/%d/%d.png", blogID, record.ID),
		AltText: altText,
	}, "", nil, nil
}

// List は記事に紐づく画像一覧を返します。
func (uc Usecase) List(ctx context.Context, blogID int64) (ImagesListResponse, error) {
	if _, err := uc.Store.GetBlog(ctx, blogID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ImagesListResponse{}, errImageBlogNotFound
		}
		return ImagesListResponse{}, err
	}

	images, err := uc.Store.ListImagesByBlog(ctx, blogID)
	if err != nil {
		return ImagesListResponse{}, err
	}

	items := make([]ImagesListItemResponse, 0, len(images))
	for _, img := range images {
		items = append(items, ImagesListItemResponse{
			ID:        img.ID,
			BlogID:    img.BlogID,
			URL:       fmt.Sprintf("/admin/images/%d/%d.png", blogID, img.ID),
			AltText:   img.AltText,
			CreatedAt: img.CreatedAt,
			UpdatedAt: img.UpdatedAt,
		})
	}

	return ImagesListResponse{Items: items}, nil
}

// Delete は設計書 3.9 の画像削除処理を行います。
func (uc Usecase) Delete(ctx context.Context, blogID, imageID int64) (ImagesDeleteResponse, error) {
	if err := uc.Store.DeleteImage(ctx, blogID, imageID); err != nil {
		return ImagesDeleteResponse{}, err
	}
	_ = os.Remove(filepath.Join(uc.DataDir, "images", strconv.FormatInt(blogID, 10), strconv.FormatInt(imageID, 10)+".png"))
	return ImagesDeleteResponse{
		ID:     imageID,
		BlogID: blogID,
		Result: "deleted",
	}, nil
}

// AdminImagePath は管理画面で参照する画像ファイルの保存先を解決します。
func (uc Usecase) AdminImagePath(blogID int64, imageName string) (string, error) {
	if !strings.HasSuffix(imageName, ".png") {
		return "", errors.New("not found")
	}
	imageID, err := strconv.ParseInt(strings.TrimSuffix(imageName, ".png"), 10, 64)
	if err != nil {
		return "", errors.New("not found")
	}
	return filepath.Join(uc.DataDir, "images", strconv.FormatInt(blogID, 10), strconv.FormatInt(imageID, 10)+".png"), nil
}

// savePNG は画像を最大 1920x1080 に収めて PNG 保存します。
func savePNG(dataDir string, blogID, imageID int64, src image.Image) error {
	dst := resizeToFit(src, 1920, 1080)
	dir := filepath.Join(dataDir, "images", strconv.FormatInt(blogID, 10))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, strconv.FormatInt(imageID, 10)+".png"))
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, dst)
}

// resizeToFit は元画像の縦横比を保ったまま指定サイズ以内に縮小します。
func resizeToFit(src image.Image, maxW, maxH int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxW && h <= maxH {
		dst := image.NewRGBA(b)
		draw.Draw(dst, b, src, b.Min, draw.Src)
		return dst
	}
	scaleW := float64(maxW) / float64(w)
	scaleH := float64(maxH) / float64(h)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}
	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			sx := b.Min.X + int(float64(x)/float64(newW)*float64(w))
			sy := b.Min.Y + int(float64(y)/float64(newH)*float64(h))
			if sx >= b.Max.X {
				sx = b.Max.X - 1
			}
			if sy >= b.Max.Y {
				sy = b.Max.Y - 1
			}
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
