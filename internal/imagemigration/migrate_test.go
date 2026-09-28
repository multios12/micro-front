package imagemigration

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"micro-front/internal/store"
)

func TestRunConvertsFilesAndUpdatesURLs(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.New(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	blog, err := st.CreateBlog(ctx, store.BlogEntitty{
		Title: "test", Content: "![image](/admin/images/1/2.png)", Status: "private",
	})
	if err != nil {
		t.Fatal(err)
	}
	imageDir := filepath.Join(dataDir, "images", "1")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(imageDir, "2.png")
	writePNG(t, pngPath)

	result, err := Run(ctx, st, dataDir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Converted != 1 || result.UpdatedRows != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(imageDir, "2.jpg")); err != nil {
		t.Fatalf("JPEG was not created: %v", err)
	}
	if _, err := os.Stat(pngPath); err != nil {
		t.Fatalf("PNG should remain by default: %v", err)
	}
	updated, err := st.GetBlog(ctx, blog.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Content != "![image](/admin/images/1/2.jpg)" {
		t.Fatalf("unexpected content: %s", updated.Content)
	}

	result, err = Run(ctx, st, dataDir, Options{DeleteSource: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Skipped != 1 || result.Deleted != 1 || result.UpdatedRows != 0 {
		t.Fatalf("unexpected second result: %+v", result)
	}
	if _, err := os.Stat(pngPath); !os.IsNotExist(err) {
		t.Fatalf("PNG should be deleted, stat error: %v", err)
	}
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
