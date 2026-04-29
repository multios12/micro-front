package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"micro-front/internal/config"
)

func New(cfg config.Config) Server {
	return Server{
		cfg: cfg,
		mux: http.NewServeMux(),
	}
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.routes = append(s.routes, pattern)
	s.mux.Handle(pattern, handler)
}

func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.routes = append(s.routes, pattern)
	s.mux.HandleFunc(pattern, handler)
}

func (s Server) Run(ctx context.Context) error {
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}

	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	s.routes = append(s.routes, "GET /healthz")

	log.Printf("[server] start port=%s", s.cfg.Port)
	log.Printf("[server] routes:\n%s", strings.Join(sortedRoutes(s.routes), "\n"))

	srv := &http.Server{
		Addr:    s.cfg.Port,
		Handler: accessLogMiddleware(s.mux),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		_ = srv.Shutdown(context.Background())
		return nil
	case err := <-errCh:
		if err == nil {
			return nil
		}
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("run server: %w", err)
	}
}

type accessLogResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *accessLogResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *accessLogResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &accessLogResponseWriter{ResponseWriter: w}

		next.ServeHTTP(rw, r)

		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}

		remoteAddr := r.RemoteAddr
		if remoteAddr == "" {
			remoteAddr = "-"
		}
		userAgent := r.UserAgent()
		if userAgent == "" {
			userAgent = "-"
		}

		log.Printf("[access] remote=%s method=%s path=%s status=%d bytes=%d duration=%s ua=%q", remoteAddr, r.Method, r.URL.RequestURI(), status, rw.bytes, time.Since(start).Truncate(time.Millisecond), userAgent)
	})
}

func sortedRoutes(routes []string) []string {
	out := append([]string(nil), routes...)
	slices.Sort(out)
	return out
}
