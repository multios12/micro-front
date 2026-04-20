package blogs

import (
	"errors"
	"net/http"
	"strconv"

	"micro-front/internal/apiutil"
	"micro-front/internal/server"
)

func (h Handler) Init(s *server.Server) {
	s.HandleFunc("GET /admin/api/blogs", h.handlerGetBlogs)
	s.HandleFunc("POST /admin/api/blogs", h.handlerPostBlogs)
	s.HandleFunc("GET /admin/api/blogs/{blog_id}", h.handlerGetBlog)
	s.HandleFunc("PUT /admin/api/blogs/{blog_id}", h.handlerPutBlog)
	s.HandleFunc("DELETE /admin/api/blogs/{blog_id}", h.handlerDeleteBlog)
}

// handlerGetBlogs は、設計書 3.3 の記事一覧取得APIを処理します。
func (h Handler) handlerGetBlogs(w http.ResponseWriter, r *http.Request) {
	page := readIntQuery(r, "page", 1)
	perPage := readIntQuery(r, "per_page", 20)
	status := r.URL.Query().Get("status")

	resp, code, fields, err := Usecase{Store: h.Store, DataDir: h.DataDir}.List(r.Context(), page, perPage, status)
	if len(fields) > 0 {
		apiutil.WriteValidationErrorCode(w, code, fields)
		return
	}
	if err != nil {
		apiutil.WriteInternalServerError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, resp)
}

// handlerGetBlog は、設計書 3.4 の記事詳細取得APIを処理します。
func (h Handler) handlerGetBlog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("blog_id"), 10, 64)
	if err != nil {
		apiutil.WriteNotFound(w, "not found")
		return
	}
	resp, err := Usecase{Store: h.Store, DataDir: h.DataDir}.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, errBlogNotFound) {
			apiutil.WriteNotFound(w, "blog not found")
			return
		}
		apiutil.WriteInternalServerError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, resp)
}

// handlerPostBlogs は、設計書 3.5 の記事新規作成APIを処理します。
func (h Handler) handlerPostBlogs(w http.ResponseWriter, r *http.Request) {
	var req BlogsCreateRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteValidationBodyError(w)
		return
	}

	resp, code, fields, err := Usecase{Store: h.Store, DataDir: h.DataDir}.Create(r.Context(), req)
	if len(fields) > 0 {
		apiutil.WriteValidationErrorCode(w, code, fields)
		return
	}
	if err != nil {
		if errors.Is(err, errBlogTitleConflict) {
			apiutil.WriteConflict(w, "BLOG_TITLE_CONFLICT", "同じタイトルの記事がすでに存在します", map[string]string{
				"title": "同じタイトルの記事がすでに存在します",
			})
			return
		}
		apiutil.WriteInternalServerError(w, err)
		return
	}

	apiutil.WriteJSON(w, http.StatusCreated, resp)
}

// handlerPutBlog は、設計書 3.6 の記事更新APIを処理します。
func (h Handler) handlerPutBlog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("blog_id"), 10, 64)
	if err != nil {
		apiutil.WriteNotFound(w, "not found")
		return
	}
	var req BlogsUpdateRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteValidationBodyError(w)
		return
	}

	resp, code, fields, err := Usecase{Store: h.Store, DataDir: h.DataDir}.Update(r.Context(), id, req)
	if len(fields) > 0 {
		apiutil.WriteValidationErrorCode(w, code, fields)
		return
	}
	if err != nil {
		if errors.Is(err, errBlogNotFound) {
			apiutil.WriteNotFound(w, "blog not found")
			return
		}
		if errors.Is(err, errBlogTitleConflict) {
			apiutil.WriteConflict(w, "BLOG_TITLE_CONFLICT", "同じタイトルの記事がすでに存在します", map[string]string{
				"title": "同じタイトルの記事がすでに存在します",
			})
			return
		}
		apiutil.WriteInternalServerError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, resp)
}

// handlerDeleteBlog は、設計書 3.7 の記事削除APIを処理します。
func (h Handler) handlerDeleteBlog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("blog_id"), 10, 64)
	if err != nil {
		apiutil.WriteNotFound(w, "not found")
		return
	}
	resp, err := Usecase{Store: h.Store, DataDir: h.DataDir}.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, errBlogNotFound) {
			apiutil.WriteNotFound(w, "blog not found")
			return
		}
		apiutil.WriteInternalServerError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, resp)
}

// readIntQuery は数値クエリを読み取り、異常時は既定値を返します。
func readIntQuery(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return fallback
	}
	return v
}
