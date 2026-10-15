package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrCategoryInUse = errors.New("category is used by blogs")

const categoryColumns = `id, parent_id, is_heading, name, sort_order, created_at, updated_at`

// ListCategories は親見出しとその小見出しを表示順で返します。
// 同じ sort_order の場合はID順で安定化します。
func (s *Store) ListCategories(ctx context.Context) ([]CategoryEntity, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT
		c.id, c.parent_id, c.is_heading, c.name, c.sort_order, c.created_at, c.updated_at
		FROM categories c LEFT JOIN categories p ON p.id = c.parent_id
		ORDER BY COALESCE(p.sort_order, c.sort_order), COALESCE(p.id, c.id),
			CASE WHEN c.parent_id IS NULL THEN 0 ELSE 1 END, c.sort_order, c.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]CategoryEntity, 0)
	for rows.Next() {
		item, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) GetCategory(ctx context.Context, id int64) (CategoryEntity, error) {
	return scanCategory(s.DB.QueryRowContext(ctx, `SELECT `+categoryColumns+` FROM categories WHERE id = ?`, id))
}

func (s *Store) CreateCategory(ctx context.Context, category CategoryEntity) (CategoryEntity, error) {
	category.Name = strings.TrimSpace(category.Name)
	if err := validateCategory(category); err != nil {
		return CategoryEntity{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	return scanCategory(s.DB.QueryRowContext(ctx, `INSERT INTO categories
		(parent_id, is_heading, name, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?) RETURNING `+categoryColumns,
		category.ParentID, category.IsHeading, category.Name, category.SortOrder, now, now))
}

// UpdateCategory は表示名・親・表示順を更新します。
func (s *Store) UpdateCategory(ctx context.Context, id int64, category CategoryEntity) (CategoryEntity, error) {
	category.Name = strings.TrimSpace(category.Name)
	if err := validateCategory(category); err != nil {
		return CategoryEntity{}, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return CategoryEntity{}, err
	}
	defer tx.Rollback()
	updated, err := scanCategory(tx.QueryRowContext(ctx, `UPDATE categories
		SET parent_id = ?, name = ?,
			sort_order = ?, updated_at = ? WHERE id = ? RETURNING `+categoryColumns,
		category.ParentID, category.Name,
		category.SortOrder, time.Now().UTC().Format(time.RFC3339Nano), id))
	if err != nil {
		return CategoryEntity{}, err
	}
	if err := tx.Commit(); err != nil {
		return CategoryEntity{}, err
	}
	return updated, nil
}

// DeleteCategory は公開・非公開を問わず記事で利用されているカテゴリを削除しません。
// 記事の確認と削除は単一SQLで行い、判定結果は同一トランザクションで取得します。
func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var deletedID int64
	err = tx.QueryRowContext(ctx, `DELETE FROM categories WHERE id = ?
		AND NOT EXISTS (SELECT 1 FROM blogs WHERE blogs.category_id = categories.id
			OR blogs.parent_category_id = categories.id
			OR (blogs.parent_category_id IS NULL AND blogs.parent_category = categories.name
				AND (categories.is_heading OR EXISTS (
					SELECT 1 FROM categories child WHERE child.parent_id = categories.id)))
			OR (blogs.category_id IS NULL AND blogs.category = categories.name
				AND NOT categories.is_heading
				AND NOT EXISTS (SELECT 1 FROM categories child WHERE child.parent_id = categories.id)
				AND ((categories.parent_id IS NULL AND blogs.parent_category = ''
					AND blogs.parent_category_id IS NULL)
					OR EXISTS (SELECT 1 FROM categories parent WHERE parent.id = categories.parent_id
						AND (blogs.parent_category_id = parent.id
							OR (blogs.parent_category_id IS NULL AND blogs.parent_category = parent.name))))))
		RETURNING id`, id).Scan(&deletedID)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)`, id).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrCategoryInUse
		}
		return sql.ErrNoRows
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func validateCategory(category CategoryEntity) error {
	if category.Name == "" {
		return fmt.Errorf("category name is required")
	}
	if category.SortOrder < 0 {
		return fmt.Errorf("category sort_order must be non-negative")
	}
	return nil
}

func scanCategory(row interface{ Scan(...any) error }) (CategoryEntity, error) {
	var category CategoryEntity
	err := row.Scan(&category.ID, &category.ParentID, &category.IsHeading, &category.Name,
		&category.SortOrder, &category.CreatedAt, &category.UpdatedAt)
	return category, err
}

// MoveCategory は表示上の兄弟内で1つ移動します。同順序のデータも連番で安定化します。
func (s *Store) MoveCategory(ctx context.Context, id int64, direction string) error {
	if direction != "up" && direction != "down" {
		return fmt.Errorf("invalid category move direction")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var parentID *int64
	if err := tx.QueryRowContext(ctx, "SELECT parent_id FROM categories WHERE id=?", id).Scan(&parentID); err != nil {
		return err
	}
	// 最上位は親見出し・通常カテゴリを同じグループとして扱う。
	rows, err := tx.QueryContext(ctx, "SELECT id FROM categories WHERE parent_id IS ? ORDER BY sort_order,id", parentID)
	if err != nil {
		return err
	}
	var ids []int64
	index := -1
	for rows.Next() {
		var siblingID int64
		if err := rows.Scan(&siblingID); err != nil {
			rows.Close()
			return err
		}
		if siblingID == id {
			index = len(ids)
		}
		ids = append(ids, siblingID)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if index < 0 {
		return sql.ErrNoRows
	}
	next := index + 1
	if direction == "up" {
		next = index - 1
	}
	if next < 0 || next >= len(ids) {
		return tx.Commit()
	}
	ids[index], ids[next] = ids[next], ids[index]
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for position, siblingID := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE categories SET sort_order=?,updated_at=?
			WHERE id=? AND sort_order<>?`, position*10, now, siblingID, position*10); err != nil {
			return err
		}
	}
	return tx.Commit()
}
