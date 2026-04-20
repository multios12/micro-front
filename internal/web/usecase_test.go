package web

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
	t.Cleanup(func() {
		_ = s.Close()
	})

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

	uc := Usecase{
		Store:      s,
		PublishDir: filepath.Join(dataDir, "public"),
	}

	if _, _, err := uc.Publish(ctx, WebPublishRequest{Target: "blog", BlogID: blog.ID}); err != nil {
		t.Fatalf("Publish blog: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(uc.PublishDir, "blogs", formatBlogFileName(blog.ID)))
	if err != nil {
		t.Fatalf("ReadFile blog: %v", err)
	}
	body := string(got)
	for _, want := range []string{
		"<h1>Hello</h1>",
		"<strong>bold</strong>",
		"<em>italic</em>",
		"href=\"../index.html\"",
		"href=\"./index.html\"",
		"Article",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("blog page missing %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, blog.UpdatedAt) {
		t.Fatalf("blog page did not show updatedAt near title:\n%s", body)
	}
	if strings.Contains(body, "# Hello") || strings.Contains(body, "**bold**") || strings.Contains(body, "*italic*") {
		t.Fatalf("blog page still contains raw markdown:\n%s", body)
	}

	if _, _, err := uc.Publish(ctx, WebPublishRequest{Target: "about"}); err != nil {
		t.Fatalf("Publish about: %v", err)
	}

	got, err = os.ReadFile(filepath.Join(uc.PublishDir, "about", "index.html"))
	if err != nil {
		t.Fatalf("ReadFile about: %v", err)
	}
	body = string(got)
	for _, want := range []string{
		"<h2>Profile</h2>",
		"<strong>about</strong>",
	} {
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

	ctx := context.Background()
	dataDir := t.TempDir()
	s, err := store.New(dataDir)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	blog1, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "first-post",
		Content:     "# First",
		Summary:     "first summary",
		Category:    "news",
		Status:      "public",
		PublishedAt: "2026-04-19 00:00:00",
	})
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
	if _, err := s.UpdateBlog(ctx, blog1.ID, store.BlogEntitty{
		Title:       blog1.Title,
		Content:     "# First\n\n![photo](/admin/images/" + strconv.FormatInt(blog1.ID, 10) + "/" + strconv.FormatInt(img.ID, 10) + ".png)",
		Summary:     blog1.Summary,
		Category:    blog1.Category,
		Status:      blog1.Status,
		PublishedAt: blog1.PublishedAt,
	}); err != nil {
		t.Fatalf("UpdateBlog 1: %v", err)
	}
	blog1, err = s.GetBlog(ctx, blog1.ID)
	if err != nil {
		t.Fatalf("GetBlog 1: %v", err)
	}

	blog2, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "second-post",
		Content:     "## Second",
		Summary:     "second summary",
		Category:    "news",
		Status:      "public",
		PublishedAt: "2026-04-20 00:00:00",
	})
	if err != nil {
		t.Fatalf("CreateBlog 2: %v", err)
	}

	if _, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "about",
		Content:     "## About",
		Summary:     "about summary",
		Category:    "",
		Status:      "public",
		PublishedAt: "2026-04-18 12:00:00",
	}); err != nil {
		t.Fatalf("CreateBlog about: %v", err)
	}

	uc := Usecase{
		Store:      s,
		PublishDir: filepath.Join(dataDir, "public"),
	}

	if _, _, err := uc.Publish(ctx, WebPublishRequest{Target: "blogs"}); err != nil {
		t.Fatalf("Publish blogs: %v", err)
	}

	for _, path := range []string{
		filepath.Join(uc.PublishDir, "index.html"),
		filepath.Join(uc.PublishDir, "blogs", "index.html"),
		filepath.Join(uc.PublishDir, "blogs", "page2.html"),
		filepath.Join(uc.PublishDir, "blogs", "category", "news", "index.html"),
		filepath.Join(uc.PublishDir, "blogs", "category", "news", "page2.html"),
		filepath.Join(uc.PublishDir, "blogs", strconv.FormatInt(blog1.ID, 10)+".html"),
		filepath.Join(uc.PublishDir, "blogs", strconv.FormatInt(blog2.ID, 10)+".html"),
		filepath.Join(uc.PublishDir, "assets", "images", strconv.FormatInt(blog1.ID, 10), strconv.FormatInt(img.ID, 10)+".png"),
	} {
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
	for _, want := range []string{
		"href=\"./blogs/index.html\"",
		"href=\"./about/index.html\"",
		"href=\"./blogs/2.html\"",
		"href=\"./blogs/category/news/index.html\"",
		"category-tree",
	} {
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
	if !strings.Contains(string(blogPage), blog1.UpdatedAt) {
		t.Fatalf("blog page did not show updatedAt:\n%s", blogPage)
	}
}

func TestPublishBlogs_WithBlogID_PreservesOtherPublishedFiles(t *testing.T) {
	t.Setenv("TOP_PAGE_BLOG_LIMIT", "5")
	t.Setenv("BLOGS_PER_PAGE", "2")

	ctx := context.Background()
	dataDir := t.TempDir()
	s, err := store.New(dataDir)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	blog1, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "first-post",
		Content:     "# First",
		Summary:     "first summary",
		Category:    "news",
		Status:      "public",
		PublishedAt: "2026-04-19 00:00:00",
	})
	if err != nil {
		t.Fatalf("CreateBlog 1: %v", err)
	}

	blog2, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "second-post",
		Content:     "# Second",
		Summary:     "second summary",
		Category:    "tips",
		Status:      "public",
		PublishedAt: "2026-04-18 00:00:00",
	})
	if err != nil {
		t.Fatalf("CreateBlog 2: %v", err)
	}

	blog3, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "third-post",
		Content:     "# Third",
		Summary:     "third summary",
		Category:    "news",
		Status:      "public",
		PublishedAt: "2026-04-17 00:00:00",
	})
	if err != nil {
		t.Fatalf("CreateBlog 3: %v", err)
	}

	uc := Usecase{
		Store:      s,
		PublishDir: filepath.Join(dataDir, "public"),
	}

	if _, _, err := uc.Publish(ctx, WebPublishRequest{Target: "blogs"}); err != nil {
		t.Fatalf("Publish blogs all: %v", err)
	}

	if _, err := s.UpdateBlog(ctx, blog1.ID, store.BlogEntitty{
		Title:       blog1.Title,
		Content:     "# First updated",
		Summary:     blog1.Summary,
		Category:    blog1.Category,
		Status:      blog1.Status,
		PublishedAt: blog1.PublishedAt,
	}); err != nil {
		t.Fatalf("UpdateBlog 1: %v", err)
	}

	if _, _, err := uc.Publish(ctx, WebPublishRequest{Target: "blogs", BlogID: blog1.ID}); err != nil {
		t.Fatalf("Publish blogs partial: %v", err)
	}

	for _, path := range []string{
		filepath.Join(uc.PublishDir, "blogs", strconv.FormatInt(blog2.ID, 10)+".html"),
		filepath.Join(uc.PublishDir, "blogs", strconv.FormatInt(blog3.ID, 10)+".html"),
		filepath.Join(uc.PublishDir, "blogs", "category", "tips", "index.html"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected preserved file %s: %v", path, err)
		}
	}

	updated, err := os.ReadFile(filepath.Join(uc.PublishDir, "blogs", strconv.FormatInt(blog1.ID, 10)+".html"))
	if err != nil {
		t.Fatalf("ReadFile updated blog: %v", err)
	}
	if !strings.Contains(string(updated), "First updated") {
		t.Fatalf("updated blog page did not refresh content:\n%s", updated)
	}
}

func TestPublishAbout_SkipsPrivateAboutAndRemovesAboutTab(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	s, err := store.New(dataDir)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	if _, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "about",
		Content:     "private about body",
		Summary:     "about summary",
		Category:    "",
		Status:      "private",
		PublishedAt: "2026-04-18 00:00:00",
	}); err != nil {
		t.Fatalf("CreateBlog about: %v", err)
	}

	if _, err := s.CreateBlog(ctx, store.BlogEntitty{
		Title:       "sample-post",
		Content:     "# Hello",
		Summary:     "summary",
		Category:    "news",
		Status:      "public",
		PublishedAt: "2026-04-19 00:00:00",
	}); err != nil {
		t.Fatalf("CreateBlog sample: %v", err)
	}

	uc := Usecase{
		Store:      s,
		PublishDir: filepath.Join(dataDir, "public"),
	}

	if _, _, err := uc.Publish(ctx, WebPublishRequest{Target: "all"}); err != nil {
		t.Fatalf("Publish all: %v", err)
	}

	if _, err := os.Stat(filepath.Join(uc.PublishDir, "about", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("private about page should not be published: %v", err)
	}

	indexPage, err := os.ReadFile(filepath.Join(uc.PublishDir, "index.html"))
	if err != nil {
		t.Fatalf("ReadFile index: %v", err)
	}
	if strings.Contains(string(indexPage), "./about/index.html") {
		t.Fatalf("index page should not contain about tab when about is private:\n%s", indexPage)
	}

	errorPage, err := os.ReadFile(filepath.Join(uc.PublishDir, "error.html"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if strings.Contains(string(errorPage), "./about/index.html") {
		t.Fatalf("error page should not contain about tab when about is private:\n%s", errorPage)
	}
}

func formatBlogFileName(id int64) string {
	return strconv.FormatInt(id, 10) + ".html"
}

func writeTestPNG(path string) error {
	const pngBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7ZrNQAAAAASUVORK5CYII="
	data, err := base64.StdEncoding.DecodeString(pngBase64)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
