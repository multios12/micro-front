package web

import (
	"html/template"

	"micro-front/internal/store"
)

// Handler は静的HTML出力APIと公開ファイル配信の HTTP ハンドラです。
type Handler struct {
	Store      *store.Store
	StaticDir  string
	PublishDir string
}

// Usecase は静的HTMLの生成処理を扱うユースケースです。
type Usecase struct {
	Store      *store.Store
	StaticDir  string
	PublishDir string
}

// PageTemplateData は公開ページ共通レイアウトに渡すデータです。
type PageTemplateData struct {
	Title           string
	SiteTitle       string
	SiteSubtitle    string
	SiteDescription string
	HomeURL         string
	Tabs            []PageTab
	Body            template.HTML
	FootInformation string
	Copyright       string
}

// PageTab は公開ページ共通レイアウト用のタブ情報です。
type PageTab struct {
	TabLabel string
	TabURL   string
	Active   bool
}

// IndexPageData はトップページ専用テンプレートに渡すデータです。
type IndexPageData struct {
	SiteTitle       string
	SiteSubtitle    string
	SiteDescription string
	HomeURL         string
	Tabs            []PageTab
	LatestPosts     []IndexPostCard
	Categories      []IndexCategoryGroup
	Copyright       string
}

// IndexPostCard はトップページの最新記事カードです。
type IndexPostCard struct {
	Title       string
	Summary     string
	Category    string
	PublishedAt string
	URL         string
}

// IndexCategoryCard はトップページのカテゴリ一覧項目です。
type IndexCategoryCard struct {
	Name  string
	Count int
	URL   string
}

// IndexCategoryGroup はトップページのカテゴリグループです。
type IndexCategoryGroup struct {
	Name     string
	Count    int
	URL      string
	Children []IndexCategoryCard
	Open     bool
}

// BlogListPageData は一覧ページテンプレートに渡すデータです。
type BlogListPageData struct {
	Breadcrumbs []PageBreadcrumb
	Kicker      string
	Heading     string
	Items       []BlogListCard
	Pagination  template.HTML
}

// BlogListCard は一覧ページの記事カードです。
type BlogListCard struct {
	Title     string
	Summary   string
	Category  string
	UpdatedAt string
	URL       string
}

// AboutPageData は About ページの body テンプレートに渡すデータです。
type AboutPageData struct {
	BodyTitle  string
	Content    template.HTML
	LeadFigure template.HTML
}

// BlogDetailPageData は記事詳細ページの body テンプレートに渡すデータです。
type BlogDetailPageData struct {
	Breadcrumbs []PageBreadcrumb
	Title       string
	Meta        template.HTML
	UpdatedAt   string
	LeadFigure  template.HTML
	Content     template.HTML
}

// PageBreadcrumb は body テンプレートで使うパンくず項目です。
type PageBreadcrumb struct {
	Label string
	URL   string
}

// WebPublishRequest は静的HTML出力APIのリクエストです。
type WebPublishRequest struct {
	Target string `json:"target"`
	BlogID int64  `json:"blog_id"`
}

// WebPublishResponse は静的HTML出力APIのレスポンスです。
type WebPublishResponse struct {
	Result string `json:"result"`
}
