package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeAdminHTMLPrefersInternalFile(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join("internal", "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("internal", "web", "admin.html"), []byte("from-internal"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("web/static", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("web/static", "index.html"), []byte("from-static"), 0o644); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	serveAdminHTML(rr, req)

	if body := strings.TrimSpace(rr.Body.String()); body != "from-internal" {
		t.Fatalf("body = %q, want %q", body, "from-internal")
	}
}

func TestServeAdminHTMLFallsBackToStaticIndex(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	serveAdminHTML(rr, req)

	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q, want %q", got, "text/html; charset=utf-8")
	}
	if !bytes.Equal(rr.Body.Bytes(), embeddedAdminHTML) {
		t.Fatalf("body does not match embedded fallback HTML")
	}
}
