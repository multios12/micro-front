package blogs

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"micro-front/internal/store"
	"micro-front/internal/validate"
)

var errBlogNotFound = errors.New("blog not found")
var errBlogTitleConflict = errors.New("同じタイトルの記事がすでに存在します")

const aboutBlogID int64 = 9999999

// List は設計書 3.3 の記事一覧取得処理を行います。
func (uc Usecase) List(ctx context.Context, page, perPage int, status string) (BlogsListResponse, string, map[string]string, error) {
	if status != "" && status != "public" && status != "private" {
		return BlogsListResponse{}, "INVALID_STATUS", map[string]string{
			"status": "公開状態は public か private を指定してください。",
		}, nil
	}

	result, err := uc.Store.ListBlogs(ctx, store.BlogListFilter{
		Page:    page,
		PerPage: perPage,
		Status:  status,
	})
	if err != nil {
		return BlogsListResponse{}, "", nil, err
	}

	items := make([]BlogsListItemResponse, 0, len(result.Items))
	for _, blog := range result.Items {
		items = append(items, BlogsListItemResponse{
			ID:          blog.ID,
			Title:       blog.Title,
			Summary:     blog.Summary,
			Category:    blog.Category,
			Status:      blog.Status,
			PublishedAt: blog.PublishedAt,
			UpdatedAt:   blog.UpdatedAt,
		})
	}

	return BlogsListResponse{
		Items:      items,
		Page:       result.Page,
		PerPage:    result.PerPage,
		Total:      result.Total,
		TotalPages: result.TotalPages,
	}, "", nil, nil
}

// Get は設計書 3.4 の記事詳細取得処理を行います。
func (uc Usecase) Get(ctx context.Context, id int64) (BlogsDetailResponse, error) {
	blog, err := uc.getBlogForAdmin(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BlogsDetailResponse{}, errBlogNotFound
		}
		return BlogsDetailResponse{}, err
	}
	return BlogsDetailResponse{
		ID:          blog.ID,
		Title:       blog.Title,
		Content:     blog.Content,
		Summary:     blog.Summary,
		Category:    blog.Category,
		Status:      blog.Status,
		PublishedAt: blog.PublishedAt,
		UpdatedAt:   blog.UpdatedAt,
	}, nil
}

// Create は設計書 3.5 の記事新規作成処理を行います。
func (uc Usecase) Create(ctx context.Context, req BlogsCreateRequest) (BlogsCreateResponse, string, map[string]string, error) {
	blog, code, fields := buildBlog(req.Title, req.Content, req.Category, req.Status, req.PublishedAt, true)
	if len(fields) > 0 {
		return BlogsCreateResponse{}, code, fields, nil
	}

	if _, err := uc.Store.GetBlogByTitle(ctx, blog.Title); err == nil {
		return BlogsCreateResponse{}, "", nil, errBlogTitleConflict
	} else if !errors.Is(err, sql.ErrNoRows) {
		return BlogsCreateResponse{}, "", nil, err
	}

	created, err := uc.createBlogRecord(ctx, blog)
	if err != nil {
		return BlogsCreateResponse{}, "", nil, err
	}
	return BlogsCreateResponse{
		ID:          created.ID,
		Title:       created.Title,
		Content:     created.Content,
		Summary:     created.Summary,
		Category:    created.Category,
		Status:      created.Status,
		PublishedAt: created.PublishedAt,
		UpdatedAt:   created.UpdatedAt,
	}, "", nil, nil
}

// Update は設計書 3.6 の記事更新処理を行います。
func (uc Usecase) Update(ctx context.Context, id int64, req BlogsUpdateRequest) (BlogsUpdateResponse, string, map[string]string, error) {
	if id == aboutBlogID {
		req.Title = "about"
		req.Category = ""
	}

	blog, code, fields := buildBlog(req.Title, req.Content, req.Category, req.Status, req.PublishedAt, id != aboutBlogID)
	if len(fields) > 0 {
		return BlogsUpdateResponse{}, code, fields, nil
	}

	existing, err := uc.getBlogForAdmin(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && id == aboutBlogID {
			created, err := uc.createBlogRecord(ctx, blog)
			if err != nil {
				if errors.Is(err, errBlogTitleConflict) {
					return BlogsUpdateResponse{}, "", nil, errBlogTitleConflict
				}
				return BlogsUpdateResponse{}, "", nil, err
			}
			return BlogsUpdateResponse{
				ID:          created.ID,
				Title:       created.Title,
				Content:     created.Content,
				Summary:     created.Summary,
				Category:    created.Category,
				Status:      created.Status,
				PublishedAt: created.PublishedAt,
				UpdatedAt:   created.UpdatedAt,
			}, "", nil, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			return BlogsUpdateResponse{}, "", nil, errBlogNotFound
		}
		return BlogsUpdateResponse{}, "", nil, err
	}

	duplicate, err := uc.Store.GetBlogByTitle(ctx, blog.Title)
	if err == nil && duplicate.ID != existing.ID {
		return BlogsUpdateResponse{}, "", nil, errBlogTitleConflict
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return BlogsUpdateResponse{}, "", nil, err
	}

	updated, err := uc.Store.UpdateBlog(ctx, id, blog)
	if err != nil {
		return BlogsUpdateResponse{}, "", nil, err
	}
	return BlogsUpdateResponse{
		ID:          updated.ID,
		Title:       updated.Title,
		Content:     updated.Content,
		Summary:     updated.Summary,
		Category:    updated.Category,
		Status:      updated.Status,
		PublishedAt: updated.PublishedAt,
		UpdatedAt:   updated.UpdatedAt,
	}, "", nil, nil
}

// Delete は設計書 3.7 の記事削除処理を行います。
func (uc Usecase) Delete(ctx context.Context, id int64) (BlogsDeleteResponse, error) {
	blog, err := uc.getBlogForAdmin(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BlogsDeleteResponse{}, errBlogNotFound
		}
		return BlogsDeleteResponse{}, err
	}

	images, _ := uc.Store.ListImagesByBlog(ctx, blog.ID)
	if err := uc.Store.DeleteBlog(ctx, blog.ID); err != nil {
		return BlogsDeleteResponse{}, err
	}

	for _, img := range images {
		_ = os.Remove(filepath.Join(uc.DataDir, "images", strconv.FormatInt(blog.ID, 10), strconv.FormatInt(img.ID, 10)+".png"))
	}

	return BlogsDeleteResponse{
		ID:     blog.ID,
		Result: "deleted",
	}, nil
}

func (uc Usecase) getBlogForAdmin(ctx context.Context, id int64) (store.BlogEntitty, error) {
	if id == aboutBlogID {
		return uc.Store.GetBlogByTitle(ctx, "about")
	}
	return uc.Store.GetBlog(ctx, id)
}

func (uc Usecase) createBlogRecord(ctx context.Context, blog store.BlogEntitty) (store.BlogEntitty, error) {
	if blog.Title == "about" {
		if existing, err := uc.Store.GetBlogByTitle(ctx, "about"); err == nil {
			return existing, errBlogTitleConflict
		} else if !errors.Is(err, sql.ErrNoRows) {
			return store.BlogEntitty{}, err
		}
		return uc.Store.CreateBlogWithID(ctx, blog, aboutBlogID)
	}
	return uc.Store.CreateBlog(ctx, blog)
}

// buildBlog は記事作成・更新で共通の入力検証とDBモデル変換を行います。
func buildBlog(title, content, category, status, publishedAt string, validatePublishedAt bool) (store.BlogEntitty, string, map[string]string) {
	fields := map[string]string{}
	code := "VALIDATION_ERROR"
	if validate.Length(title) == 0 {
		fields["title"] = "タイトルを入力してください。"
	} else if validate.Length(title) > 100 {
		fields["title"] = "タイトルは100文字以内で入力してください。"
	}
	if validate.Length(content) == 0 {
		fields["content"] = "本文を入力してください。"
	} else if validate.Length(content) > 20000 {
		fields["content"] = "本文は20000文字以内で入力してください。"
	}
	if validate.Length(category) > 20 {
		fields["category"] = "カテゴリは20文字以内で入力してください。"
	}
	if strings.TrimSpace(category) != "" && !validate.IsCategory(category) {
		fields["category"] = "カテゴリの形式が不正です。"
	}
	if status == "" {
		status = "private"
	}
	if status != "public" && status != "private" {
		fields["status"] = "公開状態は public か private を指定してください。"
		code = "INVALID_STATUS"
	}
	if validatePublishedAt {
		if validate.Length(publishedAt) == 0 {
			fields["published_at"] = "更新日を入力してください。"
		} else if !validate.IsDateTime(publishedAt) {
			fields["published_at"] = "更新日の形式が不正です(yyyy-mm-dd hh:mm:ss)。"
			if code == "VALIDATION_ERROR" {
				code = "INVALID_PUBLISHED_AT"
			}
		}
	}

	if len(fields) > 0 {
		return store.BlogEntitty{}, code, fields
	}

	return store.BlogEntitty{
		Title:       title,
		Content:     content,
		Summary:     validate.SummaryFromContent(content),
		Category:    strings.TrimSpace(category),
		Status:      status,
		PublishedAt: publishedAt,
	}, "", nil
}
