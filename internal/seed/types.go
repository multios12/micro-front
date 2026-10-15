package seed

import "micro-front/internal/store"

type Options struct {
	SeedDir string
	Reset   bool
}

type SiteSeed struct {
	SiteTitle       string      `json:"site_title"`
	SiteSubtitle    string      `json:"site_subtitle"`
	SiteDescription string      `json:"site_description"`
	SiteURL         string      `json:"site_url"`
	Tabs            []store.Tab `json:"tabs"`
	FootInformation string      `json:"foot_information"`
	Copyright       string      `json:"copyright"`
}

type BlogSeed struct {
	ID               int64  `json:"id"`
	Title            string `json:"title"`
	Content          string `json:"content"`
	ContentFile      string `json:"content_file"`
	Category         string `json:"category"`
	ParentCategory   string `json:"parent_category"`
	Status           string `json:"status"`
	PublishedAt      string `json:"published_at"`
	UseSampleContent bool   `json:"use_sample_content"`
}

// CategorySeed のキーはファイル内だけの参照名で、DBには保存しません。
// ParentKey は親の CategoryKey を参照します。空の場合は親見出しです。
type CategorySeed struct {
	IsHeading   *bool  `json:"is_heading"`
	Name        string `json:"name"`
	CategoryKey string `json:"category_key"`
	ParentKey   string `json:"parent_key"`
	SortOrder   int    `json:"sort_order"`
}

type ImageSeed struct {
	ID      int64  `json:"id"`
	BlogID  int64  `json:"blog_id"`
	AltText string `json:"alt_text"`
	File    string `json:"file"`
}
