package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type blogCategoryIDs struct {
	Category *int64
	Parent   *int64
}

var ErrBlogCategoryNotFound = errors.New("selected category is not registered")

// syncBlogCategories は登録済みの選択値をIDに対応付けます。カテゴリの作成は行いません。
func syncBlogCategories(ctx context.Context, tx *sql.Tx, blog *BlogEntitty) (blogCategoryIDs, error) {
	return resolveBlogCategories(ctx, tx, blog, resolveRegisteredCategory)
}

func resolveBlogCategories(ctx context.Context, tx *sql.Tx, blog *BlogEntitty, resolve func(context.Context, *sql.Tx, *int64, bool, string) (CategoryEntity, error)) (blogCategoryIDs, error) {
	var ids blogCategoryIDs
	blog.Category = strings.TrimSpace(blog.Category)
	blog.ParentCategory = strings.TrimSpace(blog.ParentCategory)
	if blog.Title == "about" {
		blog.Category, blog.ParentCategory = "", ""
		return ids, nil
	}

	if blog.ParentCategory != "" {
		parent, err := resolve(ctx, tx, nil, true, blog.ParentCategory)
		if err != nil {
			return ids, err
		}
		blog.ParentCategory = parent.Name
		ids.Parent = &parent.ID
	}
	if blog.Category != "" {
		category, err := resolve(ctx, tx, ids.Parent, false, blog.Category)
		if err != nil {
			return ids, err
		}
		blog.Category = category.Name
		ids.Category = &category.ID
	}
	return ids, nil
}

func findScopedCategory(ctx context.Context, tx *sql.Tx, parentID *int64, heading bool, name string) (CategoryEntity, error) {
	return scanCategory(tx.QueryRowContext(ctx, `SELECT `+categoryColumns+` FROM categories c
		WHERE parent_id IS ? AND name=? AND
		(c.is_heading OR EXISTS(SELECT 1 FROM categories child WHERE child.parent_id=c.id))=?`, parentID, name, heading))
}

func resolveRegisteredCategory(ctx context.Context, tx *sql.Tx, parentID *int64, heading bool, name string) (CategoryEntity, error) {
	item, err := findScopedCategory(ctx, tx, parentID, heading, name)
	if errors.Is(err, sql.ErrNoRows) {
		return CategoryEntity{Name: name}, fmt.Errorf("%w: %s", ErrBlogCategoryNotFound, name)
	}
	return item, err
}
