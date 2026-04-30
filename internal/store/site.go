package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (s *Store) GetSiteSettings(ctx context.Context) (SiteEntitty, error) {
	var rawTabs string
	var settings SiteEntitty
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, site_title, site_subtitle, site_description, site_url, tabs, foot_information, copyright, updated_at
		FROM site
		WHERE id = 1
	`)
	if err := row.Scan(&settings.ID, &settings.SiteTitle, &settings.SiteSubtitle, &settings.SiteDescription, &settings.SiteURL, &rawTabs, &settings.FootInformation, &settings.Copyright, &settings.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return SiteEntitty{}, fmt.Errorf("site not found")
		}
		return SiteEntitty{}, err
	}

	tabs, err := unmarshalTabs(rawTabs)
	if err != nil {
		return SiteEntitty{}, err
	}
	settings.Tabs = tabs
	return settings, nil
}

func (s *Store) UpdateSiteSettings(ctx context.Context, settings SiteEntitty) (SiteEntitty, error) {
	rawTabs, err := json.Marshal(settings.Tabs)
	if err != nil {
		return SiteEntitty{}, err
	}
	updatedAt := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err = s.DB.ExecContext(ctx, `
		UPDATE site
		SET site_title = ?, site_subtitle = ?, site_description = ?, site_url = ?, tabs = ?, foot_information = ?, copyright = ?, updated_at = ?
		WHERE id = 1
	`, settings.SiteTitle, settings.SiteSubtitle, settings.SiteDescription, settings.SiteURL, string(rawTabs), settings.FootInformation, settings.Copyright, updatedAt)
	if err != nil {
		return SiteEntitty{}, err
	}
	return s.GetSiteSettings(ctx)
}
