package config

import "testing"

func TestLoadAdminNavLinks(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		count     int
		invalid   bool
	}{
		{"empty", "", 0, false},
		{"links", `[{"title":"アクセス集計","url":"/access-report/"},{"title":"検索","url":"/tools/?q=a#result"}]`, 2, false},
		{"malformed", `[`, 0, true},
		{"null", `null`, 0, true},
		{"title", `[{"title":" ","url":"/report/"}]`, 0, true},
		{"external", `[{"title":"report","url":"https://example.com/"}]`, 0, true},
		{"host", `[{"title":"report","url":"//example.com/"}]`, 0, true},
		{"backslash", `[{"title":"report","url":"/\\example.com/"}]`, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ADMIN_NAV_LINKS", tc.raw)
			links, err := LoadAdminNavLinks()
			if (err != nil) != tc.invalid {
				t.Fatalf("error = %v", err)
			}
			if !tc.invalid && (links == nil || len(links) != tc.count) {
				t.Fatalf("links = %#v", links)
			}
		})
	}
}
