package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Titles are presentation data: refreshing them never changes the log cursor or
// aggregate schema. Do not use micro-front's store initializer, which migrates DBs.
func loadArticleTitles(ctx context.Context, path string, required bool) (map[int64]string, error) {
	titles := make(map[int64]string)
	if path == "" {
		return titles, nil
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) && !required {
			return titles, nil
		}
		return nil, fmt.Errorf("記事DB: %w", err)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	uriPath := filepath.ToSlash(abs)
	if !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	uri := url.URL{Scheme: "file", Path: uriPath, RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, "SELECT id, title FROM blogs WHERE title <> 'about'")
	if err != nil {
		return nil, fmt.Errorf("記事タイトル取得: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, err
		}
		titles[id] = title
	}
	return titles, rows.Err()
}
