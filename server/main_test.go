package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The embedded frontends are not built in a plain checkout, so they answer 503;
// that is enough to tell which area a path was routed to.
func TestRoutes(t *testing.T) {
	apiHit := ""
	h := routes(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiHit = r.URL.Path
		w.WriteHeader(299)
	}))
	cases := []struct {
		path     string
		status   int
		location string
	}{
		{"/api/tracking/X", 299, ""},
		{"/api/admin/me", 299, ""},
		{"/admin", http.StatusTemporaryRedirect, "/admin/"}, // ServeMux's trailing-slash redirect
		{"/admin/", 503, ""},
		{"/admin/tracking", 503, ""},
		{"/", 503, ""},
		{"/tracking/X", 503, ""},
		{"/administrator", 503, ""}, // customer site, not the admin panel
	}
	for _, tc := range cases {
		apiHit = ""
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", tc.path, nil))
		if rec.Code != tc.status {
			t.Errorf("%s: status %d, want %d", tc.path, rec.Code, tc.status)
		}
		if tc.location != "" && rec.Header().Get("Location") != tc.location {
			t.Errorf("%s: Location %q", tc.path, rec.Header().Get("Location"))
		}
		if tc.status == 299 && apiHit != tc.path {
			t.Errorf("%s: API saw %q (prefix must not be stripped)", tc.path, apiHit)
		}
	}
}
