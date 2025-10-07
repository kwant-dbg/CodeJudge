package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"codejudge/common/env"
	"codejudge/common/health"

	"go.uber.org/zap"
)

var logger *zap.Logger

func newProxy(targetHost string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}
	return httputil.NewSingleHostReverseProxy(url), nil
}

func main() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	problemsURL := env.Get("PROBLEMS_SERVICE_URL", "http://problems:8000")
	problemsProxy, err := newProxy(problemsURL)
	if err != nil {
		logger.Fatal("failed to create problems proxy", zap.Error(err))
	}
	submissionsURL := env.Get("SUBMISSIONS_SERVICE_URL", "http://submissions:8001")
	submissionsProxy, err := newProxy(submissionsURL)
	if err != nil {
		logger.Fatal("failed to create submissions proxy", zap.Error(err))
	}
	plagiarismURL := env.Get("PLAGIARISM_SERVICE_URL", "http://plagiarism:8002")
	plagiarismProxy, err := newProxy(plagiarismURL)
	if err != nil {
		logger.Fatal("failed to create plagiarism proxy", zap.Error(err))
	}
	authURL := env.Get("AUTH_SERVICE_URL", "http://auth:8003")
	authProxy, err := newProxy(authURL)
	if err != nil {
		logger.Fatal("failed to create auth proxy", zap.Error(err))
	}

	stripAPI := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
			next.ServeHTTP(w, r)
		})
	}

	http.Handle("/api/problems/", stripAPI(problemsProxy))
	http.Handle("/api/submissions", stripAPI(submissionsProxy))
	http.Handle("/api/plagiarism/", stripAPI(plagiarismProxy))
	http.Handle("/api/auth/", stripAPI(authProxy))

	// Serve static files from the static directory
	http.Handle("/", http.FileServer(http.Dir("./static/")))

	http.HandleFunc("/health", health.HealthHandler())
	http.HandleFunc("/ready", health.HealthHandler())

	logger.Info("API Gateway starting on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("ListenAndServe failed", zap.Error(err))
	}
}
