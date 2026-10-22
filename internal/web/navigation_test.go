package web

import (
	"micro-front/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNavigationResponse(t *testing.T) {
	for _, tc := range []struct {
		links []config.AdminNavLink
		want  string
	}{
		{nil, "{\"links\":[]}"},
		{[]config.AdminNavLink{{Title: "アクセス集計", URL: "/access-report/"}}, `{"links":[{"title":"アクセス集計","url":"/access-report/"}]}`},
	} {
		rr := httptest.NewRecorder()
		Handler{AdminNavLinks: tc.links}.handlerGetNavigation(rr, httptest.NewRequest(http.MethodGet, "/admin/api/navigation", nil))
		if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != tc.want {
			t.Fatalf("response: %d %s", rr.Code, rr.Body.String())
		}
	}
}
