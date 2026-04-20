package web

import (
	"net/http"

	"micro-front/internal/apiutil"
	"micro-front/internal/server"
)

func (h Handler) Init(s *server.Server) {
	staticDir := h.StaticDir
	if staticDir == "" {
		staticDir = "web/static"
	}
	publishDir := h.PublishDir
	if publishDir == "" {
		publishDir = staticDir
	}
	h.PublishDir = publishDir
	s.Handle("/", http.FileServer(http.Dir(staticDir)))
	s.HandleFunc("POST /admin/api/publish", h.handlerPostPublish)
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
