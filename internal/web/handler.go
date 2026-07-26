package web

import (
	_ "embed"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"micro-front/internal/apiutil"
	"micro-front/internal/publish"
	"micro-front/internal/server"
)

//go:embed admin.html
var embeddedAdminHTML []byte

func (h Handler) Init(s *server.Server) {
	staticDir := h.StaticDir
	if staticDir == "" {
		staticDir = "web/static"
	}
	dataDir := h.DataDir
	if dataDir == "" {
		dataDir = "./data"
	}
	publishDir := h.PublishDir
	if publishDir == "" {
		publishDir = staticDir
	}
	previewDir := h.PreviewDir
	if previewDir == "" {
		previewDir = filepath.Join(dataDir, "preview")
	}
	h.PublishDir = publishDir
	h.PreviewDir = previewDir
	h.DataDir = dataDir
	s.HandleFunc("GET  /{$}", serveAdminHTML)
	s.HandleFunc("GET  /", http.FileServer(http.Dir(staticDir)).ServeHTTP)
	s.HandleFunc("POST /admin/api/publish", h.handlerPostPublish)
	s.HandleFunc("POST /admin/api/site/preview", h.handlerPostSitePreview)
	s.HandleFunc("GET  /admin/preview/", http.StripPrefix("/admin/preview/", http.FileServer(http.Dir(previewDir))).ServeHTTP)
	s.HandleFunc("POST /admin/api/blogs/{blog_id}/preview", h.handlerPostBlogPreview)
}

func serveAdminHTML(w http.ResponseWriter, r *http.Request) {
	adminPath := filepath.Join("internal", "web", "admin.html")
	if _, err := os.Stat(adminPath); err == nil {
		http.ServeFile(w, r, adminPath)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(embeddedAdminHTML)
}

// handlerPostPublish は設計書 3.10 の静的HTML出力APIを処理します。
func (h Handler) handlerPostPublish(w http.ResponseWriter, r *http.Request) {
	var req WebPublishRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "JSON ボディが不正です", nil)
		return
	}

	publishDir := h.PublishDir
	if publishDir == "" {
		publishDir = h.StaticDir
	}
	resp, fields, err := publish.Usecase{Store: h.Store, PublishDir: publishDir}.Run(r.Context(), publish.Request(req))
	if len(fields) > 0 {
		apiutil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "入力内容を確認してください", fields)
		return
	}
	if err != nil {
		apiutil.WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error(), nil)
		return
	}

	apiutil.WriteJSON(w, http.StatusOK, resp)
}

// handlerPostSitePreview はトップページプレビュー生成APIを処理します。
func (h Handler) handlerPostSitePreview(w http.ResponseWriter, r *http.Request) {
	previewDir := h.PreviewDir
	if previewDir == "" {
		previewDir = filepath.Join(h.DataDir, "preview")
	}
	resp, fields, err := publish.Usecase{
		Store: h.Store,
	}.PreviewIndex(r.Context(), previewDir)
	if len(fields) > 0 {
		apiutil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "入力内容を確認してください", fields)
		return
	}
	if err != nil {
		apiutil.WriteInternalServerError(w, err)
		return
	}

	apiutil.WriteJSON(w, http.StatusOK, resp)
}

func parseBlogID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

// handlerPostBlogPreview は記事プレビュー生成APIを処理します。
func (h Handler) handlerPostBlogPreview(w http.ResponseWriter, r *http.Request) {
	blogID := r.PathValue("blog_id")

	var req WebPreviewRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteValidationBodyError(w)
		return
	}
	if blogID != "about" {
		id, err := parseBlogID(blogID)
		if err != nil {
			apiutil.WriteNotFound(w, "not found")
			return
		}
		if req.BlogID != 0 && req.BlogID != id {
			apiutil.WriteValidationErrorCode(w, "VALIDATION_ERROR", map[string]string{
				"blog_id": "記事IDが不正です",
			})
			return
		}

		previewDir := h.PreviewDir
		if previewDir == "" {
			previewDir = filepath.Join(h.DataDir, "preview")
		}
		resp, fields, err := publish.Usecase{
			Store: h.Store,
		}.PreviewBlog(r.Context(), id, previewDir)
		if len(fields) > 0 {
			apiutil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "入力内容を確認してください", fields)
			return
		}
		if err != nil {
			apiutil.WriteInternalServerError(w, err)
			return
		}

		apiutil.WriteJSON(w, http.StatusOK, resp)
		return
	}

	if req.BlogID != 0 {
		apiutil.WriteValidationErrorCode(w, "VALIDATION_ERROR", map[string]string{
			"blog_id": "記事IDが不正です",
		})
		return
	}

	previewDir := h.PreviewDir
	if previewDir == "" {
		previewDir = filepath.Join(h.DataDir, "preview")
	}
	resp, fields, err := publish.Usecase{
		Store: h.Store,
	}.PreviewAbout(r.Context(), previewDir)
	if len(fields) > 0 {
		apiutil.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "入力内容を確認してください", fields)
		return
	}
	if err != nil {
		apiutil.WriteInternalServerError(w, err)
		return
	}

	apiutil.WriteJSON(w, http.StatusOK, resp)
}
