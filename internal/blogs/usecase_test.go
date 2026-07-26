package blogs

import (
	"context"
	"testing"

	"micro-front/internal/store"
)

func TestCreateAboutUsesFixedID(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	st, err := store.New(dataDir)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	uc := Usecase{Store: st, DataDir: dataDir}
	got, code, fields, err := uc.Create(ctx, BlogsCreateRequest{
		Title:       "about",
		Content:     "about body",
		Category:    "",
		Status:      "private",
		PublishedAt: "2026-04-19",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if code != "" || len(fields) > 0 {
		t.Fatalf("Create validation unexpectedly failed: code=%q fields=%v", code, fields)
	}
	if got.ID != aboutBlogID {
		t.Fatalf("Create returned id=%d want %d", got.ID, aboutBlogID)
	}

	byTitle, err := st.GetBlogByTitle(ctx, "about")
	if err != nil {
		t.Fatalf("GetBlogByTitle: %v", err)
	}
	if byTitle.ID != aboutBlogID {
		t.Fatalf("stored about id=%d want %d", byTitle.ID, aboutBlogID)
	}
}

func TestUpdateAboutCreatesWhenMissing(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	st, err := store.New(dataDir)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	uc := Usecase{Store: st, DataDir: dataDir}
	got, code, fields, err := uc.Update(ctx, aboutBlogID, BlogsUpdateRequest{
		Title:       "ignored title",
		Content:     "profile body",
		Category:    "ignored-category",
		Status:      "public",
		PublishedAt: "",
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if code != "" || len(fields) > 0 {
		t.Fatalf("Update validation unexpectedly failed: code=%q fields=%v", code, fields)
	}
	if got.ID != aboutBlogID {
		t.Fatalf("Update returned id=%d want %d", got.ID, aboutBlogID)
	}
	if got.Title != "about" {
		t.Fatalf("Update title=%q want about", got.Title)
	}
	if got.Category != "" {
		t.Fatalf("Update category=%q want empty", got.Category)
	}

	loaded, err := uc.Get(ctx, aboutBlogID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if loaded.ID != aboutBlogID {
		t.Fatalf("Get id=%d want %d", loaded.ID, aboutBlogID)
	}
	if loaded.Title != "about" {
		t.Fatalf("Get title=%q want about", loaded.Title)
	}
}

func TestBuildBlogAllowsEmptyCategory(t *testing.T) {
	_, code, fields := buildBlog("sample", "body", "", "private", "diary", "2026-04-19", true)
	if code != "" {
		t.Fatalf("buildBlog code=%q want empty", code)
	}
	if len(fields) > 0 {
		t.Fatalf("buildBlog fields=%v want empty", fields)
	}
}

func TestBuildBlogRejectsPublishedAtWithTime(t *testing.T) {
	_, code, fields := buildBlog("sample", "body", "", "private", "diary", "2026-04-19 00:00:00", true)
	if code != "INVALID_PUBLISHED_AT" {
		t.Fatalf("buildBlog code=%q want INVALID_PUBLISHED_AT", code)
	}
	if fields["published_at"] != "公開日の形式が不正です(yyyy-mm-dd)。" {
		t.Fatalf("published_at error=%q", fields["published_at"])
	}
}
