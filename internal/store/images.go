package store

import (
	"context"
	"database/sql"
	"time"
)

func (s *Store) ListImagesByBlog(ctx context.Context, blogID int64) ([]ImageEntitty, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, blog_id, alt_text, created_at, updated_at
		FROM images
		WHERE blog_id = ?
		ORDER BY id ASC
	`, blogID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []ImageEntitty
	for rows.Next() {
		var img ImageEntitty
		var alt sql.NullString
		if err := rows.Scan(&img.ID, &img.BlogID, &alt, &img.CreatedAt, &img.UpdatedAt); err != nil {
			return nil, err
		}
		img.AltText = alt.String
		images = append(images, img)
	}
	return images, rows.Err()
}

func (s *Store) GetImage(ctx context.Context, blogID, id int64) (ImageEntitty, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, blog_id, alt_text, created_at, updated_at
		FROM images
		WHERE blog_id = ? AND id = ?
	`, blogID, id)
	var img ImageEntitty
	var alt sql.NullString
	if err := row.Scan(&img.ID, &img.BlogID, &alt, &img.CreatedAt, &img.UpdatedAt); err != nil {
		return ImageEntitty{}, err
	}
	img.AltText = alt.String
	return img, nil
}

func (s *Store) CreateImage(ctx context.Context, blogID int64, altText string) (ImageEntitty, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	result, err := s.DB.ExecContext(ctx, `
		INSERT INTO images (blog_id, alt_text, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, blogID, nullString(altText), now, now)
	if err != nil {
		return ImageEntitty{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return ImageEntitty{}, err
	}
	return s.GetImage(ctx, blogID, id)
}

func (s *Store) CreateImageWithID(ctx context.Context, blogID, id int64, altText string) (ImageEntitty, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if _, err := s.DB.ExecContext(ctx, `
		INSERT INTO images (id, blog_id, alt_text, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, blogID, nullString(altText), now, now); err != nil {
		return ImageEntitty{}, err
	}
	return s.GetImage(ctx, blogID, id)
}

func (s *Store) UpdateImageTimestamp(ctx context.Context, blogID, id int64) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE images
		SET updated_at = ?
		WHERE blog_id = ? AND id = ?
	`, time.Now().UTC().Format("2006-01-02 15:04:05"), blogID, id)
	return err
}

func (s *Store) DeleteImage(ctx context.Context, blogID, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM images WHERE blog_id = ? AND id = ?`, blogID, id)
	return err
}
