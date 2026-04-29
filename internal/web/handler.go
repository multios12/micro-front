package web

import (
	_ "embed"
	"net/http"
	"os"
	"path/filepath"

	"micro-front/internal/apiutil"
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
	h.PublishDir = publishDir
	h.DataDir = dataDir
	s.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		serveAdminHTML(w, r)
	})
	s.Handle("/", http.FileServer(http.Dir(staticDir)))
	s.HandleFunc("POST /admin/api/publish", h.handlerPostPublish)
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
	resp, fields, err := Usecase{Store: h.Store, StaticDir: h.StaticDir, PublishDir: publishDir}.Publish(r.Context(), req)
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
