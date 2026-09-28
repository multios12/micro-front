package imagemigration

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"micro-front/internal/store"
)

var pngImageURLPattern = regexp.MustCompile(`((?:/admin/images/)?\d+/\d+)\.png\b`)

type Options struct {
	DeleteSource bool
	Quality      int
}

type Result struct {
	Converted   int
	Skipped     int
	Deleted     int
	UpdatedRows int64
}

// Run は保存済みPNGをJPEGへ変換し、DB内の管理画像URLを更新します。
// 既に変換済みのファイルは検証してスキップするため、再実行できます。
func Run(ctx context.Context, st *store.Store, dataDir string, opts Options) (Result, error) {
	if opts.Quality == 0 {
		opts.Quality = 85
	}
	if opts.Quality < 1 || opts.Quality > 100 {
		return Result{}, fmt.Errorf("quality must be between 1 and 100")
	}

	result, err := convertFiles(filepath.Join(dataDir, "images"), opts)
	if err != nil {
		return result, err
	}
	updated, err := updateImageURLs(ctx, st)
	result.UpdatedRows = updated
	if err != nil {
		return result, err
	}
	return result, nil
}

func convertFiles(root string, opts Options) (Result, error) {
	var result Result
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return result, nil
	} else if err != nil {
		return result, err
	}

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".png") {
			return nil
		}

		jpegPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".jpg"
		if _, err := os.Stat(jpegPath); err == nil {
			if err := validateJPEG(jpegPath); err != nil {
				return fmt.Errorf("validate existing jpeg %s: %w", jpegPath, err)
			}
			result.Skipped++
		} else if !os.IsNotExist(err) {
			return err
		} else {
			if err := convertPNG(path, jpegPath, opts.Quality); err != nil {
				return fmt.Errorf("convert %s: %w", path, err)
			}
			result.Converted++
		}

		if opts.DeleteSource {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("delete source %s: %w", path, err)
			}
			result.Deleted++
		}
		return nil
	})
	return result, err
}

func convertPNG(sourcePath, destinationPath string, quality int) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	img, format, err := image.Decode(source)
	closeErr := source.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if format != "png" {
		return fmt.Errorf("source format is %s, not png", format)
	}

	tmp, err := os.CreateTemp(filepath.Dir(destinationPath), ".jpeg-migration-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := jpeg.Encode(tmp, img, &jpeg.Options{Quality: quality}); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, destinationPath)
}

func validateJPEG(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, format, err := image.DecodeConfig(f)
	if err != nil {
		return err
	}
	if format != "jpeg" {
		return fmt.Errorf("file format is %s, not jpeg", format)
	}
	return nil
}

func updateImageURLs(ctx context.Context, st *store.Store) (int64, error) {
	tx, err := st.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var updated int64
	rows, err := tx.QueryContext(ctx, `SELECT id, content FROM blogs`)
	if err != nil {
		return 0, err
	}
	type blogUpdate struct {
		id      int64
		content string
	}
	var updates []blogUpdate
	for rows.Next() {
		var item blogUpdate
		if err := rows.Scan(&item.id, &item.content); err != nil {
			rows.Close()
			return 0, err
		}
		replaced := replaceImageURLs(item.content)
		if replaced != item.content {
			item.content = replaced
			updates = append(updates, item)
		}
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, item := range updates {
		res, err := tx.ExecContext(ctx, `UPDATE blogs SET content = ? WHERE id = ?`, item.content, item.id)
		if err != nil {
			return 0, err
		}
		count, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		updated += count
	}

	var description string
	if err := tx.QueryRowContext(ctx, `SELECT site_description FROM site WHERE id = 1`).Scan(&description); err != nil {
		return 0, err
	}
	replaced := replaceImageURLs(description)
	if replaced != description {
		res, err := tx.ExecContext(ctx, `UPDATE site SET site_description = ? WHERE id = 1`, replaced)
		if err != nil {
			return 0, err
		}
		count, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		updated += count
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return updated, nil
}

func replaceImageURLs(value string) string {
	return pngImageURLPattern.ReplaceAllString(value, `${1}.jpg`)
}
