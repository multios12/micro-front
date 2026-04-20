package images

import "micro-front/internal/store"

// Handler は画像関連 API の HTTP ハンドラです。
type Handler struct {
	Store   *store.Store
	DataDir string
}

// Usecase は画像のアップロード・削除を扱うユースケースです。
type Usecase struct {
	Store   *store.Store
	DataDir string
}

// ImagesUploadResponse は画像アップロードAPIのレスポンスです。
type ImagesUploadResponse struct {
	Result  string `json:"result"`
	URL     string `json:"url"`
	AltText string `json:"alt_text"`
}

// ImagesDeleteResponse は画像削除APIのレスポンスです。
type ImagesDeleteResponse struct {
	ID     int64  `json:"id"`
	BlogID int64  `json:"blog_id"`
	Result string `json:"result"`
}

// ImagesListItemResponse は画像一覧APIの1件を表します。
type ImagesListItemResponse struct {
	ID        int64  `json:"id"`
	BlogID    int64  `json:"blog_id"`
	URL       string `json:"url"`
	AltText   string `json:"alt_text"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ImagesListResponse は画像一覧APIのレスポンスです。
type ImagesListResponse struct {
	Items []ImagesListItemResponse `json:"items"`
}
