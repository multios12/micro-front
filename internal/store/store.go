package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const defaultSiteTabsJSON = `[{"tab_label":"Home","tab_url":"/"},{"tab_label":"Blogs","tab_url":"/blogs"},{"tab_label":"About","tab_url":"/about"}]`

func New(dataDir string) (*Store, error) {
	if dataDir == "" {
		dataDir = "./data"
	}

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "app.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	s := &Store{DB: db, DataDir: dataDir}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	if s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func (s *Store) initSchema() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS site (
			id INTEGER PRIMARY KEY,
			site_title TEXT NOT NULL,
			site_subtitle TEXT NOT NULL,
			site_description TEXT NOT NULL,
			site_url TEXT NOT NULL DEFAULT '',
			tabs TEXT NOT NULL,
			foot_information TEXT NOT NULL,
			copyright TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS blogs (
			id INTEGER PRIMARY KEY,
			title TEXT NOT NULL UNIQUE,
			content TEXT NOT NULL,
			summary TEXT NOT NULL,
			category TEXT,
			status TEXT NOT NULL DEFAULT 'private' CHECK (status IN ('public', 'private')),
			published_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS images (
			id INTEGER PRIMARY KEY,
			blog_id INTEGER NOT NULL,
			alt_text TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (blog_id) REFERENCES blogs(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_blogs_status_published_at
			ON blogs (status, published_at)`,
		`CREATE INDEX IF NOT EXISTS idx_blogs_category
			ON blogs (category)`,
		`CREATE INDEX IF NOT EXISTS idx_images_blog_id
			ON images (blog_id)`,
	}

	for _, stmt := range schema {
		if _, err := s.DB.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}

	if _, err := s.DB.Exec(`ALTER TABLE site ADD COLUMN site_url TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return fmt.Errorf("migrate site schema: %w", err)
	}

	_, err := s.DB.Exec(`
		INSERT OR IGNORE INTO site (id, site_title, site_subtitle, site_description, site_url, tabs, foot_information, copyright, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`, "micro-front", "静的HTMLで配信する公開サイト", "管理画面で更新された記事を、そのまま静的HTMLとして出力する前提のテンプレートモックです。", "", defaultSiteTabsJSON, "micro-front", "© 2026 micro-front")
	if err != nil {
		return fmt.Errorf("seed site settings: %w", err)
	}

	return nil
}

func marshalTabs(tabs []Tab) (string, error) {
	b, err := json.Marshal(tabs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalTabs(raw string) ([]Tab, error) {
	if raw == "" {
		return []Tab{}, nil
	}
	var tabs []Tab
	if err := json.Unmarshal([]byte(raw), &tabs); err != nil {
		return nil, err
	}
	return tabs, nil
}
