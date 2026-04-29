package server

import (
	"net/http"
	"reflect"
	"testing"

	"micro-front/internal/config"
)

func TestServerRouteRegistration(t *testing.T) {
	s := New(config.Config{Port: ":1234"})
	s.HandleFunc("GET /a", func(http.ResponseWriter, *http.Request) {})
	s.Handle("POST /b", http.NotFoundHandler())

	got := sortedRoutes(append(append([]string(nil), s.routes...), "GET /healthz"))
	want := []string{"GET /a", "GET /healthz", "POST /b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sortedRoutes() = %v, want %v", got, want)
	}
}
