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

func TestBlogParentCategoryAPI(t *testing.T) {
	ctx := context.Background()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	uc := Usecase{Store: st, DataDir: st.DataDir}
	parent, err := st.CreateCategory(ctx, store.CategoryEntity{Name: "parent", IsHeading: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []store.CategoryEntity{
		{Name: "child", ParentID: &parent.ID},
		{Name: "child"},
	} {
		if _, err := st.CreateCategory(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	parentName := "parent"
	created, code, fields, err := uc.Create(ctx, BlogsCreateRequest{
		Title: "親見出し確認", Content: "本文", Category: "child", ParentCategory: &parentName,
		Status: "public", PublishedAt: "2026-10-15",
	})
	if err != nil || code != "" || len(fields) > 0 || created.ParentCategory != "parent" {
		t.Fatalf("create: %#v code=%q fields=%v err=%v", created, code, fields, err)
	}
	detail, err := uc.Get(ctx, created.ID)
	if err != nil || detail.ParentCategory != "parent" {
		t.Fatalf("get: %#v %v", detail, err)
	}
	list, _, _, err := uc.List(ctx, 1, 20, "")
	if err != nil || len(list.Items) != 1 || list.Items[0].ParentCategory != "parent" {
		t.Fatalf("list: %#v %v", list, err)
	}
	// 旧クライアントが親見出しを送らなくても、同じカテゴリなら保存済み値を維持する。
	updated, _, fields, err := uc.Update(ctx, created.ID, BlogsUpdateRequest{
		Title: created.Title, Content: "更新後", Category: "child", Status: "public", PublishedAt: "2026-10-15",
	})
	if err != nil || len(fields) > 0 || updated.ParentCategory != "parent" {
		t.Fatalf("update: %#v fields=%v err=%v", updated, fields, err)
	}
	emptyParent := ""
	detached, _, fields, err := uc.Update(ctx, created.ID, BlogsUpdateRequest{
		Title: created.Title, Content: "親を解除", Category: "child", ParentCategory: &emptyParent,
		Status: "public", PublishedAt: "2026-10-15",
	})
	if err != nil || len(fields) > 0 || detached.ParentCategory != "" {
		t.Fatalf("explicit parent removal: %#v fields=%v err=%v", detached, fields, err)
	}
	parentName = "入力 / 親見出し"
	symbolParent, err := st.CreateCategory(ctx, store.CategoryEntity{Name: parentName, IsHeading: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateCategory(ctx, store.CategoryEntity{Name: "child", ParentID: &symbolParent.ID}); err != nil {
		t.Fatal(err)
	}
	_, _, fields, err = uc.Create(ctx, BlogsCreateRequest{
		Title: "記号を含む親", Content: "本文", Category: "child", ParentCategory: &parentName, Status: "private", PublishedAt: "2026-10-15",
	})
	if err != nil || len(fields) > 0 {
		t.Fatalf("validation: fields=%v err=%v", fields, err)
	}
}
