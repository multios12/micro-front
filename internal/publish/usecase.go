package publish

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"micro-front/internal/store"
	"micro-front/internal/titleimage"
	"micro-front/pkg/markdown"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var templateFuncs = template.FuncMap{
	"dateOnly": formatDateOnly,
}

var layoutTemplate = mustParseTemplate("layout", "templates/styles.tmpl", "templates/layout.tmpl")
var indexBodyTemplate = mustParseTemplate("index", "templates/styles.tmpl", "templates/index.tmpl")
var listBodyTemplate = mustParseTemplate("blog-list", "templates/list.tmpl")
var aboutBodyTemplate = mustParseTemplate("about", "templates/about.tmpl")
var blogBodyTemplate = mustParseTemplate("blog", "templates/blog.tmpl")

var publicImagePattern = regexp.MustCompile(`src="(?:/admin/images/)?(\d+)/(\d+)\.png"`)

const previewTTL = 24 * time.Hour

func mustParseTemplate(name string, patterns ...string) *template.Template {
	return template.Must(template.New(name).Funcs(templateFuncs).ParseFS(templateFS, patterns...))
}

// run は公開対象に応じて静的HTMLを再生成します。
func (uc Usecase) run(ctx context.Context, req Request) (Response, map[string]string, error) {
	if req.Target != "all" && req.Target != "index" && req.Target != "blogs" && req.Target != "blog" && req.Target != "about" {
		return Response{}, map[string]string{"target": "公開対象が不正です。"}, nil
	}
	if req.Target == "blog" && req.BlogID == 0 {
		return Response{}, map[string]string{"blog_id": "記事IDを入力してください。"}, nil
	}

	log.Printf("[publish] start target=%s blog_id=%d output_dir=%s", req.Target, req.BlogID, uc.PublishDir)
	if err := uc.render(ctx, req); err != nil {
		return Response{}, nil, err
	}
	log.Printf("[publish] done target=%s blog_id=%d output_dir=%s", req.Target, req.BlogID, uc.PublishDir)
	return Response{Result: "success"}, nil, nil
}

// PreviewBlog は保存済み記事のプレビュー用HTMLを一時ディレクトリへ生成します。
func (uc Usecase) PreviewBlog(ctx context.Context, blogID int64, previewRoot string) (PreviewResponse, map[string]string, error) {
	if blogID == 0 {
		return PreviewResponse{}, map[string]string{"blog_id": "記事IDを入力してください。"}, nil
	}
	token := strconv.FormatInt(blogID, 10) + "-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	outputDir := filepath.Join(previewRoot, token)
	if err := cleanupPreviewRoot(previewRoot, previewTTL); err != nil {
		return PreviewResponse{}, nil, err
	}
	previewUsecase := Usecase{Store: uc.Store, PublishDir: outputDir}
	if err := os.RemoveAll(outputDir); err != nil {
		return PreviewResponse{}, nil, err
	}
	if err := previewUsecase.renderBlogDetail(ctx, blogID); err != nil {
		return PreviewResponse{}, nil, err
	}
	return PreviewResponse{Result: "success", URL: filepath.ToSlash(filepath.Join("admin/preview", token, "blogs", strconv.FormatInt(blogID, 10)+".html"))}, nil, nil
}

// PreviewAbout は about 記事のプレビュー用HTMLを一時ディレクトリへ生成します。
func (uc Usecase) PreviewAbout(ctx context.Context, previewRoot string) (PreviewResponse, map[string]string, error) {
	blog, err := uc.Store.GetBlogByTitle(ctx, "about")
	if err != nil {
		return PreviewResponse{}, nil, err
	}
	token := "about-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	outputDir := filepath.Join(previewRoot, token)
	if err := cleanupPreviewRoot(previewRoot, previewTTL); err != nil {
		return PreviewResponse{}, nil, err
	}
	previewUsecase := Usecase{Store: uc.Store, PublishDir: outputDir}
	if err := os.RemoveAll(outputDir); err != nil {
		return PreviewResponse{}, nil, err
	}
	if err := previewUsecase.renderAboutPreview(ctx, blog); err != nil {
		return PreviewResponse{}, nil, err
	}
	return PreviewResponse{Result: "success", URL: filepath.ToSlash(filepath.Join("admin/preview", token, "about", "index.html"))}, nil, nil
}

// PreviewIndex はトップページのプレビュー用HTMLを一時ディレクトリへ生成します。
func (uc Usecase) PreviewIndex(ctx context.Context, previewRoot string) (PreviewResponse, map[string]string, error) {
	token := "index-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	outputDir := filepath.Join(previewRoot, token)
	if err := cleanupPreviewRoot(previewRoot, previewTTL); err != nil {
		return PreviewResponse{}, nil, err
	}
	previewUsecase := Usecase{Store: uc.Store, PublishDir: outputDir}
	if err := os.RemoveAll(outputDir); err != nil {
		return PreviewResponse{}, nil, err
	}
	if err := previewUsecase.renderIndex(ctx); err != nil {
		return PreviewResponse{}, nil, err
	}
	return PreviewResponse{Result: "success", URL: filepath.ToSlash(filepath.Join("admin/preview", token, "index.html"))}, nil, nil
}

func (uc Usecase) render(ctx context.Context, req Request) error {
	if err := os.MkdirAll(uc.PublishDir, 0o755); err != nil {
		return err
	}
	switch req.Target {
	case "all":
		if err := uc.renderBlogs(ctx, 0); err != nil {
			return err
		}
		if err := uc.renderAbout(ctx); err != nil {
			return err
		}
		return uc.renderError()
	case "index":
		return uc.renderIndex(ctx)
	case "blogs":
		return uc.renderBlogs(ctx, req.BlogID)
	case "blog":
		return uc.renderBlog(ctx, req.BlogID)
	case "about":
		return uc.renderAbout(ctx)
	default:
		return fmt.Errorf("unsupported target")
	}
}

func (uc Usecase) renderPublicMetadata(ctx context.Context) error {
	settings, err := uc.Store.GetSiteSettings(ctx)
	if err != nil {
		return err
	}
	siteURL := strings.TrimSpace(settings.SiteURL)
	if err := uc.renderRobotsTxt(siteURL); err != nil {
		return err
	}
	if baseURL := siteURL; baseURL != "" {
		return uc.renderSitemapXML(ctx, baseURL)
	}
	return nil
}

func (uc Usecase) renderIndex(ctx context.Context) error {
	if err := uc.renderIndexWithLimit(ctx, loadPublishLimit("TOP_PAGE_BLOG_LIMIT", 20)); err != nil {
		return err
	}
	if err := uc.writeIndexTitleImageSVGs(ctx); err != nil {
		return err
	}
	return uc.renderPublicMetadata(ctx)
}

func (uc Usecase) writeIndexTitleImageSVGs(ctx context.Context) error {
	blogs, err := uc.Store.ListPublicBlogs(ctx)
	if err != nil {
		return err
	}
	return uc.writeTitleImageSVGs(limit(blogs, loadPublishLimit("TOP_PAGE_BLOG_LIMIT", 20)))
}

func (uc Usecase) renderIndexWithLimit(ctx context.Context, topLimit int) error {
	settings, err := uc.Store.GetSiteSettings(ctx)
	if err != nil {
		return err
	}
	blogs, err := uc.Store.ListPublicBlogs(ctx)
	if err != nil {
		return err
	}
	tabs, err := uc.publicTabs(ctx, settings)
	if err != nil {
		return err
	}
	page, err := renderIndexDocument(IndexPageData{SiteTitle: settings.SiteTitle, SiteSubtitle: settings.SiteSubtitle, SiteDescription: settings.SiteDescription, SiteDescriptionHTML: template.HTML(markdown.ToHTML(settings.SiteDescription)), HomeURL: "./index.html", Tabs: pageTabs("index.html", tabs), LatestPosts: buildIndexBlogCards(limit(blogs, topLimit)), Categories: buildIndexCategories(blogs), FootInformation: settings.FootInformation, Copyright: settings.Copyright})
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(uc.PublishDir, "index.html"), page)
}

func (uc Usecase) renderBlogs(ctx context.Context, blogID int64) error {
	settings, err := uc.Store.GetSiteSettings(ctx)
	if err != nil {
		return err
	}
	blogs, err := uc.Store.ListPublicBlogs(ctx)
	if err != nil {
		return err
	}
	limits := publishLimits()
	if blogID > 0 {
		if err := uc.cleanupBlogListPages(); err != nil {
			return err
		}
	} else {
		if err := os.RemoveAll(filepath.Join(uc.PublishDir, "blogs")); err != nil {
			return err
		}
		if err := os.RemoveAll(filepath.Join(uc.PublishDir, "assets", "images")); err != nil {
			return err
		}
		if err := os.RemoveAll(filepath.Join(uc.PublishDir, "assets", "title-images")); err != nil {
			return err
		}
	}
	if err := uc.renderIndexWithLimit(ctx, limits.topLimit); err != nil {
		return err
	}
	if blogID > 0 {
		if err := uc.copyPublicImagesForBlog(ctx, blogID); err != nil {
			return err
		}
		if err := uc.writeTitleImageSVGs(blogs); err != nil {
			return err
		}
	} else {
		if err := uc.copyPublicImagesForBlogs(ctx, blogs); err != nil {
			return err
		}
		if err := uc.writeTitleImageSVGs(blogs); err != nil {
			return err
		}
	}
	if err := uc.renderBlogListPages(ctx, settings, blogs, limits.blogsPerPage); err != nil {
		return err
	}
	if blogID > 0 {
		targetBlog, err := uc.Store.GetBlog(ctx, blogID)
		if err != nil {
			return err
		}
		if err := uc.renderBlogDetail(ctx, blogID); err != nil {
			return err
		}
		if targetBlog.Category != "" {
			if err := uc.cleanupCategoryPages(targetBlog.Category); err != nil {
				return err
			}
			catBlogs := filterBlogsByCategory(blogs, targetBlog.Category)
			if err := uc.renderCategory(targetBlog.Category, catBlogs, limits.blogsPerPage); err != nil {
				return err
			}
		}
		return uc.renderPublicMetadata(ctx)
	}
	for _, blog := range blogs {
		if err := uc.renderBlogDetail(ctx, blog.ID); err != nil {
			return err
		}
	}
	for category, items := range groupBlogsByCategory(blogs) {
		if err := uc.renderCategory(category, items, limits.blogsPerPage); err != nil {
			return err
		}
	}
	return uc.renderPublicMetadata(ctx)
}

func (uc Usecase) renderBlogListPages(ctx context.Context, settings store.SiteEntitty, blogs []store.BlogEntitty, perPage int) error {
	totalPages := (len(blogs) + perPage - 1) / perPage
	pageFile := "blogs/index.html"
	pageBody, err := renderBlogListDocument(BlogListPageData{Breadcrumbs: buildListBreadcrumbs(pageFile, ""), Kicker: "Blogs", Heading: "記事一覧", Items: buildBlogListCards(pageFile, pageSlice(blogs, 0, perPage)), Pagination: template.HTML(renderPagination(pageFile, "blogs", 1, totalPages))})
	if err != nil {
		return err
	}
	page, err := uc.renderPageAt(ctx, settings, "Blogs", "blogs/index.html", pageBody)
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(uc.PublishDir, "blogs", "index.html"), page); err != nil {
		return err
	}
	for pageNum := 2; (pageNum-1)*perPage < len(blogs); pageNum++ {
		pageFile := filepath.ToSlash(filepath.Join("blogs", "page"+strconv.Itoa(pageNum)+".html"))
		pbody, err := renderBlogListDocument(BlogListPageData{Breadcrumbs: buildListBreadcrumbs(pageFile, ""), Kicker: "Blogs", Heading: "記事一覧", Items: buildBlogListCards(pageFile, pageSlice(blogs, pageNum-1, perPage)), Pagination: template.HTML(renderPagination(pageFile, "blogs", pageNum, totalPages))})
		if err != nil {
			return err
		}
		p, err := uc.renderPageAt(ctx, settings, "Blogs", pageFile, pbody)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(uc.PublishDir, "blogs", "page"+strconv.Itoa(pageNum)+".html"), p); err != nil {
			return err
		}
	}
	return nil
}

func (uc Usecase) renderBlog(ctx context.Context, id int64) error {
	if err := uc.renderBlogDetail(ctx, id); err != nil {
		return err
	}
	return uc.renderBlogRelatedPages(ctx, id)
}

func (uc Usecase) renderBlogDetail(ctx context.Context, id int64) error {
	settings, err := uc.Store.GetSiteSettings(ctx)
	if err != nil {
		return err
	}
	blog, err := uc.Store.GetBlog(ctx, id)
	if err != nil {
		return err
	}
	pageFile := filepath.ToSlash(filepath.Join("blogs", strconv.FormatInt(id, 10)+".html"))
	if blog.Title != "about" {
		if err := uc.writeTitleImageSVG(blog); err != nil {
			return err
		}
	}
	breadcrumbItems := []PageBreadcrumb{{Label: "Home", URL: "../index.html"}, {Label: "Blogs", URL: "./index.html"}}
	if blog.Category != "" {
		breadcrumbItems = append(breadcrumbItems, PageBreadcrumb{Label: blog.Category, URL: "./category/" + categorySlug(blog.Category) + "/index.html"})
	}
	breadcrumbItems = append(breadcrumbItems, PageBreadcrumb{Label: blog.Title})
	body, err := renderBlogDocument(BlogDetailPageData{Breadcrumbs: breadcrumbItems, Title: blog.Title, Meta: template.HTML(blogMetaHTML(pageFile, blog)), PublishedAt: blog.PublishedAt, TitleImageURL: titleImageURL(pageFile, blog.ID), Content: template.HTML(publicMarkdownHTML(pageFile, blog.ID, blog.Content))})
	if err != nil {
		return err
	}
	page, err := uc.renderPageAt(ctx, settings, blog.Title, pageFile, body)
	if err != nil {
		return err
	}
	if err := uc.copyPublicImagesForBlog(ctx, blog.ID); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(uc.PublishDir, "blogs", strconv.FormatInt(id, 10)+".html"), page); err != nil {
		return err
	}
	return nil
}

func (uc Usecase) renderBlogRelatedPages(ctx context.Context, blogID int64) error {
	settings, err := uc.Store.GetSiteSettings(ctx)
	if err != nil {
		return err
	}
	blogs, err := uc.Store.ListPublicBlogs(ctx)
	if err != nil {
		return err
	}
	limits := publishLimits()
	if err := uc.cleanupBlogListPages(); err != nil {
		return err
	}
	if err := uc.renderIndexWithLimit(ctx, limits.topLimit); err != nil {
		return err
	}
	if err := uc.renderBlogListPages(ctx, settings, blogs, limits.blogsPerPage); err != nil {
		return err
	}
	if err := uc.writeTitleImageSVGs(blogs); err != nil {
		return err
	}
	targetBlog, err := uc.Store.GetBlog(ctx, blogID)
	if err != nil {
		return err
	}
	if targetBlog.Category != "" {
		if err := uc.cleanupCategoryPages(targetBlog.Category); err != nil {
			return err
		}
		catBlogs := filterBlogsByCategory(blogs, targetBlog.Category)
		if err := uc.renderCategory(targetBlog.Category, catBlogs, limits.blogsPerPage); err != nil {
			return err
		}
	}
	return uc.renderPublicMetadata(ctx)
}

func (uc Usecase) renderAbout(ctx context.Context) error {
	blog, err := uc.Store.GetBlogByTitle(ctx, "about")
	if err != nil || blog.Status != "public" {
		if err := os.RemoveAll(filepath.Join(uc.PublishDir, "about")); err != nil {
			return err
		}
		return uc.renderPublicMetadata(ctx)
	}
	return uc.renderAboutPage(ctx, blog)
}

func (uc Usecase) renderAboutPreview(ctx context.Context, blog store.BlogEntitty) error {
	return uc.renderAboutPage(ctx, blog)
}

func (uc Usecase) renderAboutPage(ctx context.Context, blog store.BlogEntitty) error {
	settings, err := uc.Store.GetSiteSettings(ctx)
	if err != nil {
		return err
	}
	leadFigure, err := uc.renderLeadImageFigure(ctx, "about/index.html", blog.ID, blog.Title)
	if err != nil {
		return err
	}
	body, err := renderAboutDocument(AboutPageData{
		Breadcrumbs: []PageBreadcrumb{{Label: "Home", URL: "../index.html"}, {Label: "About"}},
		BodyTitle:   blog.Title,
		Content:     template.HTML(publicMarkdownHTML("about/index.html", blog.ID, blog.Content)),
		LeadFigure:  template.HTML(leadFigure),
	})
	if err != nil {
		return err
	}
	page, err := uc.renderPageAt(ctx, settings, "About", "about/index.html", body)
	if err != nil {
		return err
	}
	if err := uc.copyPublicImagesForBlog(ctx, blog.ID); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(uc.PublishDir, "about", "index.html"), page); err != nil {
		return err
	}
	return uc.renderPublicMetadata(ctx)
}

func (uc Usecase) renderCategory(category string, blogs []store.BlogEntitty, perPage int) error {
	sort.SliceStable(blogs, func(i, j int) bool { return blogs[i].PublishedAt > blogs[j].PublishedAt })
	page, err := uc.Store.GetSiteSettings(context.Background())
	if err != nil {
		return err
	}
	slug := categorySlug(category)
	pageFile := filepath.ToSlash(filepath.Join("blogs", "category", slug, "index.html"))
	totalPages := (len(blogs) + perPage - 1) / perPage
	body, err := renderBlogListDocument(BlogListPageData{Breadcrumbs: buildListBreadcrumbs(pageFile, category), Kicker: "Category", Heading: category, Items: buildBlogListCards(pageFile, pageSlice(blogs, 0, perPage)), Pagination: template.HTML(renderPagination(pageFile, filepath.ToSlash(filepath.Join("blogs", "category", slug)), 1, totalPages))})
	if err != nil {
		return err
	}
	html, err := uc.renderPageAt(context.Background(), page, esc(category), pageFile, body)
	if err != nil {
		return err
	}
	if err := writeFile(filepath.Join(uc.PublishDir, "blogs", "category", slug, "index.html"), html); err != nil {
		return err
	}
	for pageNum := 2; (pageNum-1)*perPage < len(blogs); pageNum++ {
		pageFile := filepath.ToSlash(filepath.Join("blogs", "category", slug, "page"+strconv.Itoa(pageNum)+".html"))
		body, err := renderBlogListDocument(BlogListPageData{Breadcrumbs: buildListBreadcrumbs(pageFile, category), Kicker: "Category", Heading: category, Items: buildBlogListCards(pageFile, pageSlice(blogs, pageNum-1, perPage)), Pagination: template.HTML(renderPagination(pageFile, filepath.ToSlash(filepath.Join("blogs", "category", slug)), pageNum, totalPages))})
		if err != nil {
			return err
		}
		pageHTML, err := uc.renderPageAt(context.Background(), page, esc(category), pageFile, body)
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(uc.PublishDir, "blogs", "category", slug, "page"+strconv.Itoa(pageNum)+".html"), pageHTML); err != nil {
			return err
		}
	}
	return uc.renderPublicMetadata(context.Background())
}

func (uc Usecase) renderError() error {
	settings, err := uc.Store.GetSiteSettings(context.Background())
	if err != nil {
		return err
	}
	page, err := uc.renderPageAt(context.Background(), settings, "Error", "error.html", `<section class="hero">
  <div class="hero-card">
    <p class="kicker">Error</p>
    <h1>404</h1>
    <p>公開ページの生成対象が見つからないときに表示されるエラーページです。</p>
  </div>
</section>`)
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(uc.PublishDir, "error.html"), page)
}

func (uc Usecase) renderRobotsTxt(siteURL string) error {
	content := "User-agent: *\nAllow: /\n"
	if strings.TrimSpace(siteURL) != "" {
		content += "Sitemap: " + strings.TrimRight(strings.TrimSpace(siteURL), "/") + "/sitemap.xml\n"
	}
	return writeFile(filepath.Join(uc.PublishDir, "robots.txt"), content)
}

type sitemapURL struct {
	Loc string `xml:"loc"`
}

type sitemapXML struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func (uc Usecase) renderSitemapXML(ctx context.Context, baseURL string) error {
	blogs, err := uc.Store.ListPublicBlogs(ctx)
	if err != nil {
		return err
	}
	baseURL = strings.TrimRight(baseURL, "/")
	urls := []string{
		baseURL + "/index.html",
		baseURL + "/blogs/index.html",
	}
	if about, err := uc.Store.GetBlogByTitle(ctx, "about"); err == nil && about.Status == "public" {
		urls = append(urls, baseURL+"/about/index.html")
	}
	limits := publishLimits()
	totalPages := (len(blogs) + limits.blogsPerPage - 1) / limits.blogsPerPage
	for page := 2; page <= totalPages; page++ {
		urls = append(urls, baseURL+"/blogs/page"+strconv.Itoa(page)+".html")
	}
	for _, blog := range blogs {
		urls = append(urls, baseURL+"/blogs/"+strconv.FormatInt(blog.ID, 10)+".html")
	}
	for category := range groupBlogsByCategory(blogs) {
		slug := categorySlug(category)
		categoryBlogs := filterBlogsByCategory(blogs, category)
		urls = append(urls, baseURL+"/blogs/category/"+slug+"/index.html")
		totalCategoryPages := (len(categoryBlogs) + limits.blogsPerPage - 1) / limits.blogsPerPage
		for page := 2; page <= totalCategoryPages; page++ {
			urls = append(urls, baseURL+"/blogs/category/"+slug+"/page"+strconv.Itoa(page)+".html")
		}
	}
	doc := sitemapXML{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, loc := range urls {
		doc.URLs = append(doc.URLs, sitemapURL{Loc: loc})
	}
	buf, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(uc.PublishDir, "sitemap.xml"), xml.Header+string(buf)+"\n")
}

func (uc Usecase) renderPageAt(ctx context.Context, settings store.SiteEntitty, title, pageFile, body string) (string, error) {
	tabs, err := uc.publicTabs(ctx, settings)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = layoutTemplate.ExecuteTemplate(&buf, "layout", PageTemplateData{Title: title, SiteTitle: settings.SiteTitle, SiteSubtitle: settings.SiteSubtitle, SiteDescription: settings.SiteDescription, SiteDescriptionHTML: template.HTML(markdown.ToHTML(settings.SiteDescription)), HomeURL: relURL(pageFile, "index.html"), Tabs: pageTabs(pageFile, tabs), Body: template.HTML(body), FootInformation: settings.FootInformation, Copyright: settings.Copyright})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (uc Usecase) publicTabs(ctx context.Context, settings store.SiteEntitty) ([]store.Tab, error) {
	tabs := settings.Tabs
	if len(tabs) == 0 {
		tabs = defaultPublicTabs()
	}
	about, err := uc.Store.GetBlogByTitle(ctx, "about")
	if err == nil && about.Status == "public" {
		return tabs, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	filtered := make([]store.Tab, 0, len(tabs))
	for _, tab := range tabs {
		if !isAboutTabURL(tab.TabURL) {
			filtered = append(filtered, tab)
		}
	}
	return filtered, nil
}

func blogMetaHTML(pageFile string, blog store.BlogEntitty) string {
	parts := []string{}
	if blog.Category != "" {
		parts = append(parts, `<a href="`+relURL(pageFile, filepath.ToSlash(filepath.Join("blogs", "category", categorySlug(blog.Category), "index.html")))+`">`+esc(blog.Category)+`</a>`)
	}
	return strings.Join(parts, ` / `)
}

func renderIndexDocument(data IndexPageData) (string, error) {
	var buf bytes.Buffer
	if err := indexBodyTemplate.ExecuteTemplate(&buf, "index", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
func renderBlogListDocument(data BlogListPageData) (string, error) {
	var buf bytes.Buffer
	if err := listBodyTemplate.ExecuteTemplate(&buf, "blog-list", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
func buildListBreadcrumbs(pageFile, category string) []PageBreadcrumb {
	breadcrumbs := []PageBreadcrumb{{Label: "Home", URL: relURL(pageFile, "index.html")}, {Label: "Blogs", URL: relURL(pageFile, filepath.ToSlash(filepath.Join("blogs", "index.html")))}}
	if category != "" {
		breadcrumbs = append(breadcrumbs, PageBreadcrumb{Label: category, URL: relURL(pageFile, filepath.ToSlash(filepath.Join("blogs", "category", categorySlug(category), "index.html")))})
	}
	return breadcrumbs
}
func renderAboutDocument(data AboutPageData) (string, error) {
	var buf bytes.Buffer
	if err := aboutBodyTemplate.ExecuteTemplate(&buf, "about", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
func renderBlogDocument(data BlogDetailPageData) (string, error) {
	var buf bytes.Buffer
	if err := blogBodyTemplate.ExecuteTemplate(&buf, "blog", data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
func buildBlogListCards(pageFile string, blogs []store.BlogEntitty) []BlogListCard {
	cards := make([]BlogListCard, 0, len(blogs))
	for _, blog := range blogs {
		cards = append(cards, BlogListCard{Title: blog.Title, Summary: blog.Summary, Category: blog.Category, PublishedAt: blog.PublishedAt, URL: relURL(pageFile, filepath.ToSlash(filepath.Join("blogs", strconv.FormatInt(blog.ID, 10)+".html"))), TitleImageURL: titleImageURL(pageFile, blog.ID)})
	}
	return cards
}
func buildIndexBlogCards(blogs []store.BlogEntitty) []IndexPostCard {
	cards := make([]IndexPostCard, 0, len(blogs))
	for _, blog := range blogs {
		cards = append(cards, IndexPostCard{Title: blog.Title, Summary: blog.Summary, Category: blog.Category, PublishedAt: blog.PublishedAt, URL: relURL("index.html", filepath.ToSlash(filepath.Join("blogs", strconv.FormatInt(blog.ID, 10)+".html"))), TitleImageURL: titleImageURL("index.html", blog.ID)})
	}
	return cards
}

func formatDateOnly(value string) string {
	if value == "" {
		return ""
	}
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}
func buildIndexCategories(blogs []store.BlogEntitty) []IndexCategoryGroup {
	groups := groupBlogsByCategory(blogs)
	if len(groups) == 0 {
		return nil
	}
	type categoryNode struct {
		name  string
		count int
		url   string
	}
	byRoot := map[string][]categoryNode{}
	rootOrder := make([]string, 0, len(groups))
	for category, items := range groups {
		root, child := splitIndexCategory(category)
		node := categoryNode{name: category, count: len(items), url: relURL("index.html", filepath.ToSlash(filepath.Join("blogs", "category", categorySlug(category), "index.html")))}
		if child == "" || root == category {
			root = category
		}
		if _, ok := byRoot[root]; !ok {
			rootOrder = append(rootOrder, root)
		}
		byRoot[root] = append(byRoot[root], node)
	}
	sort.Strings(rootOrder)
	out := make([]IndexCategoryGroup, 0, len(rootOrder))
	for _, root := range rootOrder {
		children := byRoot[root]
		sort.Slice(children, func(i, j int) bool {
			if children[i].name == children[j].name {
				return children[i].url < children[j].url
			}
			return children[i].name < children[j].name
		})
		group := IndexCategoryGroup{Name: root}
		if len(children) == 1 && children[0].name == root {
			group.Count = children[0].count
			group.URL = children[0].url
			group.Open = false
		} else {
			group.Open = true
			for _, child := range children {
				group.Children = append(group.Children, IndexCategoryCard{Name: child.name, Count: child.count, URL: child.url})
				group.Count += child.count
			}
		}
		out = append(out, group)
	}
	return out
}
func splitIndexCategory(category string) (string, string) {
	category = strings.TrimSpace(category)
	if category == "" {
		return "", ""
	}
	if strings.Contains(category, "/") {
		parts := strings.Split(category, "/")
		if len(parts) >= 2 {
			root := strings.Join(parts[:len(parts)-1], "/")
			return root, category
		}
	}
	if strings.Count(category, "-") >= 1 {
		parts := strings.Split(category, "-")
		if len(parts) >= 2 {
			root := strings.Join(parts[:len(parts)-1], "-")
			return root, category
		}
	}
	return category, ""
}
func pageTabs(pageFile string, tabs []store.Tab) []PageTab {
	activeTarget := activeTabTarget(pageFile)
	out := make([]PageTab, 0, len(tabs))
	for _, tab := range tabs {
		target := tabURLToFile(tab.TabURL)
		out = append(out, PageTab{TabLabel: tab.TabLabel, TabURL: relURL(pageFile, target), Active: isSameFile(activeTarget, target)})
	}
	return out
}
func defaultPublicTabs() []store.Tab {
	return []store.Tab{{TabLabel: "Home", TabURL: "/"}, {TabLabel: "Blogs", TabURL: "/blogs"}, {TabLabel: "About", TabURL: "/about"}}
}
func activeTabTarget(pageFile string) string {
	switch {
	case pageFile == "index.html":
		return "index.html"
	case strings.HasPrefix(pageFile, "blogs/"):
		return filepath.ToSlash(filepath.Join("blogs", "index.html"))
	case strings.HasPrefix(pageFile, "about/"):
		return filepath.ToSlash(filepath.Join("about", "index.html"))
	default:
		return "index.html"
	}
}
func tabURLToFile(url string) string {
	switch url {
	case "/":
		return "index.html"
	case "/blogs":
		return filepath.ToSlash(filepath.Join("blogs", "index.html"))
	case "/about":
		return filepath.ToSlash(filepath.Join("about", "index.html"))
	}
	if strings.HasPrefix(url, "/blogs/category/") {
		slug := strings.TrimPrefix(url, "/blogs/category/")
		return filepath.ToSlash(filepath.Join("blogs", "category", slug, "index.html"))
	}
	if strings.HasPrefix(url, "/") {
		return strings.TrimPrefix(url, "/")
	}
	return url
}

func isAboutTabURL(url string) bool {
	normalized := strings.TrimSpace(url)
	normalized = strings.TrimSuffix(normalized, "/")
	return normalized == "/about"
}

func relURL(fromFile, targetFile string) string {
	fromDir := path.Dir(fromFile)
	if fromDir == "." {
		fromDir = "."
	}
	rel, err := filepath.Rel(fromDir, targetFile)
	if err != nil {
		return targetFile
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return "./index.html"
	}
	if !strings.HasPrefix(rel, ".") {
		return "./" + rel
	}
	return rel
}

func isSameFile(a, b string) bool { return path.Clean(a) == path.Clean(b) }

func categorySlug(category string) string {
	slug := strings.TrimSpace(category)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	if slug == "" {
		slug = "category"
	}
	return slug
}

func publicMarkdownHTML(pageFile string, blogID int64, content string) string {
	html := markdown.ToHTML(content)
	return publicImagePattern.ReplaceAllStringFunc(html, func(match string) string {
		parts := publicImagePattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		target := filepath.ToSlash(filepath.Join("assets", "images", parts[1], parts[2]+".png"))
		return `src="` + relURL(pageFile, target) + `"`
	})
}

func renderPagination(pageFile, prefix string, currentPage, totalPages int) string {
	if totalPages <= 1 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<nav class="pagination" aria-label="ページネーション">`)
	if currentPage > 1 {
		b.WriteString(`<a class="pagination-link" href="` + relURL(pageFile, pageURL(prefix, currentPage-1)) + `">Prev</a>`)
	}
	for page := 1; page <= totalPages; page++ {
		class := "pagination-link"
		if page == currentPage {
			class += " is-active"
		}
		attrs := ""
		if page == currentPage {
			attrs = ` aria-current="page"`
		}
		b.WriteString(`<a class="` + class + `" href="` + relURL(pageFile, pageURL(prefix, page)) + `"` + attrs + `>` + strconv.Itoa(page) + `</a>`)
	}
	if currentPage < totalPages {
		b.WriteString(`<a class="pagination-link" href="` + relURL(pageFile, pageURL(prefix, currentPage+1)) + `" aria-label="次のページへ">Next</a>`)
	}
	b.WriteString(`</nav>`)
	return b.String()
}

func pageURL(prefix string, page int) string {
	if page <= 1 {
		return filepath.ToSlash(filepath.Join(prefix, "index.html"))
	}
	return filepath.ToSlash(filepath.Join(prefix, "page"+strconv.Itoa(page)+".html"))
}

func (uc Usecase) renderLeadImageFigure(ctx context.Context, pageFile string, blogID int64, fallback string) (string, error) {
	images, err := uc.Store.ListImagesByBlog(ctx, blogID)
	if err != nil {
		return "", err
	}
	if len(images) == 0 {
		return "", nil
	}
	img := images[0]
	alt := img.AltText
	if strings.TrimSpace(alt) == "" {
		alt = fallback
	}
	src := relURL(pageFile, filepath.ToSlash(filepath.Join("assets", "images", strconv.FormatInt(blogID, 10), strconv.FormatInt(img.ID, 10)+".png")))
	return `<figure class="mock-figure"><img class="mock-image" alt="` + esc(alt) + `" src="` + src + `"></figure>`, nil
}

func pageSlice(blogs []store.BlogEntitty, page, perPage int) []store.BlogEntitty {
	start := page * perPage
	if start >= len(blogs) {
		return nil
	}
	end := start + perPage
	if end > len(blogs) {
		end = len(blogs)
	}
	return blogs[start:end]
}

func limit(blogs []store.BlogEntitty, n int) []store.BlogEntitty {
	if len(blogs) < n {
		n = len(blogs)
	}
	return blogs[:n]
}

type publishLimitsConfig struct {
	topLimit     int
	blogsPerPage int
}

func publishLimits() publishLimitsConfig {
	return publishLimitsConfig{
		topLimit:     loadPublishLimit("TOP_PAGE_BLOG_LIMIT", 20),
		blogsPerPage: loadPublishLimit("BLOGS_PER_PAGE", 20),
	}
}

func loadPublishLimit(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func groupBlogsByCategory(blogs []store.BlogEntitty) map[string][]store.BlogEntitty {
	groups := map[string][]store.BlogEntitty{}
	for _, blog := range blogs {
		if blog.Category != "" {
			groups[blog.Category] = append(groups[blog.Category], blog)
		}
	}
	return groups
}

func filterBlogsByCategory(blogs []store.BlogEntitty, category string) []store.BlogEntitty {
	filtered := make([]store.BlogEntitty, 0, len(blogs))
	for _, blog := range blogs {
		if blog.Category == category {
			filtered = append(filtered, blog)
		}
	}
	return filtered
}

func (uc Usecase) copyPublicImagesForBlogs(ctx context.Context, blogs []store.BlogEntitty) error {
	seen := map[int64]struct{}{}
	for _, blog := range blogs {
		if _, ok := seen[blog.ID]; ok {
			continue
		}
		seen[blog.ID] = struct{}{}
		if err := uc.copyPublicImagesForBlog(ctx, blog.ID); err != nil {
			return err
		}
	}
	return nil
}

func (uc Usecase) copyPublicImagesForBlog(ctx context.Context, blogID int64) error {
	images, err := uc.Store.ListImagesByBlog(ctx, blogID)
	if err != nil {
		return err
	}
	for _, img := range images {
		src := filepath.Join(uc.Store.DataDir, "images", strconv.FormatInt(blogID, 10), strconv.FormatInt(img.ID, 10)+".png")
		dst := filepath.Join(uc.PublishDir, "assets", "images", strconv.FormatInt(blogID, 10), strconv.FormatInt(img.ID, 10)+".png")
		if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func (uc Usecase) writeTitleImageSVGs(blogs []store.BlogEntitty) error {
	for _, blog := range blogs {
		if err := uc.writeTitleImageSVG(blog); err != nil {
			return err
		}
	}
	return nil
}

func (uc Usecase) writeTitleImageSVG(blog store.BlogEntitty) error {
	if blog.Title == "about" {
		return nil
	}
	svg, err := titleimage.GenerateSVG(titleimage.GenerateInput{
		Title:    blog.Title,
		Category: blog.Category,
		Template: titleimage.TemplateID(blog.TitleImageTemplate),
	})
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(uc.PublishDir, "assets", "title-images", strconv.FormatInt(blog.ID, 10)+".svg"), svg)
}

func (uc Usecase) cleanupBlogListPages() error {
	blogDir := filepath.Join(uc.PublishDir, "blogs")
	if err := os.Remove(filepath.Join(blogDir, "index.html")); err != nil && !os.IsNotExist(err) {
		return err
	}
	for page := 2; ; page++ {
		p := filepath.Join(blogDir, "page"+strconv.Itoa(page)+".html")
		err := os.Remove(p)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (uc Usecase) cleanupCategoryPages(category string) error {
	return os.RemoveAll(filepath.Join(uc.PublishDir, "blogs", "category", categorySlug(category)))
}

func esc(value string) string { return html.EscapeString(value) }

func titleImageURL(pageFile string, blogID int64) string {
	return relURL(pageFile, filepath.ToSlash(filepath.Join("assets", "title-images", strconv.FormatInt(blogID, 10)+".svg")))
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	log.Printf("[publish] copied %s -> %s", src, dst)
	return nil
}

func cleanupPreviewRoot(previewRoot string, ttl time.Duration) error {
	entries, err := os.ReadDir(previewRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().Add(-ttl)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(previewRoot, entry.Name()))
		}
	}
	return nil
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	log.Printf("[publish] wrote %s", path)
	return nil
}
