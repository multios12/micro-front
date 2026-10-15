package publish

import (
	"reflect"
	"strings"
	"testing"

	"micro-front/internal/store"
)

func TestIndexCategoriesUsesRegisteredHierarchy(t *testing.T) {
	parentID := int64(1)
	secondID := int64(4)
	categories := []store.CategoryEntity{
		{ID: 1, Name: "親/ハイフン-付き", IsHeading: true},
		{ID: 2, Name: "子-B", ParentID: &parentID},
		{ID: 3, Name: "子/A", ParentID: &parentID},
		{ID: 4, Name: "別の親", IsHeading: true},
		{ID: 5, Name: "子/A", ParentID: &secondID},
		{ID: 6, Name: "親なし-カテゴリ"},
		{ID: 7, Name: "公開記事なし", IsHeading: true},
		{ID: 8, Name: "親のみ", IsHeading: true},
	}
	blogs := []store.BlogEntitty{
		{ParentCategory: categories[0].Name, Category: "子/A"},
		{ParentCategory: categories[0].Name, Category: "子-B"},
		{ParentCategory: categories[0].Name, Category: "子-B"},
		{ParentCategory: categories[0].Name},
		{ParentCategory: "別の親", Category: "子/A"},
		{Category: "親なし-カテゴリ"},
		{ParentCategory: "親のみ"},
		{Title: "about", ParentCategory: "公開記事なし"},
	}
	got := buildIndexCategories(blogs, categories)
	if len(got) != 4 {
		t.Fatalf("groups: %#v", got)
	}
	if got[0].Name != categories[0].Name || got[0].Count != 4 || !got[0].Open {
		t.Fatalf("parent: %#v", got[0])
	}
	want := []IndexCategoryCard{
		{Name: "子-B", Count: 2, URL: "./blogs/category/子-B/index.html"},
		{Name: "子/A", Count: 1, URL: "./blogs/category/子-A/index.html"},
	}
	if !reflect.DeepEqual(got[0].Children, want) {
		t.Fatalf("children: %#v", got[0].Children)
	}
	if got[1].Count != 1 || len(got[1].Children) != 1 || got[1].Children[0].Count != 1 {
		t.Fatalf("scoped count: %#v", got[1])
	}
	if got[2].Name != "親なし-カテゴリ" || got[2].IsHeading || len(got[2].Children) != 0 {
		t.Fatalf("orphan: %#v", got[2])
	}
	html, err := renderIndexDocument(IndexPageData{HomeURL: "./index.html", Categories: got})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `<details class="category-group" open>`) || !strings.Contains(html, `<div class="category-branch">`) {
		t.Fatal("hierarchy markup missing")
	}
	if strings.Contains(html, `href=""`) {
		t.Fatal("heading has an empty link")
	}
	if strings.Contains(html, "公開記事なし") {
		t.Fatal("empty category was published")
	}
}
