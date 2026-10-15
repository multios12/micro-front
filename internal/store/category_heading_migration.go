package store

import (
	"context"
	"time"
)

// migrateCategoryHeading は子がなくても親見出しとして扱う区分を追加します。
func (s *Store) migrateCategoryHeading() error {
	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	const version = "003_category_heading"
	var applied bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM store_migrations WHERE version=?)`, version).Scan(&applied); err != nil {
		return err
	}
	if applied {
		return tx.Commit()
	}
	var hasColumn bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pragma_table_info('categories') WHERE name='is_heading')`).Scan(&hasColumn); err != nil {
		return err
	}
	if !hasColumn {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE categories ADD COLUMN is_heading INTEGER NOT NULL DEFAULT 0
			CHECK(is_heading IN (0,1) AND (is_heading=0 OR parent_id IS NULL))`); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE categories SET is_heading=1 WHERE EXISTS(SELECT 1 FROM categories child WHERE child.parent_id=categories.id)`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO store_migrations(version,applied_at) VALUES (?,?)`, version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}
