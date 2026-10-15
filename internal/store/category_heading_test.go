package store

import (
	"context"
	"testing"
)

func TestEmptyHeadingPersistsAndMoves(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	first, err := st.CreateCategory(ctx, CategoryEntity{Name: "空の親A", IsHeading: true, SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	second, err := st.CreateCategory(ctx, CategoryEntity{Name: "空の親B", IsHeading: true, SortOrder: 20})
	if err != nil {
		t.Fatal(err)
	}
	orphan := addCategory(t, st, "親なし", "orphan", nil, 5)
	if err := st.MoveCategory(ctx, second.ID, "up"); err != nil {
		t.Fatal(err)
	}
	a, _ := st.GetCategory(ctx, first.ID)
	b, _ := st.GetCategory(ctx, second.ID)
	if !a.IsHeading || !b.IsHeading || b.SortOrder >= a.SortOrder {
		t.Fatalf("headings: %#v %#v", a, b)
	}
	got, _ := st.GetCategory(ctx, orphan.ID)
	if got.SortOrder >= b.SortOrder || got.IsHeading {
		t.Fatalf("orphan order/type: %#v", got)
	}
	st.Close()
	st, err = New(st.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	got, err = st.GetCategory(ctx, first.ID)
	if err != nil || !got.IsHeading {
		t.Fatalf("reopen: %#v %v", got, err)
	}
}

func TestCategoryHeadingMigration(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	parent := addCategory(t, st, "親", "root", nil, 10)
	child := addCategory(t, st, "子", "child", &parent.ID, 10)
	orphan := addCategory(t, st, "単独", "orphan", nil, 20)
	_, err := st.DB.Exec(`DELETE FROM store_migrations WHERE version IN ('003_category_heading','004_category_name_scope');
		DROP INDEX idx_categories_root_name;
		ALTER TABLE categories DROP COLUMN is_heading;
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
	for id, want := range map[int64]bool{parent.ID: true, child.ID: false, orphan.ID: false} {
		got, err := st.GetCategory(ctx, id)
		if err != nil || got.IsHeading != want {
			t.Fatalf("category=%#v want heading=%v err=%v", got, want, err)
		}
	}
}
