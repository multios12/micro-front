package categories

import "micro-front/internal/store"

// Handler はカテゴリ管理APIのHTTPハンドラです。
type Handler struct {
	Store *store.Store
}

// Usecase はカテゴリの取得・追加・更新・削除を扱います。
type Usecase struct {
	Store *store.Store
}

// ListResponse は親見出しとその小見出しを表示順に並べた一覧です。
type ListResponse struct {
	Items []store.CategoryEntity `json:"items"`
}

type CreateRequest struct {
	ParentID  *int64 `json:"parent_id"`
	IsHeading *bool  `json:"is_heading"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type UpdateRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}
