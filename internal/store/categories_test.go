package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newCategoryStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func addCategory(t *testing.T, st *Store, name, key string, parent *int64, order int) CategoryEntity {
	t.Helper()
	c, err := st.CreateCategory(context.Background(), CategoryEntity{
		Name: name, ParentID: parent, SortOrder: order,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCategoriesCRUDAndOrder(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	items, err := st.ListCategories(ctx)
	if err != nil || items == nil || len(items) != 0 {
		t.Fatalf("empty list: %#v, %v", items, err)
	}
	later := addCategory(t, st, "日記", "diary", nil, 20)
	root := addCategory(t, st, "技術", "tech", nil, 10)
	goCategory := addCategory(t, st, " Go　", "tech-Go", &root.ID, 20)
	sqlite := addCategory(t, st, "SQLite", "tech-SQLite", &root.ID, 0)
	tied := addCategory(t, st, "同順", "tie", nil, 10)
	items, err = st.ListCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, c := range items {
		ids = append(ids, c.ID)
	}
	if !reflect.DeepEqual(ids, []int64{root.ID, sqlite.ID, goCategory.ID, tied.ID, later.ID}) {
		t.Fatalf("order: %v", ids)
	}
	if goCategory.Name != "Go" || goCategory.ParentID == nil || *goCategory.ParentID != root.ID {
		t.Fatalf("child: %#v", goCategory)
	}
	if _, err := time.Parse(time.RFC3339Nano, goCategory.CreatedAt); err != nil {
		t.Fatal(err)
	}
	// 子を別の親見出しに移し、名前・表示順のみを変更する。
	updated, err := st.UpdateCategory(ctx, goCategory.ID, CategoryEntity{ParentID: &later.ID, Name: "Golang", SortOrder: 5})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CreatedAt != goCategory.CreatedAt || updated.UpdatedAt == goCategory.UpdatedAt || updated.Name != "Golang" {
		t.Fatalf("update: %#v", updated)
	}
	if err := st.DeleteCategory(ctx, goCategory.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetCategory(ctx, goCategory.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted get: %v", err)
	}
	if _, err := st.UpdateCategory(ctx, goCategory.ID, CategoryEntity{Name: "missing"}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing update: %v", err)
	}
	if err := st.DeleteCategory(ctx, goCategory.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing delete: %v", err)
	}
}

func TestCategoryDatabaseConstraints(t *testing.T) {
	st := newCategoryStore(t)
	root := addCategory(t, st, "技術", "tech", nil, 0)
	child := addCategory(t, st, "Go", "go", &root.ID, 0)
	other := addCategory(t, st, "日記", "diary", nil, 0)
	// 別の親の配下なら同名が使える。
	addCategory(t, st, "Go", "diary-Go", &other.ID, 0)
	cases := []struct {
		name string
		sql  string
		args []any
	}{
		{"third level", `INSERT INTO categories(parent_id,name,created_at,updated_at) VALUES (?, '孫','','')`, []any{child.ID}},
		{"missing parent", `UPDATE categories SET parent_id=9999 WHERE id=?`, []any{child.ID}},
		{"self parent", `UPDATE categories SET parent_id=id WHERE id=?`, []any{child.ID}},
		{"cycle", `UPDATE categories SET parent_id=? WHERE id=?`, []any{child.ID, root.ID}},
		{"move parent with children", `UPDATE categories SET parent_id=? WHERE id=?`, []any{other.ID, root.ID}},
		{"delete parent", `DELETE FROM categories WHERE id=?`, []any{root.ID}},
		{"duplicate root", `UPDATE categories SET name='技術' WHERE id=?`, []any{other.ID}},
		{"duplicate child", `INSERT INTO categories(parent_id,name,created_at,updated_at) VALUES (?, 'Go','','')`, []any{root.ID}},
		{"negative sort", `UPDATE categories SET sort_order=-1 WHERE id=?`, []any{child.ID}},
		{"fractional sort", `UPDATE categories SET sort_order=1.5 WHERE id=?`, []any{child.ID}},
		{"empty name", `UPDATE categories SET name='' WHERE id=?`, []any{child.ID}},
		{"untrimmed name", `UPDATE categories SET name=' Go ' WHERE id=?`, []any{child.ID}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := st.DB.Exec(tc.sql, tc.args...); err == nil {
				t.Fatal("constraint did not reject invalid data")
			}
		})
	}
	if err := st.DeleteCategory(context.Background(), root.ID); err == nil {
		t.Fatal("store deleted parent with children")
	}

	for _, c := range []CategoryEntity{
		{Name: "　\t"},
		{Name: "negative", SortOrder: -1},
	} {
		if _, err := st.CreateCategory(context.Background(), c); err == nil {
			t.Fatalf("accepted %#v", c)
		}
	}
}

func TestDeleteCategoryUsedByBlogs(t *testing.T) {
	st := newCategoryStore(t)
	ctx := context.Background()
	for _, status := range []string{"public", "private"} {
		c := addCategory(t, st, status, status, nil, 0)
		blog, err := st.CreateBlog(ctx, BlogEntitty{Title: status, Category: c.Name, Status: status})
		if err != nil {
			t.Fatal(err)
		}
		if err := st.DeleteCategory(ctx, c.ID); !errors.Is(err, ErrCategoryInUse) {
			t.Fatalf("delete %s category: %v", status, err)
		}
		if err := st.DeleteBlog(ctx, blog.ID); err != nil {
			t.Fatal(err)
		}
		if err := st.DeleteCategory(ctx, c.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDeleteCategoryUsedByName(t *testing.T) {
	for _, status := range []string{"public", "private"} {
		for _, kind := range []string{"parent", "child", "orphan"} {
			t.Run(status+"/"+kind, func(t *testing.T) {
				st := newCategoryStore(t)
				ctx := context.Background()
				parent, err := st.CreateCategory(ctx, CategoryEntity{Name: "親", IsHeading: true})
				if err != nil {
					t.Fatal(err)
				}
				target := parent
				parentName, categoryName := parent.Name, ""
				if kind == "child" {
					target = addCategory(t, st, "カテゴリ", "child", &parent.ID, 0)
					categoryName = target.Name
				} else if kind == "orphan" {
					target = addCategory(t, st, "カテゴリ", "orphan", nil, 0)
					parentName, categoryName = "", target.Name
				}
				result, err := st.DB.Exec(`INSERT INTO blogs(title,content,summary,category,parent_category,status,published_at,updated_at)
					VALUES ('記事','','',?,?,?,'','')`, categoryName, parentName, status)
				if err != nil {
					t.Fatal(err)
				}
				if err := st.DeleteCategory(ctx, target.ID); !errors.Is(err, ErrCategoryInUse) {
					t.Fatalf("delete category used by name: %v", err)
				}
				otherParent, err := st.CreateCategory(ctx, CategoryEntity{Name: "別の親", IsHeading: true})
				if err != nil {
					t.Fatal(err)
				}
				other := addCategory(t, st, target.Name, "other-child", &otherParent.ID, 0)
				if err := st.DeleteCategory(ctx, other.ID); err != nil {
					t.Fatalf("delete unrelated category: %v", err)
				}
				blogID, err := result.LastInsertId()
				if err != nil {
					t.Fatal(err)
				}
				if err := st.DeleteBlog(ctx, blogID); err != nil {
					t.Fatal(err)
				}
				if err := st.DeleteCategory(ctx, target.ID); err != nil {
					t.Fatalf("delete unused category: %v", err)
				}
			})
		}
	}
}

func TestForeignKeysOnEveryConnection(t *testing.T) {
	// URIで特別な意味を持つ文字を含む保存先も扱える。
	st, err := New(filepath.Join(t.TempDir(), "data #? 日本語"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	root := addCategory(t, st, "技術", "tech", nil, 0)
	addCategory(t, st, "Go", "go", &root.ID, 0)
	st.DB.SetMaxOpenConns(3)
	var conns []*sql.Conn
	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()
	for range 3 {
		conn, err := st.DB.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, conn)
		var enabled int
		if err := conn.QueryRowContext(context.Background(), `PRAGMA foreign_keys`).Scan(&enabled); err != nil || enabled != 1 {
			t.Fatalf("foreign_keys=%d: %v", enabled, err)
		}
		if _, err := conn.ExecContext(context.Background(), `DELETE FROM categories WHERE id=?`, root.ID); err == nil {
			t.Fatal("foreign key not enforced")
		}
	}
}

// legacyCategoryDB はカテゴリ機能導入前のスキーマと記事を用意します。
func legacyCategoryDB(t *testing.T, keys ...string) string {
	t.Helper()
	dir := t.TempDir()
	st, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := st.DB.Exec(`DROP TRIGGER categories_sync_blog_names;
		DROP INDEX idx_blogs_category_id; DROP INDEX idx_blogs_parent_category_id;
 ALTER TABLE blogs DROP COLUMN category_id; ALTER TABLE blogs DROP COLUMN parent_category_id;
 DROP TABLE categories; DROP TABLE store_migrations;`); err != nil {
		t.Fatal(err)
	}
	for i, key := range keys {
		// 導入前DBを再現するため、新ストアのカテゴリ補完処理を通さず投入する。
		_, err := st.DB.Exec(`INSERT INTO blogs
			(title,content,summary,category,status,published_at,updated_at)
			VALUES (?, '', '', ?, 'private', '', '')`, time.Unix(int64(i), 0).String(), key)
		if err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestCategoryMigrationOnce(t *testing.T) {
	dir := legacyCategoryDB(t, "技術-Go", "日記", "技術-Go", "", " 過去/カテゴリ ")
	st, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	items, err := st.ListCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("categories: %#v", items)
	}
	wantKeys := []string{" 過去/カテゴリ ", "技術-Go", "日記"}
	for i, c := range items {
		if c.Name != strings.TrimSpace(wantKeys[i]) || c.ParentID != nil || c.SortOrder != (i+1)*10 {
			t.Fatalf("migration row: %#v", c)
		}
	}
	if items[0].Name != "過去/カテゴリ" {
		t.Fatalf("name: %q", items[0].Name)
	}
	// 記事の値とカテゴリURLに使う値は元のまま残る。
	blogs, err := st.ListBlogsAll(ctx)
	if err != nil || len(blogs) != 5 {
		t.Fatalf("blogs: %#v %v", blogs, err)
	}
	// 未使用にして削除し、再起動後に復元されないことを確認する。
	if _, err := st.DB.Exec(`DELETE FROM blogs WHERE category_id=?`, items[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteCategory(ctx, items[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateBlog(ctx, BlogEntitty{Title: "after migration", Category: items[0].Name, Status: "private"}); !errors.Is(err, ErrBlogCategoryNotFound) {
		t.Fatal(err)
	}
	st.Close()
	st, err = New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	items, err = st.ListCategories(ctx)
	if err != nil || len(items) != 2 {
		t.Fatalf("reopened: %#v %v", items, err)
	}
}

func TestCategoryMigrationRollsBackInvalidLegacyNames(t *testing.T) {
	for _, keys := range [][]string{{"valid", "　\t"}, {"duplicate", " duplicate "}} {
		dir := legacyCategoryDB(t, keys...)
		if st, err := New(dir); err == nil {
			st.Close()
			t.Fatal("invalid legacy data accepted")
		}
		db, err := sql.Open("sqlite", filepath.Join(dir, "app.db"))
		if err != nil {
			t.Fatal(err)
		}
		var count int
		if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name IN ('categories','store_migrations')`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("migration not rolled back: count=%d err=%v", count, err)
		}
		// 原因を解消すれば同じDBで移行をやり直せる。
		if _, err := db.Exec(`DELETE FROM blogs`); err != nil {
			t.Fatal(err)
		}
		db.Close()
		st, err := New(dir)
		if err != nil {
			t.Fatal(err)
		}
		st.Close()
	}
}

// restoreLegacyCategoryKeys は006適用前のDBを再現します。
func restoreLegacyCategoryKeys(t *testing.T, st *Store, keys map[int64]string) {
	t.Helper()
	if _, err := st.DB.Exec(`ALTER TABLE categories ADD COLUMN category_key TEXT NOT NULL DEFAULT '';
 UPDATE categories SET category_key='legacy-'||id;
 DELETE FROM store_migrations WHERE version='006_remove_category_key';`); err != nil {
		t.Fatal(err)
	}
	for id, key := range keys {
		if _, err := st.DB.Exec(`UPDATE categories SET category_key=? WHERE id=?`, key, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.DB.Exec(`CREATE UNIQUE INDEX legacy_category_keys ON categories(category_key)`); err != nil {
		t.Fatal(err)
	}
}
