package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// migrateRemoveCategoryKey は旧データのID対応付け後にキーを削除します。
func (s *Store) migrateRemoveCategoryKey() error {
	ctx := context.Background()
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var applied bool
	const version = "006_remove_category_key"
	if err := conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM store_migrations WHERE version=?)`, version).Scan(&applied); err != nil {
		return err
	}
	if applied {
		return nil
	}
	// SQLiteのテーブル再作成は外部キーを無効にした専用接続で実施する。
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, `PRAGMA foreign_keys=ON`)
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT sql FROM sqlite_master WHERE tbl_name='categories' AND type IN ('index','trigger') AND sql IS NOT NULL AND name<>'categories_keep_key'`)
	if err != nil {
		return err
	}
	var objects []string
	for rows.Next() {
		var sql string
		if err := rows.Scan(&sql); err != nil {
			rows.Close()
			return err
		}
		if !strings.Contains(sql, "category_key") {
			objects = append(objects, sql)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	// 旧キーはUNIQUE制約を持つためテーブルを再作成する。
	schema := `CREATE TABLE categories_without_key (
 id INTEGER PRIMARY KEY,
 parent_id INTEGER DEFAULT NULL REFERENCES categories(id) ON DELETE RESTRICT,
 is_heading INTEGER NOT NULL DEFAULT 0 CHECK(is_heading IN (0,1) AND (is_heading=0 OR parent_id IS NULL)),
 name TEXT NOT NULL CHECK(length(trim(name))>0 AND name=trim(name)),
 sort_order INTEGER NOT NULL DEFAULT 0 CHECK(typeof(sort_order)='integer' AND sort_order>=0),
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL,
 CHECK(parent_id IS NULL OR parent_id<>id)
 )`
	if _, err := tx.ExecContext(ctx, schema); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO categories_without_key (`+categoryColumns+`) SELECT `+categoryColumns+` FROM categories;
 DROP TABLE categories;
 ALTER TABLE categories_without_key RENAME TO categories;`); err != nil {
		return err
	}
	for _, sql := range objects {
		if _, err := tx.ExecContext(ctx, sql); err != nil {
			return err
		}
	}
	checks, err := tx.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	invalid := checks.Next()
	err = checks.Err()
	checks.Close()
	if err != nil {
		return err
	}
	if invalid {
		return fmt.Errorf("category key migration violated foreign keys")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO store_migrations(version,applied_at) VALUES (?,?)`, version, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}
