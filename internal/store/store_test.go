package store

import (
	"context"
	"testing"
)

func TestNewSeedsDefaultSiteTabs(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	settings, err := st.GetSiteSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSiteSettings: %v", err)
	}

	want := []Tab{
		{TabLabel: "Home", TabURL: "/"},
		{TabLabel: "Blogs", TabURL: "/blogs"},
		{TabLabel: "About", TabURL: "/about"},
	}
	if len(settings.Tabs) != len(want) {
		t.Fatalf("tabs length=%d want %d: %#v", len(settings.Tabs), len(want), settings.Tabs)
	}
	for i := range want {
		if settings.Tabs[i] != want[i] {
			t.Fatalf("tabs[%d]=%#v want %#v", i, settings.Tabs[i], want[i])
		}
	}
}
