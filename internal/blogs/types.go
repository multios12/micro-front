package blogs

import "micro-front/internal/store"

// Handler は記事関連 API の HTTP ハンドラです。
type Handler struct {
	Store   *store.Store
	DataDir string
}

// Usecase は記事の一覧・詳細・更新を扱うユースケースです。
type Usecase struct {
	Store   *store.Store
	DataDir string
}

// BlogsListItemResponse は記事一覧APIの1件分のレスポンスです。
type BlogsListItemResponse struct {
	ID                 int64  `json:"id"`
	Title              string `json:"title"`
	Summary            string `json:"summary"`
	Category           string `json:"category"`
	Status             string `json:"status"`
	TitleImageTemplate string `json:"title_image_template"`
	PublishedAt        string `json:"published_at"`
	UpdatedAt          string `json:"updated_at"`
}

// BlogsListResponse は記事一覧APIのレスポンスです。
type BlogsListResponse struct {
	Items      []BlogsListItemResponse `json:"items"`
	Page       int                     `json:"page"`
	PerPage    int                     `json:"per_page"`
	Total      int                     `json:"total"`
	TotalPages int                     `json:"total_pages"`
}

// BlogsDetailResponse は記事詳細APIのレスポンスです。
type BlogsDetailResponse struct {
	ID                 int64  `json:"id"`
	Title              string `json:"title"`
	Content            string `json:"content"`
	Summary            string `json:"summary"`
	Category           string `json:"category"`
	Status             string `json:"status"`
	TitleImageTemplate string `json:"title_image_template"`
	PublishedAt        string `json:"published_at"`
	UpdatedAt          string `json:"updated_at"`
}

// BlogsCreateRequest は記事新規作成APIのリクエストです。
type BlogsCreateRequest struct {
	Title              string `json:"title"`
	Content            string `json:"content"`
	Category           string `json:"category"`
	Status             string `json:"status"`
	TitleImageTemplate string `json:"title_image_template"`
	PublishedAt        string `json:"published_at"`
}

// BlogsCreateResponse は記事新規作成APIのレスポンスです。
type BlogsCreateResponse struct {
	ID                 int64  `json:"id"`
	Title              string `json:"title"`
	Content            string `json:"content"`
	Summary            string `json:"summary"`
	Category           string `json:"category"`
	Status             string `json:"status"`
	TitleImageTemplate string `json:"title_image_template"`
	PublishedAt        string `json:"published_at"`
	UpdatedAt          string `json:"updated_at"`
}

// BlogsUpdateRequest は記事更新APIのリクエストです。
type BlogsUpdateRequest struct {
	Title              string `json:"title"`
	Content            string `json:"content"`
	Category           string `json:"category"`
	Status             string `json:"status"`
	TitleImageTemplate string `json:"title_image_template"`
	PublishedAt        string `json:"published_at"`
}

// BlogsUpdateResponse は記事更新APIのレスポンスです。
type BlogsUpdateResponse struct {
	ID                 int64  `json:"id"`
	Title              string `json:"title"`
	Content            string `json:"content"`
	Summary            string `json:"summary"`
	Category           string `json:"category"`
	Status             string `json:"status"`
	TitleImageTemplate string `json:"title_image_template"`
	PublishedAt        string `json:"published_at"`
	UpdatedAt          string `json:"updated_at"`
}

// BlogsDeleteResponse は記事削除APIのレスポンスです。
type BlogsDeleteResponse struct {
	ID     int64  `json:"id"`
	Result string `json:"result"`
}
