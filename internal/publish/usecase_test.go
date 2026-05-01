package publish

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"micro-front/internal/store"
)

func TestPublish_RendersMarkdownWithPkgMarkdown(t *testing.T) {
	ctx := context.Background()

	dataDir := t.TempDir()
	s, err := store.New(dataDir)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	blog, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "sample-post",
		Content:     "# Hello\n\nThis is **bold** and *italic*.",
		Summary:     "summary",
		Category:    "news",
		Status:      "public",
		PublishedAt: "2026-04-19 00:00:00",
	})
	if err != nil {
		t.Fatalf("CreateBlog: %v", err)
	}
	if _, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "about",
		Content:     "## Profile\n\nThis is **about**.",
		Summary:     "about summary",
		Category:    "profile",
		Status:      "public",
		PublishedAt: "2026-04-18 00:00:00",
	}); err != nil {
		t.Fatalf("CreateBlog about: %v", err)
	}

	uc := Usecase{Store: s, PublishDir: filepath.Join(dataDir, "public")}
	if _, _, err := uc.Run(ctx, Request{Target: "blog", BlogID: blog.ID}); err != nil {
		t.Fatalf("Run blog: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(uc.PublishDir, "blogs", formatBlogFileName(blog.ID)))
	if err != nil {
		t.Fatalf("ReadFile blog: %v", err)
	}
	body := string(got)
	for _, want := range []string{"<h1>Hello</h1>", "<strong>bold</strong>", "<em>italic</em>", "href=\"../index.html\"", "href=\"./index.html\""} {
		if !strings.Contains(body, want) {
			t.Fatalf("blog page missing %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, formatDateOnly(blog.PublishedAt)) {
		t.Fatalf("blog page did not show publishedAt near title:\n%s", body)
	}
	if strings.Contains(body, "# Hello") || strings.Contains(body, "**bold**") || strings.Contains(body, "*italic*") {
		t.Fatalf("blog page still contains raw markdown:\n%s", body)
	}

	if _, _, err := uc.Run(ctx, Request{Target: "about"}); err != nil {
		t.Fatalf("Run about: %v", err)
	}
	got, err = os.ReadFile(filepath.Join(uc.PublishDir, "about", "index.html"))
	if err != nil {
		t.Fatalf("ReadFile about: %v", err)
	}
	body = string(got)
	for _, want := range []string{"<h2>Profile</h2>", "<strong>about</strong>"} {
		if !strings.Contains(body, want) {
			t.Fatalf("about page missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "## Profile") || strings.Contains(body, "**about**") {
		t.Fatalf("about page still contains raw markdown:\n%s", body)
	}
}

func TestPublishBlogs_RegeneratesPagesAndAssets(t *testing.T) {
	t.Setenv("TOP_PAGE_BLOG_LIMIT", "1")
	t.Setenv("BLOGS_PER_PAGE", "1")
	t.Setenv("SITE_URL", "https://example.com")
	ctx := context.Background()
	dataDir := t.TempDir()
	s, err := store.New(dataDir)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	blog1, err := s.CreateBlog(ctx, store.BlogEntitty{Title: "first-post", Content: "# First", Summary: "first summary", Category: "news", Status: "public", PublishedAt: "2026-04-19 00:00:00"})
	if err != nil {
		t.Fatalf("CreateBlog 1: %v", err)
	}
	img, err := s.CreateImage(ctx, blog1.ID, "photo")
	if err != nil {
		t.Fatalf("CreateImage: %v", err)
	}
	if err := writeTestPNG(filepath.Join(dataDir, "images", strconv.FormatInt(blog1.ID, 10), strconv.FormatInt(img.ID, 10)+".png")); err != nil {
		t.Fatalf("writeTestPNG: %v", err)
	}
	if _, err := s.UpdateBlog(ctx, blog1.ID, store.BlogEntitty{Title: blog1.Title, Content: "# First\n\n![photo](/admin/images/" + strconv.FormatInt(blog1.ID, 10) + "/" + strconv.FormatInt(img.ID, 10) + ".png)", Summary: blog1.Summary, Category: blog1.Category, Status: blog1.Status, PublishedAt: blog1.PublishedAt}); err != nil {
		t.Fatalf("UpdateBlog 1: %v", err)
	}
	blog1, err = s.GetBlog(ctx, blog1.ID)
	if err != nil {
		t.Fatalf("GetBlog 1: %v", err)
	}
	blog2, err := s.CreateBlog(ctx, store.BlogEntitty{Title: "second-post", Content: "## Second", Summary: "second summary", Category: "news", Status: "public", PublishedAt: "2026-04-20 00:00:00"})
	if err != nil {
		t.Fatalf("CreateBlog 2: %v", err)
	}
	if _, err := s.CreateBlog(ctx, store.BlogEntitty{Title: "about", Content: "## About", Summary: "about summary", Category: "", Status: "public", PublishedAt: "2026-04-18 12:00:00"}); err != nil {
		t.Fatalf("CreateBlog about: %v", err)
	}
	if _, err := s.UpdateSiteSettings(ctx, store.SiteEntitty{
		SiteTitle:       "micro-front",
		SiteSubtitle:    "subtitle",
		SiteDescription: "description",
		SiteURL:         "https://example.com",
		Tabs: []store.Tab{
			{TabLabel: "Home", TabURL: "/"},
			{TabLabel: "Blogs", TabURL: "/blogs"},
			{TabLabel: "About", TabURL: "/about"},
		},
		FootInformation: "foot",
		Copyright:       "copyright",
	}); err != nil {
		t.Fatalf("UpdateSiteSettings: %v", err)
	}

	uc := Usecase{Store: s, PublishDir: filepath.Join(dataDir, "public")}
	if _, _, err := uc.Run(ctx, Request{Target: "blogs"}); err != nil {
		t.Fatalf("Run blogs: %v", err)
	}

	for _, path := range []string{filepath.Join(uc.PublishDir, "index.html"), filepath.Join(uc.PublishDir, "blogs", "index.html"), filepath.Join(uc.PublishDir, "blogs", "page2.html"), filepath.Join(uc.PublishDir, "blogs", "category", "news", "index.html"), filepath.Join(uc.PublishDir, "blogs", "category", "news", "page2.html"), filepath.Join(uc.PublishDir, "blogs", strconv.FormatInt(blog1.ID, 10)+".html"), filepath.Join(uc.PublishDir, "blogs", strconv.FormatInt(blog2.ID, 10)+".html"), filepath.Join(uc.PublishDir, "assets", "images", strconv.FormatInt(blog1.ID, 10), strconv.FormatInt(img.ID, 10)+".png"), filepath.Join(uc.PublishDir, "robots.txt"), filepath.Join(uc.PublishDir, "sitemap.xml")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", path, err)
		}
	}

	topPage, err := os.ReadFile(filepath.Join(uc.PublishDir, "index.html"))
	if err != nil {
		t.Fatalf("ReadFile top page: %v", err)
	}
	if !strings.Contains(string(topPage), "second-post") {
		t.Fatalf("top page was not regenerated with latest post:\n%s", topPage)
	}
	for _, want := range []string{"href=\"./blogs/index.html\"", "href=\"./about/index.html\"", "href=\"./blogs/2.html\"", "href=\"./blogs/category/news/index.html\"", "category-tree"} {
		if !strings.Contains(string(topPage), want) {
			t.Fatalf("top page missing %q:\n%s", want, topPage)
		}
	}

	blogPage, err := os.ReadFile(filepath.Join(uc.PublishDir, "blogs", strconv.FormatInt(blog1.ID, 10)+".html"))
	if err != nil {
		t.Fatalf("ReadFile blog page: %v", err)
	}
	if !strings.Contains(string(blogPage), "../assets/images/"+strconv.FormatInt(blog1.ID, 10)+"/"+strconv.FormatInt(img.ID, 10)+".png") {
		t.Fatalf("blog page did not rewrite image URL:\n%s", blogPage)
	}
	if strings.Contains(string(blogPage), "/admin/images/") {
		t.Fatalf("blog page still contains admin image URL:\n%s", blogPage)
	}
	if !strings.Contains(string(blogPage), "href=\"../index.html\"") || !strings.Contains(string(blogPage), "href=\"./index.html\"") {
		t.Fatalf("blog page breadcrumb is missing:\n%s", blogPage)
	}
	if !strings.Contains(string(blogPage), "<span>"+blog1.Title+"</span>") {
		t.Fatalf("blog page breadcrumb did not include non-link title:\n%s", blogPage)
	}
	if strings.Contains(string(blogPage), ">"+blog1.Title+"</a>") {
		t.Fatalf("blog page breadcrumb title should not be linked:\n%s", blogPage)
	}
	if !strings.Contains(string(blogPage), formatDateOnly(blog1.PublishedAt)) {
		t.Fatalf("blog page did not show publishedAt:\n%s", blogPage)
	}

	robots, err := os.ReadFile(filepath.Join(uc.PublishDir, "robots.txt"))
	if err != nil {
		t.Fatalf("ReadFile robots: %v", err)
	}
	if !strings.Contains(string(robots), "User-agent: *") || !strings.Contains(string(robots), "Sitemap: https://example.com/sitemap.xml") {
		t.Fatalf("robots.txt missing expected directives:\n%s", robots)
	}

	sitemap, err := os.ReadFile(filepath.Join(uc.PublishDir, "sitemap.xml"))
	if err != nil {
		t.Fatalf("ReadFile sitemap: %v", err)
	}
	for _, want := range []string{
		"<loc>https://example.com/index.html</loc>",
		"<loc>https://example.com/blogs/index.html</loc>",
		"<loc>https://example.com/blogs/" + strconv.FormatInt(blog1.ID, 10) + ".html</loc>",
		"<loc>https://example.com/blogs/category/news/index.html</loc>",
	} {
		if !strings.Contains(string(sitemap), want) {
			t.Fatalf("sitemap.xml missing %q:\n%s", want, sitemap)
		}
	}
}

func TestPreviewURLs_AreRelative(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	s, err := store.New(dataDir)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	blog, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "preview-post",
		Content:     "# Preview",
		Summary:     "preview summary",
		Category:    "news",
		Status:      "public",
		PublishedAt: "2026-04-21 00:00:00",
	})
	if err != nil {
		t.Fatalf("CreateBlog: %v", err)
	}
	if _, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "about",
		Content:     "## About",
		Summary:     "about summary",
		Category:    "profile",
		Status:      "public",
		PublishedAt: "2026-04-20 00:00:00",
	}); err != nil {
		t.Fatalf("CreateBlog about: %v", err)
	}

	uc := Usecase{Store: s, PublishDir: filepath.Join(dataDir, "public")}

	blogPreview, _, err := uc.PreviewBlog(ctx, blog.ID, filepath.Join(dataDir, "preview"))
	if err != nil {
		t.Fatalf("PreviewBlog: %v", err)
	}
	if strings.HasPrefix(blogPreview.URL, "/") {
		t.Fatalf("PreviewBlog returned absolute path: %q", blogPreview.URL)
	}
	if !strings.HasPrefix(blogPreview.URL, "admin/preview/") {
		t.Fatalf("PreviewBlog returned unexpected path: %q", blogPreview.URL)
	}

	sitePreview, _, err := uc.PreviewIndex(ctx, filepath.Join(dataDir, "preview"))
	if err != nil {
		t.Fatalf("PreviewIndex: %v", err)
	}
	if strings.HasPrefix(sitePreview.URL, "/") {
		t.Fatalf("PreviewIndex returned absolute path: %q", sitePreview.URL)
	}
	if !strings.HasPrefix(sitePreview.URL, "admin/preview/") {
		t.Fatalf("PreviewIndex returned unexpected path: %q", sitePreview.URL)
	}

	aboutPreview, _, err := uc.PreviewAbout(ctx, filepath.Join(dataDir, "preview"))
	if err != nil {
		t.Fatalf("PreviewAbout: %v", err)
	}
	if strings.HasPrefix(aboutPreview.URL, "/") {
		t.Fatalf("PreviewAbout returned absolute path: %q", aboutPreview.URL)
	}
	if !strings.HasPrefix(aboutPreview.URL, "admin/preview/") {
		t.Fatalf("PreviewAbout returned unexpected path: %q", aboutPreview.URL)
	}
}

func formatBlogFileName(id int64) string {
	return strconv.FormatInt(id, 10) + ".html"
}

func writeTestPNG(path string) error {
	const pngData = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO1m7W8AAAAASUVORK5CYII="
	data, err := base64.StdEncoding.DecodeString(pngData)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
