package blogs

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

func TestBlogSaveRejectsUnregisteredCategory(t *testing.T) {
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	root, err := st.CreateCategory(ctx, store.CategoryEntity{Name: "親", IsHeading: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateCategory(ctx, store.CategoryEntity{Name: "子", ParentID: &root.ID}); err != nil {
		t.Fatal(err)
	}
	before, _ := st.ListCategories(ctx)
	h := Handler{Store: st, DataDir: st.DataDir}
	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if method == "POST" {
			h.handlerPostBlogs(rr, req)
		} else {
			req.SetPathValue("blog_id", strings.TrimPrefix(path, "/admin/api/blogs/"))
			h.handlerPutBlog(rr, req)
		}
		if rr.Code != status {
			t.Fatalf("status=%d want=%d body=%s", rr.Code, status, rr.Body.String())
		}
		return rr
	}
	for _, pair := range [][2]string{{"未登録の親", "子"}, {"親", "未登録の子"}, {"", "子"}} {
		body := fmt.Sprintf(`{"title":"未登録","content":"本文","parent_category":%q,"category":%q,"status":"private","published_at":"2026-10-15"}`, pair[0], pair[1])
		rr := request("POST", "/admin/api/blogs", body, http.StatusBadRequest)
		var resp apiutil.ErrorResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil || resp.Code != "CATEGORY_NOT_FOUND" || resp.Fields["category"] == "" {
			t.Fatalf("error=%#v err=%v", resp, err)
		}
	}
	rr := request("POST", "/admin/api/blogs", `{"title":"登録済み","content":"本文","parent_category":"親","category":"子","status":"private","published_at":"2026-10-15"}`, http.StatusCreated)
	var blog BlogsCreateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &blog); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/admin/api/blogs/%d", blog.ID)
	request("PUT", path, `{"title":"登録済み","content":"変更されない本文","parent_category":"親","category":"未登録の子","status":"private","published_at":"2026-10-15"}`, http.StatusBadRequest)
	got, err := st.GetBlog(ctx, blog.ID)
	if err != nil || got.Category != "子" || got.Content != "本文" {
		t.Fatalf("failed save modified article: %#v %v", got, err)
	}
	request("PUT", path, `{"title":"登録済み","content":"本文","parent_category":"","category":"","status":"private","published_at":"2026-10-15"}`, http.StatusOK)
	got, _ = st.GetBlog(ctx, blog.ID)
	if got.ParentCategory != "" || got.Category != "" {
		t.Fatalf("unselected not saved: %#v", got)
	}
	after, err := st.ListCategories(ctx)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("article save changed categories: %#v %v", after, err)
	}
}
