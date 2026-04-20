package apiutil

// ErrorResponse は API エラー時の共通レスポンスを表します。
type ErrorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}
