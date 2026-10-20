package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeIndex(t *testing.T) {
	rec := httptest.NewRecorder()
	serveIndex(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Herald Playground") {
		t.Fatal("missing markup")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type %q", ct)
	}
}
