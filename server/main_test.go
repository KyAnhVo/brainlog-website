package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// stub answers with a fixed status and records the path it received.
type stub struct {
	code int
	seen string
}

func (s *stub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.seen = r.URL.Path
	w.WriteHeader(s.code)
}

func TestRoutes(t *testing.T) {
	api, admin, customer := &stub{code: 201}, &stub{code: 202}, &stub{code: 203}
	h := routes(api, admin, customer)
	cases := []struct {
		path   string
		status int
		seen   string // path the handler received ("" = redirect, nothing received)
	}{
		{"/api/tracking/X", 201, "/api/tracking/X"}, // API keeps its full path
		{"/api/admin/me", 201, "/api/admin/me"},
		{"/admin", http.StatusTemporaryRedirect, ""}, // ServeMux's trailing-slash redirect
		{"/admin/", 202, "/"},                        // admin SPA sees in-app paths
		{"/admin/tracking/X", 202, "/tracking/X"},
		{"/", 203, "/"},
		{"/tracking/X", 203, "/tracking/X"},
		{"/administrator", 203, "/administrator"}, // customer site, not the admin panel
		{"/healthz", 200, ""},                     // answered by routes itself
	}
	for _, tc := range cases {
		api.seen, admin.seen, customer.seen = "", "", ""
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", tc.path, nil))
		if rec.Code != tc.status {
			t.Errorf("%s: status %d, want %d", tc.path, rec.Code, tc.status)
			continue
		}
		if tc.status == http.StatusTemporaryRedirect {
			if loc := rec.Header().Get("Location"); loc != "/admin/" {
				t.Errorf("%s: Location %q", tc.path, loc)
			}
			continue
		}
		if got := api.seen + admin.seen + customer.seen; got != tc.seen {
			t.Errorf("%s: handler saw %q, want %q", tc.path, got, tc.seen)
		}
	}
}
