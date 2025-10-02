package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

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

	problemsURL := os.Getenv("PROBLEMS_SERVICE_URL")
	if problemsURL == "" {
		problemsURL = "http://problems:8000"
	}
	problemsProxy, err := newProxy(problemsURL)
	if err != nil {
		logger.Fatal("failed to create problems proxy", zap.Error(err))
	}
	submissionsURL := os.Getenv("SUBMISSIONS_SERVICE_URL")
	if submissionsURL == "" {
		submissionsURL = "http://submissions:8001"
	}
	submissionsProxy, err := newProxy(submissionsURL)
	if err != nil {
		logger.Fatal("failed to create submissions proxy", zap.Error(err))
	}
	plagiarismURL := os.Getenv("PLAGIARISM_SERVICE_URL")
	if plagiarismURL == "" {
		plagiarismURL = "http://plagiarism:8002"
	}
	plagiarismProxy, err := newProxy(plagiarismURL)
	if err != nil {
		logger.Fatal("failed to create plagiarism proxy", zap.Error(err))
	}

	// This single handler correctly forwards all requests starting with /api/problems/
	// to the problems-service. This includes /api/problems/ and /api/problems/1/testcases
	http.HandleFunc("/api/problems/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
		problemsProxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/api/submissions", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
		submissionsProxy.ServeHTTP(w, r)
	})

	http.HandleFunc("/api/plagiarism/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
		plagiarismProxy.ServeHTTP(w, r)
	})

	// Serve static files from the static directory
	http.Handle("/", http.FileServer(http.Dir("./static/")))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Reverse proxies are created at startup; if we reached here we're ready.
		w.WriteHeader(http.StatusOK)
	})

	logger.Info("API Gateway starting on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("ListenAndServe failed", zap.Error(err))
	}
}
