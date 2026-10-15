package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
)

func (s *Store) migrateBlogCategoryLinks() error {
	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	const version = "005_blog_category_links"
	var applied bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM store_migrations WHERE version=?)`, version).Scan(&applied); err != nil {
		return err
	}
	if applied {
		return tx.Commit()
	}
	for _, column := range []string{"category_id", "parent_category_id"} {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pragma_table_info('blogs') WHERE name=?)`, column).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			if _, err := tx.ExecContext(ctx, `ALTER TABLE blogs ADD COLUMN `+column+` INTEGER REFERENCES categories(id) ON DELETE RESTRICT`); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_blogs_`+column+` ON blogs(`+column+`)`); err != nil {
			return err
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,title,category,parent_category FROM blogs`)
	if err != nil {
		return err
	}
	var items []BlogEntitty
	for rows.Next() {
		var item BlogEntitty
		var category sql.NullString
		if err := rows.Scan(&item.ID, &item.Title, &category, &item.ParentCategory); err != nil {
			rows.Close()
			return err
		}
		item.Category = category.String
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := inferLegacyBlogParent(ctx, tx, &item); err != nil {
			return err
		}
		ids, err := resolveBlogCategories(ctx, tx, &item, registerLegacyBlogCategory)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE blogs SET category=?,parent_category=?,category_id=?,parent_category_id=? WHERE id=?`, nullString(item.Category), item.ParentCategory, ids.Category, ids.Parent, item.ID); err != nil {
			return err
		}
	}
	// seedを含め、カテゴリ名を更新するすべての経路で記事側も同期する。
	if _, err := tx.ExecContext(ctx, `CREATE TRIGGER categories_sync_blog_names
		AFTER UPDATE OF name ON categories WHEN NEW.name<>OLD.name
		BEGIN
			UPDATE blogs SET
			category=CASE WHEN category_id=NEW.id THEN NEW.name ELSE category END,
			parent_category=CASE WHEN parent_category_id=NEW.id THEN NEW.name ELSE parent_category END,
			updated_at=strftime('%Y-%m-%d %H:%M:%S','now')
			WHERE category_id=NEW.id OR parent_category_id=NEW.id;
		END`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO store_migrations(version,applied_at) VALUES (?,?)`, version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

// registerLegacyBlogCategory は既存DBの初回移行専用です。通常の記事保存からは呼び出しません。
func registerLegacyBlogCategory(ctx context.Context, tx *sql.Tx, parentID *int64, heading bool, name string) (CategoryEntity, error) {
	item, err := resolveLegacyCategory(ctx, tx, parentID, heading, name)
	if err == nil {
		if heading && !item.IsHeading {
			_, err = tx.ExecContext(ctx, `UPDATE categories SET is_heading=1,updated_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339Nano), item.ID)
			item.IsHeading = true
		}
		return item, err
	}
	if !errors.Is(err, ErrBlogCategoryNotFound) {
		return CategoryEntity{}, err
	}
	name = item.Name
	var order int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sort_order),0)+10 FROM categories WHERE parent_id IS ? AND is_heading=?`, parentID, heading).Scan(&order); err != nil {
		return CategoryEntity{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	return scanCategory(tx.QueryRowContext(ctx, `INSERT INTO categories(parent_id,is_heading,name,category_key,sort_order,created_at,updated_at)
 VALUES (?,?,?,?,?,?,?) RETURNING `+categoryColumns, parentID, heading, name, "category-"+uuid.NewString(), order, now, now))
}

// inferLegacyBlogParent は既存DBの移行時だけ旧カテゴリキーから親を補完します。
func inferLegacyBlogParent(ctx context.Context, tx *sql.Tx, blog *BlogEntitty) error {
	// 旧データの子カテゴリキーのみが渡された場合は、その親も補完する。
	if blog.ParentCategory == "" && blog.Category != "" {
		_, err := findScopedCategory(ctx, tx, nil, false, blog.Category)
		if errors.Is(err, sql.ErrNoRows) {
			legacy, keyErr := scanCategory(tx.QueryRowContext(ctx, `SELECT `+categoryColumns+` FROM categories WHERE category_key=?`, blog.Category))
			if keyErr == nil {
				blog.Category = legacy.Name
				if legacy.ParentID != nil {
					parent, err := scanCategory(tx.QueryRowContext(ctx, `SELECT `+categoryColumns+` FROM categories WHERE id=?`, *legacy.ParentID))
					if err != nil {
						return err
					}
					blog.ParentCategory = parent.Name
				}
			} else if !errors.Is(keyErr, sql.ErrNoRows) {
				return keyErr
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

// resolveLegacyCategory はカラム削除前の移行専用です。
func resolveLegacyCategory(ctx context.Context, tx *sql.Tx, parentID *int64, heading bool, name string) (CategoryEntity, error) {
	item, err := findScopedCategory(ctx, tx, parentID, heading, name)
	if errors.Is(err, sql.ErrNoRows) {
		// 旧カテゴリキーは表示名へ変換し、現在指定された親の中で再検索する。
		legacy, keyErr := scanCategory(tx.QueryRowContext(ctx, `SELECT `+categoryColumns+` FROM categories WHERE category_key=?`, name))
		if keyErr == nil {
			name = legacy.Name
			item, err = findScopedCategory(ctx, tx, parentID, heading, name)
		} else if !errors.Is(keyErr, sql.ErrNoRows) {
			return CategoryEntity{}, keyErr
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return CategoryEntity{Name: name}, fmt.Errorf("%w: %s", ErrBlogCategoryNotFound, name)
	}
	return item, err
}
