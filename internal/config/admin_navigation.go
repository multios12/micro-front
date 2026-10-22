package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// AdminNavLink は管理画面の追加リンクです。
type AdminNavLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// LoadAdminNavLinks は起動時に環境変数を検証します。
func LoadAdminNavLinks() ([]AdminNavLink, error) {
	links := []AdminNavLink{}
	raw := os.Getenv("ADMIN_NAV_LINKS")
	if strings.TrimSpace(raw) == "" {
		return links, nil
	}
	if err := json.Unmarshal([]byte(raw), &links); err != nil {
		return nil, fmt.Errorf("ADMIN_NAV_LINKS: %w", err)
	}
	if links == nil {
		return nil, fmt.Errorf("ADMIN_NAV_LINKS: JSON配列を指定してください")
	}
	for i, l := range links {
		u, err := url.Parse(l.URL)
		if strings.TrimSpace(l.Title) == "" || err != nil || !strings.HasPrefix(l.URL, "/") || strings.HasPrefix(l.URL, "//") || strings.ContainsAny(l.URL, "\\\r\n\t") {
			return nil, fmt.Errorf("ADMIN_NAV_LINKS[%d]: 空でないtitleと / 始まりの同一ドメイン内URLを指定してください", i)
		}
		if u.IsAbs() || u.Host != "" {
			return nil, fmt.Errorf("ADMIN_NAV_LINKS[%d]: 同一ドメイン内URLを指定してください", i)
		}
	}
	return links, nil
}
