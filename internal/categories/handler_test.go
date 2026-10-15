package categories

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"micro-front/internal/apiutil"
	"micro-front/internal/store"
)

func TestListCategories(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	h := Handler{Store: st}
	get := func() *httptest.ResponseRecorder {
		t.Helper()
		rr := httptest.NewRecorder()
		h.handlerList(rr, httptest.NewRequest(http.MethodGet, "/admin/api/categories", nil))
		return rr
	}
	decode := func(rr *httptest.ResponseRecorder) ListResponse {
		t.Helper()
		if rr.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Fatalf("content type=%q", got)
		}
		var response ListResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		return response
	}
	// 未登録はnullでなく空配列を返す。
	response := decode(get())
	if response.Items == nil || len(response.Items) != 0 {
		t.Fatalf("empty response=%#v", response)
	}
	add := func(name, key string, parent *int64, order int) store.CategoryEntity {
		t.Helper()
		c, err := st.CreateCategory(context.Background(), store.CategoryEntity{
			Name: name, ParentID: parent, SortOrder: order,
		})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	diary := add("日記", "diary", nil, 20)
	tech := add("技術", "tech", nil, 10)
	goCategory := add("Go", "tech-Go", &tech.ID, 30)
	sqlite := add("SQLite", "tech-SQLite", &tech.ID, 0)
	// 記事がないカテゴリも含め、DBで定義した階層・表示順のまま返す。
	response = decode(get())
	want := []store.CategoryEntity{tech, sqlite, goCategory, diary}
	if !reflect.DeepEqual(response.Items, want) {
		t.Fatalf("items=%#v want=%#v", response.Items, want)
	}
	// DBエラーを空一覧として扱わない。
	st.Close()
	rr := get()
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("closed DB: status=%d body=%s", rr.Code, rr.Body.String())
	}
	var apiError apiutil.ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &apiError); err != nil || apiError.Code != "INTERNAL_SERVER_ERROR" {
		t.Fatalf("error response=%#v err=%v", apiError, err)
	}
}

func TestCategoryCreateAndDelete(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	h := Handler{Store: st}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/api/categories", h.handlerCreate)
	mux.HandleFunc("DELETE /admin/api/categories/{category_id}", h.handlerDelete)
	request := func(method, path, body string, wantStatus int) *httptest.ResponseRecorder {
		t.Helper()
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(method, path, strings.NewReader(body)))
		if rr.Code != wantStatus {
			t.Fatalf("%s %s: status=%d want=%d body=%s", method, path, rr.Code, wantStatus, rr.Body.String())
		}
		return rr
	}
	create := func(body string) store.CategoryEntity {
		t.Helper()
		rr := request("POST", "/admin/api/categories", body, http.StatusCreated)
		var item store.CategoryEntity
		if err := json.Unmarshal(rr.Body.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		return item
	}
	root := create(`{"name":" 親見出し ","sort_order":10}`)
	if root.ParentID != nil || root.Name != "親見出し" {
		t.Fatalf("root=%#v", root)
	}
	child := create(fmt.Sprintf(`{"name":"小見出し","parent_id":%d,"sort_order":20}`, root.ID))
	if child.ParentID == nil || *child.ParentID != root.ID {
		t.Fatalf("child=%#v", child)
	}
	request("POST", "/admin/api/categories", `{"name":"親見出し"}`, http.StatusConflict)
	request("POST", "/admin/api/categories", fmt.Sprintf(`{"name":"小見出し","parent_id":%d}`, root.ID), http.StatusConflict)
	request("POST", "/admin/api/categories", `{"name":"　"}`, http.StatusBadRequest)
	request("POST", "/admin/api/categories", `{"name":"不正","sort_order":-1}`, http.StatusBadRequest)
	request("POST", "/admin/api/categories", `{"name":"不正","sort_order":0.5}`, http.StatusBadRequest)
	request("POST", "/admin/api/categories", `{"name":"不正","parent_id":9999}`, http.StatusBadRequest)
	request("POST", "/admin/api/categories", fmt.Sprintf(`{"name":"孫","parent_id":%d}`, child.ID), http.StatusBadRequest)
	request("POST", "/admin/api/categories", `{`, http.StatusBadRequest)
	deletePath := func(id int64) string { return fmt.Sprintf("/admin/api/categories/%d", id) }
	rr := request("DELETE", deletePath(root.ID), "", http.StatusConflict)
	if !strings.Contains(rr.Body.String(), "CATEGORY_HAS_CHILDREN") {
		t.Fatal(rr.Body.String())
	}
	blog, err := st.CreateBlog(context.Background(), store.BlogEntitty{Title: "利用中", ParentCategory: root.Name, Category: child.Name, Status: "private"})
	if err != nil {
		t.Fatal(err)
	}
	rr = request("DELETE", deletePath(child.ID), "", http.StatusConflict)
	if !strings.Contains(rr.Body.String(), "CATEGORY_IN_USE") {
		t.Fatal(rr.Body.String())
	}
	if err := st.DeleteBlog(context.Background(), blog.ID); err != nil {
		t.Fatal(err)
	}
	request("DELETE", deletePath(child.ID), "", http.StatusOK)
	request("DELETE", deletePath(root.ID), "", http.StatusOK)
	request("DELETE", deletePath(root.ID), "", http.StatusNotFound)
	request("DELETE", "/admin/api/categories/invalid", "", http.StatusBadRequest)
}

func TestCategoryUpdate(t *testing.T) {
	ctx := context.Background()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	root, err := st.CreateCategory(ctx, store.CategoryEntity{Name: "親", SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	child, err := st.CreateCategory(ctx, store.CategoryEntity{Name: "子", ParentID: &root.ID, SortOrder: 20})
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.CreateCategory(ctx, store.CategoryEntity{Name: "別の子", ParentID: &root.ID})
	if err != nil {
		t.Fatal(err)
	}
	blog, err := st.CreateBlog(ctx, store.BlogEntitty{Title: "記事", ParentCategory: root.Name, Category: child.Name, Status: "public"})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	h := Handler{Store: st}
	mux.HandleFunc("PUT /admin/api/categories/{category_id}", h.handlerUpdate)
	put := func(id, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("PUT", "/admin/api/categories/"+id, strings.NewReader(body)))
		if rr.Code != status {
			t.Fatalf("status=%d want=%d body=%s", rr.Code, status, rr.Body.String())
		}
		return rr
	}
	for _, original := range []store.CategoryEntity{root, child} {
		rr := put(fmt.Sprint(original.ID), `{"name":" 更新後 ","sort_order":5}`, http.StatusOK)
		var updated store.CategoryEntity
		if err := json.Unmarshal(rr.Body.Bytes(), &updated); err != nil {
			t.Fatal(err)
		}
		if updated.ID != original.ID || updated.Name != "更新後" || updated.SortOrder != 5 || !reflect.DeepEqual(updated.ParentID, original.ParentID) || updated.CreatedAt != original.CreatedAt || updated.UpdatedAt == original.UpdatedAt {
			t.Fatalf("updated=%#v original=%#v", updated, original)
		}
	}
	storedBlog, err := st.GetBlog(ctx, blog.ID)
	if err != nil || storedBlog.Category != "更新後" || storedBlog.ParentCategory != "更新後" {
		t.Fatalf("blog=%#v err=%v", storedBlog, err)
	}
	put(fmt.Sprint(other.ID), `{"name":"更新後","sort_order":0}`, http.StatusConflict)
	for _, body := range []string{
		`{"name":"　","sort_order":1}`, `{"name":"名前","sort_order":-1}`,
		`{"name":"名前","sort_order":1.5}`, `{"name":"名前","category_key":"changed"}`,
		`{"name":"名前","parent_id":null}`, `{`,
	} {
		put(fmt.Sprint(child.ID), body, http.StatusBadRequest)
	}
	put("999999", `{"name":"名前","sort_order":0}`, http.StatusNotFound)
	put("invalid", `{"name":"名前","sort_order":0}`, http.StatusBadRequest)
	put("0", `{"name":"名前","sort_order":0}`, http.StatusBadRequest)
	unchanged, err := st.GetCategory(ctx, child.ID)
	if err != nil || unchanged.Name != "更新後" || unchanged.SortOrder != 5 {
		t.Fatalf("invalid update persisted: %#v err=%v", unchanged, err)
	}
}

func TestCategoryMoveAPI(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	a, err := st.CreateCategory(ctx, store.CategoryEntity{Name: "A", SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.CreateCategory(ctx, store.CategoryEntity{Name: "B", SortOrder: 10})
	if err != nil {
		t.Fatal(err)
	}
	h := Handler{Store: st}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/api/categories/{category_id}/move", h.handlerMove)
	for _, tc := range []struct {
		id, body string
		status   int
	}{
		{fmt.Sprint(b.ID), `{"direction":"up"}`, http.StatusOK},
		{fmt.Sprint(a.ID), `{"direction":"down"}`, http.StatusOK},
		{fmt.Sprint(a.ID), `{"direction":"invalid"}`, http.StatusBadRequest},
		{fmt.Sprint(a.ID), `{`, http.StatusBadRequest},
		{"999999", `{"direction":"up"}`, http.StatusNotFound},
		{"invalid", `{"direction":"up"}`, http.StatusBadRequest},
	} {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("POST", "/admin/api/categories/"+tc.id+"/move", strings.NewReader(tc.body)))
		if rr.Code != tc.status {
			t.Fatalf("%#v: status=%d body=%s", tc, rr.Code, rr.Body.String())
		}
	}
	items, err := st.ListCategories(ctx)
	if err != nil || len(items) != 2 || items[0].ID != b.ID {
		t.Fatalf("items=%#v err=%v", items, err)
	}
}

func TestCreateHeadingAndUnparentedCategory(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	h := Handler{Store: st}
	create := func(body string) store.CategoryEntity {
		t.Helper()
		rr := httptest.NewRecorder()
		h.handlerCreate(rr, httptest.NewRequest("POST", "/admin/api/categories", strings.NewReader(body)))
		if rr.Code != http.StatusCreated {
			t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
		}
		var item store.CategoryEntity
		if err := json.Unmarshal(rr.Body.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		return item
	}
	heading := create(`{"name":"新しい親","is_heading":true}`)
	orphan := create(`{"name":"親なしの子","is_heading":false}`)
	child := create(fmt.Sprintf(`{"name":"子","parent_id":%d,"is_heading":false}`, heading.ID))
	if !heading.IsHeading || heading.ParentID != nil || orphan.IsHeading || orphan.ParentID != nil || child.IsHeading {
		t.Fatalf("heading=%#v orphan=%#v child=%#v", heading, orphan, child)
	}
	if err := st.DeleteCategory(context.Background(), child.ID); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetCategory(context.Background(), heading.ID)
	if err != nil || !got.IsHeading {
		t.Fatalf("last child removal: %#v %v", got, err)
	}
	rr := httptest.NewRecorder()
	h.handlerCreate(rr, httptest.NewRequest("POST", "/admin/api/categories", strings.NewReader(fmt.Sprintf(`{"name":"不正","parent_id":%d,"is_heading":true}`, heading.ID))))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid heading: status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateSameNamedHeadingAndUnparented(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	h := Handler{Store: st}
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"name":"同名カテゴリ","is_heading":false}`, http.StatusCreated},
		{`{"name":"同名カテゴリ","is_heading":true}`, http.StatusCreated},
		{`{"name":"同名カテゴリ","is_heading":false}`, http.StatusConflict},
		{`{"name":"同名カテゴリ","is_heading":true}`, http.StatusConflict},
	} {
		rr := httptest.NewRecorder()
		h.handlerCreate(rr, httptest.NewRequest("POST", "/admin/api/categories", strings.NewReader(tc.body)))
		if rr.Code != tc.status {
			t.Fatalf("%s: status=%d body=%s", tc.body, rr.Code, rr.Body.String())
		}
	}
}
