package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

func (s *Store) ListBlogs(ctx context.Context, filter BlogListFilter) (BlogListResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}

	var args []any
	var where []string
	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Category != "" {
		where = append(where, "category = ?")
		args = append(args, filter.Category)
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	row := s.DB.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM blogs %s`, whereSQL), args...)
	if err := row.Scan(&total); err != nil {
		return BlogListResult{}, err
	}

	offset := (filter.Page - 1) * filter.PerPage
	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, filter.PerPage, offset)

	rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, title, content, summary, category, status, published_at, updated_at
		FROM blogs
		%s
		ORDER BY published_at DESC, id DESC
		LIMIT ? OFFSET ?
	`, whereSQL), queryArgs...)
	if err != nil {
		return BlogListResult{}, err
	}
	defer rows.Close()

	items := make([]BlogEntitty, 0)
	for rows.Next() {
		blog, err := scanBlog(rows)
		if err != nil {
			return BlogListResult{}, err
		}
		items = append(items, blog)
	}
	if err := rows.Err(); err != nil {
		return BlogListResult{}, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + filter.PerPage - 1) / filter.PerPage
	}

	return BlogListResult{
		Items:      items,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *Store) ListBlogsAll(ctx context.Context) ([]BlogEntitty, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, title, content, summary, category, status, published_at, updated_at
		FROM blogs
		ORDER BY published_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blogs []BlogEntitty
	for rows.Next() {
		blog, err := scanBlog(rows)
		if err != nil {
			return nil, err
		}
		blogs = append(blogs, blog)
	}
	return blogs, rows.Err()
}

func (s *Store) ListPublicBlogs(ctx context.Context) ([]BlogEntitty, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, title, content, summary, category, status, published_at, updated_at
		FROM blogs
		WHERE status = 'public'
		ORDER BY published_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blogs []BlogEntitty
	for rows.Next() {
		blog, err := scanBlog(rows)
		if err != nil {
			return nil, err
		}
		blogs = append(blogs, blog)
	}
	return blogs, rows.Err()
}

func (s *Store) GetBlog(ctx context.Context, id int64) (BlogEntitty, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, title, content, summary, category, status, published_at, updated_at
		FROM blogs
		WHERE id = ?
	`, id)
	return scanBlogRow(row)
}

func (s *Store) GetBlogByTitle(ctx context.Context, title string) (BlogEntitty, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, title, content, summary, category, status, published_at, updated_at
		FROM blogs
		WHERE title = ?
	`, title)
	return scanBlogRow(row)
}

func (s *Store) CreateBlog(ctx context.Context, blog BlogEntitty) (BlogEntitty, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	result, err := s.DB.ExecContext(ctx, `
		INSERT INTO blogs (title, content, summary, category, status, published_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, blog.Title, blog.Content, blog.Summary, nullString(blog.Category), blog.Status, blog.PublishedAt, now)
	if err != nil {
		return BlogEntitty{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return BlogEntitty{}, err
	}
	return s.GetBlog(ctx, id)
}

func (s *Store) CreateBlogWithID(ctx context.Context, blog BlogEntitty, id int64) (BlogEntitty, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO blogs (id, title, content, summary, category, status, published_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, blog.Title, blog.Content, blog.Summary, nullString(blog.Category), blog.Status, blog.PublishedAt, now); err != nil {
		return BlogEntitty{}, err
	}
	return s.GetBlog(ctx, id)
}

func (s *Store) UpdateBlog(ctx context.Context, id int64, blog BlogEntitty) (BlogEntitty, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := s.DB.ExecContext(ctx, `
		UPDATE blogs
		SET title = ?, content = ?, summary = ?, category = ?, status = ?, published_at = ?, updated_at = ?
		WHERE id = ?
	`, blog.Title, blog.Content, blog.Summary, nullString(blog.Category), blog.Status, blog.PublishedAt, now, id)
	if err != nil {
		return BlogEntitty{}, err
	}
	return s.GetBlog(ctx, id)
}

func (s *Store) DeleteBlog(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM blogs WHERE id = ?`, id)
	return err
}

func scanBlog(rows interface{ Scan(dest ...any) error }) (BlogEntitty, error) {
	var blog BlogEntitty
	var category sql.NullString
	err := rows.Scan(&blog.ID, &blog.Title, &blog.Content, &blog.Summary, &category, &blog.Status, &blog.PublishedAt, &blog.UpdatedAt)
	if err != nil {
		return BlogEntitty{}, err
	}
	blog.Category = category.String
	return blog, nil
}

func scanBlogRow(row *sql.Row) (BlogEntitty, error) {
	var blog BlogEntitty
	var category sql.NullString
	err := row.Scan(&blog.ID, &blog.Title, &blog.Content, &blog.Summary, &category, &blog.Status, &blog.PublishedAt, &blog.UpdatedAt)
	if err != nil {
		return BlogEntitty{}, err
	}
	blog.Category = category.String
	return blog, nil
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
