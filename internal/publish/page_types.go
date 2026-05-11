package publish

import "html/template"

// PageTemplateData は公開ページ共通レイアウトに渡すデータです。
type PageTemplateData struct {
	Title               string
	SiteTitle           string
	SiteSubtitle        string
	SiteDescription     string
	SiteDescriptionHTML template.HTML
	HomeURL             string
	Tabs                []PageTab
	Body                template.HTML
	FootInformation     string
	Copyright           string
}

// PageTab は公開ページ共通レイアウト用のタブ情報です。
type PageTab struct {
	TabLabel string
	TabURL   string
	Active   bool
}

// IndexPageData はトップページ専用テンプレートに渡すデータです。
type IndexPageData struct {
	SiteTitle           string
	SiteSubtitle        string
	SiteDescription     string
	SiteDescriptionHTML template.HTML
	HomeURL             string
	Tabs                []PageTab
	LatestPosts         []IndexPostCard
	Categories          []IndexCategoryGroup
	FootInformation     string
	Copyright           string
}

// IndexPostCard はトップページの最新記事カードです。
type IndexPostCard struct {
	Title         string
	Summary       string
	Category      string
	PublishedAt   string
	URL           string
	TitleImageURL string
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
	Title         string
	Summary       string
	Category      string
	PublishedAt   string
	URL           string
	TitleImageURL string
}

// AboutPageData は About ページの body テンプレートに渡すデータです。
type AboutPageData struct {
	Breadcrumbs []PageBreadcrumb
	BodyTitle   string
	Content     template.HTML
	LeadFigure  template.HTML
}

// BlogDetailPageData は記事詳細ページの body テンプレートに渡すデータです。
type BlogDetailPageData struct {
	Breadcrumbs   []PageBreadcrumb
	Title         string
	Meta          template.HTML
	PublishedAt   string
	TitleImageURL string
	Content       template.HTML
}

// PageBreadcrumb は body テンプレートで使うパンくず項目です。
type PageBreadcrumb struct {
	Label string
	URL   string
}
