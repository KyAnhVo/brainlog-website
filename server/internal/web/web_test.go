package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

var built = fstest.MapFS{
	"index.html":             {Data: []byte("<!doctype html><title>app</title>")},
	"favicon.svg":            {Data: []byte("<svg/>")},
	"assets/index-abc123.js": {Data: []byte("console.log(1)")},
	"assets/logo-def456.png": {Data: []byte("png")},
	"mockServiceWorker.js":   {Data: []byte("// msw")},
}

func get(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestSPA(t *testing.T) {
	h := fromFS(built)
	cases := []struct {
		path        string
		status      int
		contentType string
		cache       string
		body        string
	}{
		{"/", 200, "text/html", "no-cache", "<title>app</title>"},
		{"/index.html", 200, "text/html", "no-cache", "<title>app</title>"},
		{"/tracking", 200, "text/html", "no-cache", "<title>app</title>"},               // client route
		{"/tracking/BLJ-2026-0001", 200, "text/html", "no-cache", "<title>app</title>"}, // nested client route
		{"/assets/index-abc123.js", 200, "text/javascript", "public, max-age=31536000, immutable", "console.log(1)"},
		{"/favicon.svg", 200, "image/svg+xml", "no-cache", "<svg/>"},
		{"/assets/old-gone.js", 404, "", "", ""}, // never HTML for a missing asset
		{"/robots.txt", 404, "", "", ""},
		{"/mockServiceWorker.js", 404, "", "", ""},                      // dev-only file is never served
		{"/assets", 200, "text/html", "no-cache", "<title>app</title>"}, // a directory is not a file
		// Cleaned to /etc/passwd, which can't escape the embedded FS: just an unknown client route.
		{"/../../etc/passwd", 200, "text/html", "no-cache", "<title>app</title>"},
	}
	for _, tc := range cases {
		rec := get(h, "GET", tc.path)
		if rec.Code != tc.status {
			t.Errorf("%s: status %d, want %d", tc.path, rec.Code, tc.status)
			continue
		}
		if tc.status != 200 {
			continue
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, tc.contentType) {
			t.Errorf("%s: Content-Type %q, want %q", tc.path, ct, tc.contentType)
		}
		if cc := rec.Header().Get("Cache-Control"); cc != tc.cache {
			t.Errorf("%s: Cache-Control %q, want %q", tc.path, cc, tc.cache)
		}
		if !strings.Contains(rec.Body.String(), tc.body) {
			t.Errorf("%s: body %q", tc.path, rec.Body.String())
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: missing nosniff", tc.path)
		}
	}

	page := get(h, "GET", "/tracking")
	if !strings.Contains(page.Header().Get("Content-Security-Policy"), "script-src 'self'") ||
		page.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("HTML is missing security headers: %v", page.Header())
	}

	if rec := get(h, "POST", "/"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: %d", rec.Code)
	}
	if rec := get(h, "HEAD", "/assets/index-abc123.js"); rec.Code != 200 || rec.Body.Len() != 0 {
		t.Errorf("HEAD: %d, %d body bytes", rec.Code, rec.Body.Len())
	}
}

func TestNotBuilt(t *testing.T) {
	h := fromFS(fstest.MapFS{".gitkeep": {}})
	if rec := get(h, "GET", "/"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503", rec.Code)
	}
}
