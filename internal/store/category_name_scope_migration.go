package store

import (
	"context"
	"time"
)

// migrateCategoryNameScope は親見出しと親なしカテゴリで名前の一意性を分離します。
func (s *Store) migrateCategoryNameScope() error {
	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	const version = "004_category_name_scope"
	var applied bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM store_migrations WHERE version=?)`, version).Scan(&applied); err != nil {
		return err
	}
	if applied {
		return tx.Commit()
	}
	if _, err := tx.ExecContext(ctx, `DROP INDEX idx_categories_root_name;
		CREATE UNIQUE INDEX idx_categories_root_name ON categories(is_heading,name) WHERE parent_id IS NULL`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO store_migrations(version,applied_at) VALUES (?,?)`, version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}
