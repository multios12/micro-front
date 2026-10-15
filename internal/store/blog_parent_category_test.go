package store

import (
	"context"
	"errors"
	"testing"
)

func TestBlogParentCategoryPersistence(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	root, err := st.CreateCategory(ctx, CategoryEntity{Name: "親", IsHeading: true, SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	addCategory(t, st, "子", "child", &root.ID, 10)
	if _, err := st.CreateBlog(ctx, BlogEntitty{Title: "親省略", Category: "child", Status: "private"}); !errors.Is(err, ErrBlogCategoryNotFound) {
		t.Fatalf("unexpected parent inference: %v", err)
	}
	blog, err := st.CreateBlog(ctx, BlogEntitty{Title: "記事", ParentCategory: "親", Category: "子", Status: "public"})
	if err != nil || blog.ParentCategory != "親" {
		t.Fatalf("create: %#v %v", blog, err)
	}
	withID, err := st.CreateBlogWithID(ctx, BlogEntitty{Title: "ID指定", ParentCategory: "親", Category: "子", Status: "private"}, 100)
	if err != nil || withID.ParentCategory != "親" {
		t.Fatalf("create with ID: %#v %v", withID, err)
	}
	for _, list := range []func() ([]BlogEntitty, error){
		func() ([]BlogEntitty, error) {
			result, err := st.ListBlogs(ctx, BlogListFilter{})
			return result.Items, err
		},
		func() ([]BlogEntitty, error) { return st.ListBlogsAll(ctx) },
		func() ([]BlogEntitty, error) { return st.ListPublicBlogs(ctx) },
	} {
		items, err := list()
		if err != nil || len(items) == 0 {
			t.Fatalf("list: %#v %v", items, err)
		}
		for _, item := range items {
			if item.ParentCategory != "親" {
				t.Fatalf("parent: %#v", item)
			}
		}
	}
	byTitle, err := st.GetBlogByTitle(ctx, blog.Title)
	if err != nil || byTitle.ParentCategory != "親" {
		t.Fatalf("by title: %#v %v", byTitle, err)
	}
	registerTestBlogSelection(t, st, "", "親")
	blog.Category = "親"
	blog.ParentCategory = ""
	blog, err = st.UpdateBlog(ctx, blog.ID, blog)
	if err != nil || blog.ParentCategory != "" {
		t.Fatalf("update root: %#v %v", blog, err)
	}
	registerTestBlogSelection(t, st, "explicit-parent", "unregistered")
	blog.Category = "unregistered"
	blog.ParentCategory = "explicit-parent"
	blog, err = st.UpdateBlog(ctx, blog.ID, blog)
	if err != nil || blog.ParentCategory != "explicit-parent" {
		t.Fatalf("explicit: %#v %v", blog, err)
	}
	var parent CategoryEntity
	items, err := st.ListCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.IsHeading && item.Name == "explicit-parent" {
			parent = item
		}
	}
	if parent.ID == 0 {
		t.Fatal("parent not registered")
	}
	if err := st.DeleteCategory(ctx, parent.ID); err != ErrCategoryInUse {
		t.Fatalf("delete referenced parent: %v", err)
	}
}

func TestBlogParentCategoryMigration(t *testing.T) {
	dir := t.TempDir()
	st, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	root := addCategory(t, st, "親", "root", nil, 10)
	child := addCategory(t, st, "子", "child", &root.ID, 10)
	restoreLegacyCategoryKeys(t, st, map[int64]string{root.ID: "root", child.ID: "child"})
	// 002適用前のDB状態を再現する。
	_, err = st.DB.Exec(`DELETE FROM store_migrations WHERE version IN ('002_blog_parent_category','005_blog_category_links');
		DROP TRIGGER categories_sync_blog_names;
		DROP INDEX idx_blogs_parent_category;
		ALTER TABLE blogs DROP COLUMN parent_category;
		INSERT INTO blogs(title,content,summary,category,status,published_at,updated_at)
		VALUES ('子の記事','','','child','public','',''),('親の記事','','','root','private','',''),('未登録','','','unknown','private','','');`)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	st, err = New(dir)
	if err != nil {
		t.Fatal(err)
	}
	for title, want := range map[string]string{"子の記事": "親", "親の記事": "", "未登録": ""} {
		blog, err := st.GetBlogByTitle(context.Background(), title)
		if err != nil || blog.ParentCategory != want {
			t.Fatalf("migration: %#v want=%q err=%v", blog, want, err)
		}
	}
	// 再起動で親見出し値を再補完しない。
	if _, err := st.DB.Exec(`UPDATE blogs SET parent_category='manual' WHERE title='子の記事'`); err != nil {
		t.Fatal(err)
	}
	st.Close()
	st, err = New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	blog, err := st.GetBlogByTitle(context.Background(), "子の記事")
	if err != nil || blog.ParentCategory != "manual" {
		t.Fatalf("reopen: %#v %v", blog, err)
	}
}
