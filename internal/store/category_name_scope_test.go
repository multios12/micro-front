package store

import (
	"context"
	"testing"
)

func TestCategoryNameScopeMigration(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	root, err := st.CreateCategory(ctx, CategoryEntity{Name: "同名", IsHeading: true})
	if err != nil {
		t.Fatal(err)
	}
	// 旧インデックスを持つ既存DBを再現する。
	_, err = st.DB.Exec(`DELETE FROM store_migrations WHERE version='004_category_name_scope';
		DROP INDEX idx_categories_root_name;
		CREATE UNIQUE INDEX idx_categories_root_name ON categories(name) WHERE parent_id IS NULL`)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	st, err = New(st.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	orphan, err := st.CreateCategory(ctx, CategoryEntity{Name: root.Name})
	if err != nil {
		t.Fatalf("heading and unparented name conflict: %v", err)
	}
	for _, heading := range []bool{true, false} {
		if _, err := st.CreateCategory(ctx, CategoryEntity{Name: root.Name, IsHeading: heading}); err == nil {
			t.Fatalf("duplicate accepted: is_heading=%v", heading)
		}
	}
	if _, err := st.CreateCategory(ctx, CategoryEntity{Name: root.Name, ParentID: &root.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateCategory(ctx, CategoryEntity{Name: root.Name, ParentID: &root.ID}); err == nil {
		t.Fatal("duplicate siblings accepted")
	}
	// 別名の親なしカテゴリを親と同名へ編集することも許可する。
	if err := st.DeleteCategory(ctx, orphan.ID); err != nil {
		t.Fatal(err)
	}
	orphan = addCategory(t, st, "別名", "orphan-again", nil, 0)
	orphan.Name = root.Name
	if _, err := st.UpdateCategory(ctx, orphan.ID, orphan); err != nil {
		t.Fatal(err)
	}
	st.Close()
	st, err = New(st.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	got, err := st.GetCategory(ctx, orphan.ID)
	if err != nil || got.Name != root.Name {
		t.Fatalf("reopen: %#v %v", got, err)
	}
}
