package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvironmentConfiguration(t *testing.T) {
	f := setup(t)
	second := filepath.Join(f.dir, "second log.json")
	write(t, f.log, event("2026-10-01", "/blogs/1.html", "192.0.2.1", "Browser", 200))
	write(t, second, event("2026-10-02", "/blogs/2.html", "192.0.2.2", "Browser", 200))
	for name, value := range map[string]string{
		"INPUT":   f.log + "\r\n" + second + "\n",
		"LOG_DIR": "", "LOG_NAME": "access.log.json", "HOST": "example.com",
		"STATE": f.state, "OUTPUT": f.output, "CRAWLER_RULES": f.rules, "BLOG_DB": "",
	} {
		t.Setenv("ACCESS_REPORT_"+name, value)
	}
	if err := run(context.Background(), nil, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 2 {
		t.Fatal(n)
	}
	if _, err := os.Stat(f.output); err != nil {
		t.Fatal(err)
	}
}

func TestCommandLineOverridesEnvironment(t *testing.T) {
	f := setup(t)
	write(t, f.log, event("2026-10-01", "/blogs/1.html", "192.0.2.1", "Browser", 200))
	for _, name := range []string{"INPUT", "HOST", "STATE", "OUTPUT", "CRAWLER_RULES", "BLOG_DB"} {
		t.Setenv("ACCESS_REPORT_"+name, "invalid")
	}
	t.Setenv("ACCESS_REPORT_LOG_DIR", "")
	args := append(f.args(), "--blog-db", "")
	if err := run(context.Background(), args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, f, "SELECT SUM(access_count) FROM article_daily"); n != 1 {
		t.Fatal(n)
	}
}

func TestEnvironmentBlogDBIsExplicit(t *testing.T) {
	f := setup(t)
	write(t, f.log, event("2026-10-01", "/blogs/1.html", "192.0.2.1", "Browser", 200))
	t.Setenv("ACCESS_REPORT_LOG_DIR", "")
	missing := filepath.Join(f.dir, "missing.db")
	t.Setenv("ACCESS_REPORT_BLOG_DB", missing)
	if err := run(context.Background(), f.args(), &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "記事DB") {
		t.Fatalf("expected missing article DB error, got %v", err)
	}
}
