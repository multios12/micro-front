package store

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
)

func registerTestBlogSelection(t *testing.T, st *Store, parent, name string) {
	t.Helper()
	ctx := context.Background()
	var parentID *int64
	if parent != "" {
		var id int64
		err := st.DB.QueryRow(`SELECT id FROM categories WHERE parent_id IS NULL AND is_heading=1 AND name=?`, parent).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			item, err := st.CreateCategory(ctx, CategoryEntity{Name: parent, IsHeading: true})
			if err != nil {
				t.Fatal(err)
			}
			id = item.ID
		} else if err != nil {
			t.Fatal(err)
		}
		parentID = &id
	}
	if name != "" {
		var id int64
		err := st.DB.QueryRow(`SELECT id FROM categories WHERE parent_id IS ? AND is_heading=0 AND name=?`, parentID, name).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			addCategory(t, st, name, "test-"+parent+"-"+name, parentID, 10)
		} else if err != nil {
			t.Fatal(err)
		}
	}
}

func TestSaveBlogDoesNotRegisterCategories(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	blog := BlogEntitty{Title: "記事", ParentCategory: "親", Category: "子", Status: "private"}
	if _, err := st.CreateBlog(ctx, blog); !errors.Is(err, ErrBlogCategoryNotFound) {
		t.Fatalf("unregistered create: %v", err)
	}
	items, _ := st.ListCategories(ctx)
	if len(items) != 0 {
		t.Fatalf("categories created: %#v", items)
	}
	registerTestBlogSelection(t, st, "親", "子")
	saved, err := st.CreateBlog(ctx, blog)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := st.ListCategories(ctx)
	saved.ParentCategory = "未登録の親"
	if _, err := st.UpdateBlog(ctx, saved.ID, saved); !errors.Is(err, ErrBlogCategoryNotFound) {
		t.Fatalf("unregistered parent: %v", err)
	}
	saved.ParentCategory = "親"
	saved.Category = "未登録の子"
	if _, err := st.UpdateBlog(ctx, saved.ID, saved); !errors.Is(err, ErrBlogCategoryNotFound) {
		t.Fatalf("unregistered child: %v", err)
	}
	after, _ := st.ListCategories(ctx)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("saving article changed category list")
	}
	current, _ := st.GetBlog(ctx, saved.ID)
	if current.ParentCategory != "親" || current.Category != "子" {
		t.Fatalf("article changed: %#v", current)
	}
	current.ParentCategory = ""
	current.Category = ""
	cleared, err := st.UpdateBlog(ctx, current.ID, current)
	if err != nil || cleared.ParentCategory != "" || cleared.Category != "" {
		t.Fatalf("clear: %#v %v", cleared, err)
	}
	after, _ = st.ListCategories(ctx)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("clearing category changed category list")
	}
}

func TestCategoryRenameUpdatesOnlyLinkedBlogs(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	var articles []BlogEntitty
	for i, input := range []struct{ title, parent, status string }{{"公開", "親A", "public"}, {"非公開", "親A", "private"}, {"別親", "親B", "public"}, {"親なし", "", "private"}} {
		registerTestBlogSelection(t, st, input.parent, "同名")
		blog, err := st.CreateBlog(ctx, BlogEntitty{Title: input.title, ParentCategory: input.parent, Category: "同名", Status: input.status})
		if err != nil {
			t.Fatal(err)
		}
		articles = append(articles, blog)
		if _, err := st.DB.Exec(`UPDATE blogs SET updated_at='2000-01-01 00:00:00' WHERE id=?`, articles[i].ID); err != nil {
			t.Fatal(err)
		}
	}
	items, err := st.ListCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var root, child CategoryEntity
	for _, c := range items {
		if c.IsHeading && c.Name == "親A" {
			root = c
		}
	}
	for _, c := range items {
		if c.ParentID != nil && *c.ParentID == root.ID {
			child = c
		}
	}
	child.Name = "変更後 / カテゴリ"
	if _, err := st.UpdateCategory(ctx, child.ID, child); err != nil {
		t.Fatal(err)
	}
	root.Name = "変更後の親"
	if _, err := st.UpdateCategory(ctx, root.ID, root); err != nil {
		t.Fatal(err)
	}
	for i, article := range articles {
		got, err := st.GetBlog(ctx, article.ID)
		if err != nil {
			t.Fatal(err)
		}
		if i < 2 {
			if got.Category != child.Name || got.ParentCategory != root.Name || got.UpdatedAt == "2000-01-01 00:00:00" {
				t.Fatalf("linked=%#v", got)
			}
		} else if got.Category != article.Category || got.ParentCategory != article.ParentCategory || got.UpdatedAt != "2000-01-01 00:00:00" {
			t.Fatalf("unrelated=%#v", got)
		}
	}
	// 名前変更後の記事を保存しても、古いカテゴリや重複カテゴリを作成しない。
	got, _ := st.GetBlog(ctx, articles[0].ID)
	before, _ := st.ListCategories(ctx)
	if _, err := st.UpdateBlog(ctx, got.ID, got); err != nil {
		t.Fatal(err)
	}
	after, _ := st.ListCategories(ctx)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("saving renamed article changed categories")
	}
}

func TestBlogCategorySyncRollback(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	if _, err := st.CreateBlog(ctx, BlogEntitty{Title: "失敗", ParentCategory: "親", Category: "子", Status: "invalid"}); err == nil {
		t.Fatal("expected blog save failure")
	}
	items, err := st.ListCategories(ctx)
	if err != nil || len(items) != 0 {
		t.Fatalf("partial category registration: %#v %v", items, err)
	}
	registerTestBlogSelection(t, st, "親", "子")
	blog, err := st.CreateBlog(ctx, BlogEntitty{Title: "記事", ParentCategory: "親", Category: "子", Status: "public"})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := st.ListCategories(ctx)
	if _, err := st.DB.Exec(`CREATE TRIGGER fail_blog_rename BEFORE UPDATE OF category ON blogs BEGIN SELECT RAISE(ABORT,'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	child := before[1]
	child.Name = "変更後"
	if _, err := st.UpdateCategory(ctx, child.ID, child); err == nil {
		t.Fatal("expected rename failure")
	}
	after, _ := st.ListCategories(ctx)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("category rename was not rolled back")
	}
	got, _ := st.GetBlog(ctx, blog.ID)
	if got.Category != blog.Category {
		t.Fatal("blog rename was not rolled back")
	}
}

func removeBlogCategoryLinks(t *testing.T, st *Store) {
	t.Helper()
	restoreLegacyCategoryKeys(t, st, nil)
	_, err := st.DB.Exec(`DROP TRIGGER categories_sync_blog_names;
		DELETE FROM store_migrations WHERE version='005_blog_category_links';
		DROP INDEX idx_blogs_category_id; DROP INDEX idx_blogs_parent_category_id;
		ALTER TABLE blogs DROP COLUMN category_id; ALTER TABLE blogs DROP COLUMN parent_category_id;`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBlogCategoryLinksMigration(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	root, err := st.CreateCategory(ctx, CategoryEntity{Name: "親見出し", IsHeading: true})
	if err != nil {
		t.Fatal(err)
	}
	child := addCategory(t, st, "カテゴリ名", "legacy-child", &root.ID, 10)
	removeBlogCategoryLinks(t, st)
	if _, err := st.DB.Exec("UPDATE categories SET category_key=CASE id WHEN ? THEN 'legacy-parent' WHEN ? THEN 'legacy-child' END", root.ID, child.ID); err != nil {
		t.Fatal(err)
	}
	_, err = st.DB.Exec(`INSERT INTO blogs(title,content,summary,category,parent_category,status,published_at,updated_at)
		VALUES ('旧記事','本文','概要','legacy-child','legacy-parent','public','2020-01-01','2020-01-01 00:00:00'),
		('未登録の記事','','','未登録の子','未登録の親','private','',''),('未分類','','',NULL,'','private','','')`)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	st, err = New(st.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	blog, err := st.GetBlogByTitle(ctx, "旧記事")
	if err != nil || blog.Category != child.Name || blog.ParentCategory != root.Name || blog.UpdatedAt != "2020-01-01 00:00:00" || blog.Content != "本文" {
		t.Fatalf("migrated=%#v err=%v", blog, err)
	}
	var categoryID, parentID int64
	if err := st.DB.QueryRow(`SELECT category_id,parent_category_id FROM blogs WHERE id=?`, blog.ID).Scan(&categoryID, &parentID); err != nil || categoryID != child.ID || parentID != root.ID {
		t.Fatalf("links=%d,%d err=%v", categoryID, parentID, err)
	}
	items, err := st.ListCategories(ctx)
	if err != nil || len(items) != 4 {
		t.Fatalf("categories=%#v err=%v", items, err)
	}
	child.Name = "名前変更"
	if _, err := st.UpdateCategory(ctx, child.ID, child); err != nil {
		t.Fatal(err)
	}
	blog, _ = st.GetBlog(ctx, blog.ID)
	if blog.Category != child.Name {
		t.Fatalf("renamed migrated article=%#v", blog)
	}
	if _, err := st.DB.Exec(`DELETE FROM categories WHERE id=?`, child.ID); err == nil {
		t.Fatal("foreign key did not protect linked category")
	}
	before, _ := st.ListCategories(ctx)
	st.Close()
	st, err = New(st.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	after, _ := st.ListCategories(ctx)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("migration repeated on restart")
	}
}

func TestBlogCategoryLinksMigrationRollback(t *testing.T) {
	st := newCategoryStore(t)
	removeBlogCategoryLinks(t, st)
	_, err := st.DB.Exec(`INSERT INTO blogs(title,content,summary,category,parent_category,status,published_at,updated_at)
		VALUES ('旧記事','','','子','親','private','','');
		CREATE TRIGGER fail_migration BEFORE UPDATE ON blogs BEGIN SELECT RAISE(ABORT,'test failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.migrateBlogCategoryLinks(); err == nil {
		t.Fatal("expected migration error")
	}
	items, err := st.ListCategories(context.Background())
	if err != nil || len(items) != 0 {
		t.Fatalf("partial migration categories=%#v err=%v", items, err)
	}
	var columns, applied int
	if err := st.DB.QueryRow(`SELECT count(*) FROM pragma_table_info('blogs') WHERE name IN ('category_id','parent_category_id')`).Scan(&columns); err != nil || columns != 0 {
		t.Fatalf("partial migration columns=%d err=%v", columns, err)
	}
	if err := st.DB.QueryRow(`SELECT count(*) FROM store_migrations WHERE version='005_blog_category_links'`).Scan(&applied); err != nil || applied != 0 {
		t.Fatalf("partial migration version=%d err=%v", applied, err)
	}
	if _, err := st.DB.Exec(`DROP TRIGGER fail_migration`); err != nil {
		t.Fatal(err)
	}
	if err := st.migrateBlogCategoryLinks(); err != nil {
		t.Fatal(err)
	}
}
