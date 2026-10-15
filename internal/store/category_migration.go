package store

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"
)

//go:embed migrations/001_categories.sql
var categoriesSchema string

// migrateCategories はDDLと既存カテゴリの初期投入を一度だけ、同一トランザクションで行います。
func (s *Store) migrateCategories() error {
	ctx := context.Background()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS store_migrations (
		version TEXT PRIMARY KEY, applied_at TEXT NOT NULL
	)`); err != nil {
		return err
	}
	const version = "001_categories"
	var applied bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM store_migrations WHERE version = ?)`, version).Scan(&applied); err != nil {
		return err
	}
	if applied {
		return tx.Commit()
	}
	if _, err := tx.ExecContext(ctx, categoriesSchema); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT category FROM blogs
		WHERE category IS NOT NULL AND category <> '' ORDER BY category COLLATE BINARY`)
	if err != nil {
		return err
	}
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			return err
		}
		keys = append(keys, key)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	names := make(map[string]bool)
	for i, key := range keys {
		name := strings.TrimSpace(key)
		if name == "" || names[name] {
			return fmt.Errorf("legacy category %q has an empty or duplicate display name; resolve before migration", key)
		}
		names[name] = true
		if _, err := tx.ExecContext(ctx, `INSERT INTO categories
			(name, category_key, sort_order, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			name, key, (i+1)*10, now, now); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO store_migrations (version, applied_at) VALUES (?, ?)`, version, now); err != nil {
		return err
	}
	return tx.Commit()
}
