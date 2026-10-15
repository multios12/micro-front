package store

import (
	"context"
	"reflect"
	"testing"
)

func TestRemoveCategoryKeyPreservesRelations(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	parent, err := st.CreateCategory(ctx, CategoryEntity{Name: "親", IsHeading: true})
	if err != nil {
		t.Fatal(err)
	}
	child := addCategory(t, st, "子", "unused", &parent.ID, 20)
	blog, err := st.CreateBlog(ctx, BlogEntitty{Title: "記事", ParentCategory: parent.Name, Category: child.Name, Status: "private"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := st.ListCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	restoreLegacyCategoryKeys(t, st, map[int64]string{parent.ID: "old-parent", child.ID: "old-child"})
	if _, err := st.DB.Exec(`CREATE TRIGGER categories_keep_key BEFORE UPDATE OF category_key ON categories
 WHEN NEW.category_key IS NOT OLD.category_key BEGIN SELECT RAISE(ABORT,'immutable'); END`); err != nil {
		t.Fatal(err)
	}
	dir := st.DataDir
	st.Close()
	reopened, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var columns int
	if err := reopened.DB.QueryRow(`SELECT count(*) FROM pragma_table_info('categories') WHERE name='category_key'`).Scan(&columns); err != nil || columns != 0 {
		t.Fatalf("key still exists: %d %v", columns, err)
	}
	after, err := reopened.ListCategories(ctx)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("categories changed: %#v %v", after, err)
	}
	var categoryID, parentID int64
	if err := reopened.DB.QueryRow(`SELECT category_id,parent_category_id FROM blogs WHERE id=?`, blog.ID).Scan(&categoryID, &parentID); err != nil || categoryID != child.ID || parentID != parent.ID {
		t.Fatalf("links: %d %d %v", categoryID, parentID, err)
	}
	child.Name = "変更後"
	if _, err := reopened.UpdateCategory(ctx, child.ID, child); err != nil {
		t.Fatal(err)
	}
	got, err := reopened.GetBlog(ctx, blog.ID)
	if err != nil || got.Category != child.Name {
		t.Fatalf("rename sync: %#v %v", got, err)
	}
	if _, err := reopened.DB.Exec(`DELETE FROM categories WHERE id=?`, child.ID); err == nil {
		t.Fatal("foreign key protection lost")
	}
	if err := reopened.migrateRemoveCategoryKey(); err != nil {
		t.Fatal(err)
	}
}
