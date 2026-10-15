package store

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
)

func TestMoveCategoryGroups(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	a := addCategory(t, st, "親A", "a", nil, 10)
	a1 := addCategory(t, st, "子A1", "a1", &a.ID, 0)
	a2 := addCategory(t, st, "子A2", "a2", &a.ID, 0)
	u1 := addCategory(t, st, "親なし1", "u1", nil, 0)
	b := addCategory(t, st, "親B", "b", nil, 10)
	b1 := addCategory(t, st, "子B", "b1", &b.ID, 0)
	u2 := addCategory(t, st, "親なし2", "u2", nil, 0)
	list := func() []CategoryEntity {
		t.Helper()
		items, err := st.ListCategories(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return items
	}
	before := list()
	if err := st.MoveCategory(ctx, a1.ID, "down"); err != nil {
		t.Fatal(err)
	}
	items := list()
	positions := map[int64]int{}
	for i, c := range items {
		positions[c.ID] = i
	}
	if positions[a2.ID] >= positions[a1.ID] {
		t.Fatal("tied children did not swap")
	}
	// 子の移動では別の親や最上位の並び順は変更しない。
	for _, c := range before {
		if c.ID != a1.ID && c.ID != a2.ID {
			got, err := st.GetCategory(ctx, c.ID)
			if err != nil || !reflect.DeepEqual(got, c) {
				t.Fatalf("unrelated row changed: %#v -> %#v err=%v", c, got, err)
			}
		}
	}
	if err := st.MoveCategory(ctx, b.ID, "up"); err != nil {
		t.Fatal(err)
	}
	positions = map[int64]int{}
	for i, c := range list() {
		positions[c.ID] = i
	}
	if positions[b.ID] >= positions[a.ID] || positions[b1.ID] != positions[b.ID]+1 {
		t.Fatal("heading did not move with children")
	}
	if err := st.MoveCategory(ctx, u2.ID, "up"); err != nil {
		t.Fatal(err)
	}
	positions = map[int64]int{}
	for i, c := range list() {
		positions[c.ID] = i
	}
	if positions[u2.ID] >= positions[u1.ID] {
		t.Fatal("unparented categories did not swap")
	}
	// 各グループの先頭・末尾は変更なし。
	for _, move := range []struct {
		id        int64
		direction string
	}{{a.ID, "down"}, {a2.ID, "up"}, {a1.ID, "down"}, {u2.ID, "up"}, {b1.ID, "up"}} {
		before := list()
		if err := st.MoveCategory(ctx, move.id, move.direction); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(list(), before) {
			t.Fatalf("boundary move modified data: %#v", move)
		}
	}
	if err := st.MoveCategory(ctx, 99999, "up"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing: %v", err)
	}
	if err := st.MoveCategory(ctx, a.ID, "invalid"); err == nil {
		t.Fatal("invalid direction accepted")
	}
	st.Close()
	reopened, err := New(st.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.GetCategory(ctx, u2.ID)
	if err != nil || got.SortOrder != 0 {
		t.Fatalf("order not persisted: %#v %v", got, err)
	}
}

func TestMoveCategoryRollback(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	a := addCategory(t, st, "A", "a", nil, 30)
	addCategory(t, st, "B", "b", nil, 40)
	before, err := st.ListCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 2行目の更新を失敗させ、途中までの並べ替えも戻ることを確認する。
	if _, err := st.DB.Exec(`CREATE TRIGGER fail_order BEFORE UPDATE OF sort_order ON categories WHEN NEW.name='A' BEGIN SELECT RAISE(ABORT,'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := st.MoveCategory(ctx, a.ID, "down"); err == nil {
		t.Fatal("expected update error")
	}
	after, err := st.ListCategories(ctx)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("partial move: %#v %v", after, err)
	}
}

func TestMoveCategoryMixesHeadingsAndUnparented(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	heading, err := st.CreateCategory(ctx, CategoryEntity{Name: "見出し", IsHeading: true, SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	child := addCategory(t, st, "子", "", &heading.ID, 10)
	normal := addCategory(t, st, "通常", "", nil, 20)
	if err := st.MoveCategory(ctx, normal.ID, "up"); err != nil {
		t.Fatal(err)
	}
	items, err := st.ListCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ids := []int64{}
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	if !reflect.DeepEqual(ids, []int64{normal.ID, heading.ID, child.ID}) {
		t.Fatalf("mixed order: %v", ids)
	}
	if err := st.MoveCategory(ctx, heading.ID, "up"); err != nil {
		t.Fatal(err)
	}
	items, err = st.ListCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ids = nil
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	if !reflect.DeepEqual(ids, []int64{heading.ID, child.ID, normal.ID}) {
		t.Fatalf("heading with children: %v", ids)
	}
}
