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
	Status           string `json:"status"`
	PublishedAt      string `json:"published_at"`
	UseSampleContent bool   `json:"use_sample_content"`
}

type ImageSeed struct {
	ID      int64  `json:"id"`
	BlogID  int64  `json:"blog_id"`
	AltText string `json:"alt_text"`
	File    string `json:"file"`
}
