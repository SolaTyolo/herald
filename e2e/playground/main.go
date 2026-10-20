// Command playground serves the Herald manual e2e UI and proxies API calls.
//
//	HERALD_API=http://localhost:8080 PLAYGROUND_ADDR=:8090 go run ./e2e/playground
package main

import (
	_ "embed"
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

//go:embed index.html
var indexHTML []byte

func main() {
	addr := flag.String("addr", envOr("PLAYGROUND_ADDR", ":8090"), "listen address")
	apiURL := flag.String("api", envOr("HERALD_API", "http://localhost:8188"), "Herald API base URL")
	flag.Parse()

	target, err := url.Parse(*apiURL)
	if err != nil {
		log.Fatalf("invalid api url: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "api unreachable: "+err.Error(), http.StatusBadGateway)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/v1/", proxy.ServeHTTP)
	mux.HandleFunc("/health", proxy.ServeHTTP)

	log.Printf("playground on http://localhost%s (api %s)", *addr, strings.TrimRight(*apiURL, "/"))
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(indexHTML)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
