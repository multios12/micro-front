package categories

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"micro-front/internal/apiutil"
	"micro-front/internal/server"
	"micro-front/internal/store"

	"modernc.org/sqlite"
)

func (h Handler) Init(s *server.Server) {
	s.HandleFunc("GET /admin/api/categories", h.handlerList)
	s.HandleFunc("POST /admin/api/categories", h.handlerCreate)
	s.HandleFunc("PUT /admin/api/categories/{category_id}", h.handlerUpdate)
	s.HandleFunc("POST /admin/api/categories/{category_id}/move", h.handlerMove)
	s.HandleFunc("DELETE /admin/api/categories/{category_id}", h.handlerDelete)
}

func (h Handler) handlerMove(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("category_id"), 10, 64)
	if err != nil || id <= 0 {
		apiutil.WriteValidationError(w, map[string]string{"category_id": "カテゴリIDが不正です。"})
		return
	}
	var req struct {
		Direction string `json:"direction"`
	}
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteValidationBodyError(w)
		return
	}
	if req.Direction != "up" && req.Direction != "down" {
		apiutil.WriteValidationError(w, map[string]string{"direction": "移動方向はupまたはdownを指定してください。"})
		return
	}
	if err := (Usecase{Store: h.Store}).Move(r.Context(), id, req.Direction); err != nil {
		writeCategorySaveError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, map[string]string{"result": "moved"})
}

func (h Handler) handlerCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteValidationBodyError(w)
		return
	}
	item, fields, err := (Usecase{Store: h.Store}).Create(r.Context(), req)
	if len(fields) > 0 {
		apiutil.WriteValidationError(w, fields)
		return
	}
	if err != nil {
		writeCategorySaveError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusCreated, item)
}

func (h Handler) handlerUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("category_id"), 10, 64)
	if err != nil || id <= 0 {
		apiutil.WriteValidationError(w, map[string]string{"category_id": "カテゴリIDが不正です。"})
		return
	}
	var req UpdateRequest
	if err := apiutil.DecodeJSON(r, &req); err != nil {
		apiutil.WriteValidationBodyError(w)
		return
	}
	item, fields, err := (Usecase{Store: h.Store}).Update(r.Context(), id, req)
	if len(fields) > 0 {
		apiutil.WriteValidationError(w, fields)
		return
	}
	if err != nil {
		writeCategorySaveError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, item)
}

func writeCategorySaveError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		apiutil.WriteNotFound(w, "カテゴリが見つかりません。")
		return
	}
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() {
		case 2067: // SQLITE_CONSTRAINT_UNIQUE
			apiutil.WriteConflict(w, "CATEGORY_NAME_CONFLICT", "同じグループに同名のカテゴリがあります。", nil)
			return
		case 787, 1811: // FOREIGNKEY / TRIGGER（親の削除・移動との競合）
			apiutil.WriteConflict(w, "INVALID_CATEGORY_PARENT", "親見出しが変更されました。一覧を再読み込みしてください。", nil)
			return
		}
	}
	apiutil.WriteInternalServerError(w, err)
}

func (h Handler) handlerDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("category_id"), 10, 64)
	if err != nil || id <= 0 {
		apiutil.WriteValidationError(w, map[string]string{"category_id": "カテゴリIDが不正です。"})
		return
	}
	err = (Usecase{Store: h.Store}).Delete(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		apiutil.WriteNotFound(w, "カテゴリが見つかりません。")
		return
	}
	if errors.Is(err, store.ErrCategoryInUse) {
		apiutil.WriteConflict(w, "CATEGORY_IN_USE", "記事で使用中のカテゴリは削除できません。", nil)
		return
	}
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && (sqliteErr.Code() == 787 || sqliteErr.Code() == 1811) {
			apiutil.WriteConflict(w, "CATEGORY_HAS_CHILDREN", "カテゴリがある親見出しは削除できません。先に小カテゴリを削除してください。", nil)
			return
		}
		apiutil.WriteInternalServerError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "result": "deleted"})
}

func (h Handler) handlerList(w http.ResponseWriter, r *http.Request) {
	resp, err := (Usecase{Store: h.Store}).List(r.Context())
	if err != nil {
		apiutil.WriteInternalServerError(w, err)
		return
	}
	apiutil.WriteJSON(w, http.StatusOK, resp)
}
