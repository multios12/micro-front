package seed

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"micro-front/internal/validate"
)

func (uc Usecase) seedCategories(ctx context.Context, seedDir string) error {
	items, err := loadJSON[[]CategorySeed](filepath.Join(seedDir, "categories.json"))
	if errors.Is(err, os.ErrNotExist) {
		// カテゴリ未対応の既存seedプロファイルも読み込めるようにする。
		return nil
	}
	if err != nil {
		return err
	}
	byKey := make(map[string]CategorySeed, len(items))
	for i := range items {
		item := &items[i]
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" || !validate.IsCategory(item.CategoryKey) || item.SortOrder < 0 {
			return fmt.Errorf("invalid seed category %q", item.CategoryKey)
		}
		if item.IsHeading != nil && *item.IsHeading && item.ParentKey != "" {
			return fmt.Errorf("seed heading %q cannot have a parent", item.CategoryKey)
		}
		if _, exists := byKey[item.CategoryKey]; exists {
			return fmt.Errorf("duplicate seed category key %q", item.CategoryKey)
		}
		byKey[item.CategoryKey] = *item
	}
	scopes := make(map[string]bool)
	for _, item := range items {
		heading := item.ParentKey == ""
		if item.IsHeading != nil {
			heading = *item.IsHeading
		}
		scope := fmt.Sprintf("%s/%t/%s", item.ParentKey, heading, item.Name)
		if scopes[scope] {
			return fmt.Errorf("duplicate seed category name %q", item.Name)
		}
		scopes[scope] = true
		if item.ParentKey == "" {
			continue
		}
		parent, exists := byKey[item.ParentKey]
		if !exists || parent.ParentKey != "" || (parent.IsHeading != nil && !*parent.IsHeading) || item.ParentKey == item.CategoryKey {
			return fmt.Errorf("seed category %q must reference a root in categories.json", item.CategoryKey)
		}
	}
	tx, err := uc.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	// JSON内の順番に依存せず、親見出しを先に作成する。
	for _, children := range []bool{false, true} {
		for _, item := range items {
			if (item.ParentKey != "") != children {
				continue
			}
			var parentID *int64
			isHeading := !children
			if item.IsHeading != nil {
				isHeading = *item.IsHeading
			}
			if children {
				var id int64
				if err := tx.QueryRowContext(ctx, `SELECT id FROM categories WHERE name=? AND is_heading=1 AND parent_id IS NULL`, byKey[item.ParentKey].Name).Scan(&id); err != nil {
					return err
				}
				parentID = &id
			}
			// 同じ親・種別・表示名のカテゴリを照合し、既存IDを維持する。
			var count int
			if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM categories WHERE parent_id IS ? AND is_heading=? AND name=?", parentID, isHeading, item.Name).Scan(&count); err != nil {
				return err
			}
			if count > 0 {
				_, err = tx.ExecContext(ctx, "UPDATE categories SET sort_order=?,updated_at=? WHERE parent_id IS ? AND is_heading=? AND name=?", item.SortOrder, now, parentID, isHeading, item.Name)
			} else {
				_, err = tx.ExecContext(ctx, "INSERT INTO categories(parent_id,is_heading,name,sort_order,created_at,updated_at) VALUES (?,?,?,?,?,?)", parentID, isHeading, item.Name, item.SortOrder, now, now)
			}
			if err != nil {
				return fmt.Errorf("seed category %q: %w", item.CategoryKey, err)
			}
		}
	}
	return tx.Commit()
}
