package site

import "micro-front/internal/store"

// Handler はサイト情報 API の HTTP ハンドラです。
type Handler struct {
	Store *store.Store
}

// Usecase はサイト情報の取得・更新を扱うユースケースです。
type Usecase struct {
	Store *store.Store
}

// SiteGetResponse はサイト情報取得APIのレスポンスです。
type SiteGetResponse struct {
	ID              int64       `json:"id"`
	SiteTitle       string      `json:"site_title"`
	SiteSubtitle    string      `json:"site_subtitle"`
	SiteDescription string      `json:"site_description"`
	SiteURL         string      `json:"site_url"`
	Tabs            []store.Tab `json:"tabs"`
	FootInformation string      `json:"foot_information"`
	Copyright       string      `json:"copyright"`
	UpdatedAt       string      `json:"updated_at"`
}

// SitePutRequest はサイト情報更新APIのリクエストです。
type SitePutRequest struct {
	SiteTitle       string      `json:"site_title"`
	SiteSubtitle    string      `json:"site_subtitle"`
	SiteDescription string      `json:"site_description"`
	SiteURL         string      `json:"site_url"`
	Tabs            []store.Tab `json:"tabs"`
	FootInformation string      `json:"foot_information"`
	Copyright       string      `json:"copyright"`
}

// SitePutResponse はサイト情報更新APIのレスポンスです。
type SitePutResponse struct {
	ID              int64       `json:"id"`
	SiteTitle       string      `json:"site_title"`
	SiteSubtitle    string      `json:"site_subtitle"`
	SiteDescription string      `json:"site_description"`
	SiteURL         string      `json:"site_url"`
	Tabs            []store.Tab `json:"tabs"`
	FootInformation string      `json:"foot_information"`
	Copyright       string      `json:"copyright"`
	UpdatedAt       string      `json:"updated_at"`
}
