package store

import (
	"context"
	"time"
)

// migrateBlogParentCategory は既存DBへのカラム追加と親見出しの初期補完を一度だけ行います。
func (s *Store) migrateBlogParentCategory() error {
	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	const version = "002_blog_parent_category"
	var applied bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM store_migrations WHERE version=?)`, version).Scan(&applied); err != nil {
		return err
	}
	if applied {
		return tx.Commit()
	}
	var hasColumn bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pragma_table_info('blogs') WHERE name='parent_category')`).Scan(&hasColumn); err != nil {
		return err
	}
	if !hasColumn {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE blogs ADD COLUMN parent_category TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE blogs SET parent_category = COALESCE((
		SELECT p.category_key FROM categories c JOIN categories p ON p.id=c.parent_id
		WHERE c.category_key=blogs.category
	), '') WHERE parent_category=''`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_blogs_parent_category ON blogs(parent_category)`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO store_migrations(version,applied_at) VALUES (?,?)`, version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}
