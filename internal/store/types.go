package store

import "database/sql"

// Store は SQLite とファイル保存先を扱う永続化層です。
type Store struct {
	DB      *sql.DB
	DataDir string
}

// CategoryEntity は親見出しまたは小見出しのDBモデルです。
// IsHeading は子の有無にかかわらず親見出しであることを表します。
type CategoryEntity struct {
	ID        int64  `json:"id"`
	ParentID  *int64 `json:"parent_id"`
	IsHeading bool   `json:"is_heading"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Tab はサイト上部のタブメニュー1件を表します。
type Tab struct {
	TabLabel string `json:"tab_label"`
	TabURL   string `json:"tab_url"`
}

// SiteEntitty はサイト全体設定のDBモデルです。
type SiteEntitty struct {
	ID              int64  `json:"id"`
	SiteTitle       string `json:"site_title"`
	SiteSubtitle    string `json:"site_subtitle"`
	SiteDescription string `json:"site_description"`
	SiteURL         string `json:"site_url"`
	Tabs            []Tab  `json:"tabs"`
	FootInformation string `json:"foot_information"`
	Copyright       string `json:"copyright"`
	UpdatedAt       string `json:"updated_at"`
}

// BlogEntitty は記事のDBモデルです。
type BlogEntitty struct {
	ID                 int64          `json:"id"`
	Title              string         `json:"title"`
	Content            string         `json:"content"`
	Summary            string         `json:"summary"`
	ParentCategory     string         `json:"parent_category"`
	Category           string         `json:"category"`
	Status             string         `json:"status"`
	TitleImageTemplate string         `json:"title_image_template"`
	PublishedAt        string         `json:"published_at"`
	UpdatedAt          string         `json:"updated_at"`
	Images             []ImageEntitty `json:"images,omitempty"`
}

// ImageEntitty は記事に紐づく画像のDBモデルです。
type ImageEntitty struct {
	ID        int64  `json:"id"`
	BlogID    int64  `json:"blog_id"`
	AltText   string `json:"alt_text"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// BlogListFilter は記事一覧取得時の絞り込み条件です。
type BlogListFilter struct {
	Page     int
	PerPage  int
	Status   string
	Category string
}

// BlogListResult は記事一覧取得の結果を表します。
type BlogListResult struct {
	Items      []BlogEntitty
	Page       int
	PerPage    int
	Total      int
	TotalPages int
}
