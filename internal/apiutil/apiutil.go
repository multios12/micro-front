package apiutil

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	WriteJSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
		Fields:  fields,
	})
}

func WriteValidationBodyError(w http.ResponseWriter) {
	WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "JSON ボディが不正です", nil)
}

func WriteValidationError(w http.ResponseWriter, fields map[string]string) {
	WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "入力内容を確認してください", fields)
}

func WriteValidationErrorCode(w http.ResponseWriter, code string, fields map[string]string) {
	WriteError(w, http.StatusBadRequest, code, "入力内容を確認してください", fields)
}

func WriteNotFound(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusNotFound, "NOT_FOUND", message, nil)
}

func WriteConflict(w http.ResponseWriter, code, message string, fields map[string]string) {
	WriteError(w, http.StatusConflict, code, message, fields)
}

func WriteInternalServerError(w http.ResponseWriter, err error) {
	WriteError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", err.Error(), nil)
}

func DecodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
