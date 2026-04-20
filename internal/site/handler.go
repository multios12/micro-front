package site

import (
	"net/http"

	"micro-front/internal/apiutil"
	"micro-front/internal/server"
)

func (h Handler) Init(s *server.Server) {
	s.HandleFunc("GET /admin/api/site", h.handlerGetSite)
	s.HandleFunc("PUT /admin/api/site", h.handlerPutSite)
}

// handlerGetSite は、設計書 3.1 のサイト情報取得APIを処理します。
func (h Handler) handlerGetSite(w http.ResponseWriter, r *http.Request) {
	resp, err := Usecase{Store: h.Store}.Get(r.Context())
	if err != nil {
		apiutil.WriteInternalServerError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, resp)
}

// handlerPutSite は、設計書 3.2 のサイト情報更新APIを処理します。
func (h Handler) handlerPutSite(w http.ResponseWriter, r *http.Request) {
	var req SitePutRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteValidationBodyError(w)
		return
	}

	resp, code, fields, err := Usecase{Store: h.Store}.Put(r.Context(), req)
	if len(fields) > 0 {
		if code == "" {
			code = "VALIDATION_ERROR"
		}
		apiutil.WriteValidationErrorCode(w, code, fields)
		return
	}
	if err != nil {
		apiutil.WriteInternalServerError(w, err)
		return
	}

	apiutil.WriteJSON(w, http.StatusOK, resp)
}
