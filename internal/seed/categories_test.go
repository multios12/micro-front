package seed

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"micro-front/internal/store"
)

func newSeedStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestDefaultSeedIncludesChildCategories(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	ctx := context.Background()
	st := newSeedStore(t)
	uc := Usecase{Store: st, DataDir: st.DataDir}
	// 再投入でも外部キーに従って小見出しを先に消せることを確認する。
	for range 2 {
		if err := uc.Run(ctx, Options{SeedDir: "seeds/default", Reset: true}); err != nil {
			t.Fatal(err)
		}
		items, err := st.ListCategories(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 12 {
			t.Fatalf("categories=%#v", items)
		}
		byKey := make(map[string]store.CategoryEntity)
		var childCount int
		byID := make(map[int64]store.CategoryEntity)
		for _, c := range items {
			byID[c.ID] = c
		}
		for _, c := range items {
			if c.IsHeading {
				byKey[c.Name] = c
			}
			if c.ParentID != nil {
				byKey[byID[*c.ParentID].Name+"-"+c.Name] = c
			}
			if c.ParentID != nil {
				childCount++
			}
		}
		if childCount != 4 {
			t.Fatalf("children=%d", childCount)
		}
		if !byKey["お知らせ"].IsHeading {
			t.Fatal("seed root without children must remain a heading")
		}
		for id, key := range map[int64]string{2: "開発メモ-画像", 3: "設計-導線", 4: "運用-公開作業", 13: "開発メモ-Markdown"} {
			blog, err := st.GetBlog(ctx, id)
			if err != nil || blog.Category != byKey[key].Name || byKey[key].ParentID == nil {
				t.Fatalf("blog=%#v category=%#v err=%v", blog, byKey[key], err)
			}
		}
	}
}

func TestSeedCategoriesPreservesIDsAndResolvesParents(t *testing.T) {
	ctx := context.Background()
	st := newSeedStore(t)
	dir := t.TempDir()
	body := `[
		{"name":"小見出し","category_key":"child","parent_key":"root","sort_order":10},
		{"name":"親見出し","category_key":"root","sort_order":20}
	]`
	if err := os.WriteFile(filepath.Join(dir, "categories.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	uc := Usecase{Store: st, DataDir: st.DataDir}
	var firstIDs []int64
	for i := range 2 {
		if err := uc.seedCategories(ctx, dir); err != nil {
			t.Fatal(err)
		}
		items, err := st.ListCategories(ctx)
		if err != nil || len(items) != 2 {
			t.Fatalf("items=%#v err=%v", items, err)
		}
		if items[1].ParentID == nil || *items[1].ParentID != items[0].ID {
			t.Fatalf("items=%#v", items)
		}
		ids := []int64{items[0].ID, items[1].ID}
		if i == 0 {
			firstIDs = ids
		} else if !reflect.DeepEqual(ids, firstIDs) {
			t.Fatalf("IDs changed: %v -> %v", firstIDs, ids)
		}
	}
}

func TestSeedCategoriesValidationAndRollback(t *testing.T) {
	cases := []string{
		`[{"name":"子","category_key":"child","parent_key":"missing"}]`,
		`[{"name":"親","category_key":"root"},{"name":"子","category_key":"child","parent_key":"root"},{"name":"孫","category_key":"grandchild","parent_key":"child"}]`,
		`[{"name":"親","category_key":"root"},{"name":"重複キー","category_key":"root"}]`,
		`[{"name":"重複名","category_key":"one"},{"name":"重複名","category_key":"two"}]`,
	}
	for _, body := range cases {
		t.Run(body, func(t *testing.T) {
			st := newSeedStore(t)
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "categories.json"), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			uc := Usecase{Store: st, DataDir: st.DataDir}
			if err := uc.seedCategories(context.Background(), dir); err == nil {
				t.Fatal("invalid categories accepted")
			}
			items, err := st.ListCategories(context.Background())
			if err != nil || len(items) != 0 {
				t.Fatalf("partial import: %#v err=%v", items, err)
			}
		})
	}
}

func TestSeedCategoriesOptional(t *testing.T) {
	st := newSeedStore(t)
	uc := Usecase{Store: st, DataDir: st.DataDir}
	if err := uc.seedCategories(context.Background(), t.TempDir()); err != nil {
		t.Fatal(err)
	}
}
