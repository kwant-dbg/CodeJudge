package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func newProxy(targetHost string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(targetHost)
	if err != nil {
		return nil, err
	}
	return httputil.NewSingleHostReverseProxy(url), nil
}

func main() {
	problemsProxy, _ := newProxy("http://problems:8000")
	submissionsProxy, _ := newProxy("http://submissions:8001")
	plagiarismProxy, _ := newProxy("http://plagiarism:8002")

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

	log.Println("API Gateway starting on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

