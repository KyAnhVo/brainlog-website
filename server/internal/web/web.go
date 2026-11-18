// Package web serves the two built frontends, embedded into the binary:
//
//	dist/customer  -> /         (customer-facing-client)
//	dist/admin     -> /admin/   (admin-panel-client, built with base "/admin/")
//
// The dist folders are filled by scripts/build.sh or the Dockerfile. In a plain checkout
// they hold only a .gitkeep, and the handlers answer 503 "frontend not built".
package web

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:embed all:dist
var dist embed.FS

// Customer returns the handler for the customer site, to be mounted at "/".
func Customer() http.Handler { return newSPA("dist/customer") }

// Admin returns the handler for the admin panel. Mount it at "/admin/" behind
// http.StripPrefix("/admin", ...), so it sees paths like "/tracking".
func Admin() http.Handler { return newSPA("dist/admin") }

// The SPAs load fonts from Google Fonts, and MUI (emotion) injects <style> tags at runtime,
// hence 'unsafe-inline' for styles only. Scripts must come from this origin.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"img-src 'self' data:; " +
	"connect-src 'self'; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

type spa struct {
	files fs.FS
	index []byte // nil when the frontend was not built into this binary
}

func newSPA(dir string) *spa {
	sub, err := fs.Sub(dist, dir)
	if err != nil {
		panic(err) // dir is a constant that exists in the embed
	}
	return fromFS(sub)
}

// fromFS serves the built app in fsys (index.html at its root).
func fromFS(fsys fs.FS) *spa {
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		index = nil
	}
	return &spa{files: fsys, index: index}
}

func setCommonHeaders(h http.Header) {
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

func (s *spa) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	setCommonHeaders(w.Header())
	if s.index == nil {
		http.Error(w, "frontend not built into this server binary (run scripts/build.sh)", http.StatusServiceUnavailable)
		return
	}

	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	switch {
	case name == "mockServiceWorker.js":
		// The MSW worker is dev-only; never serve it even if a build copied it in.
		http.NotFound(w, r)
	case name == "" || name == "index.html":
		s.serveIndex(w, r)
	case isFile(s.files, name):
		s.serveFile(w, r, name)
	case path.Ext(name) != "":
		// A missing file with an extension (e.g. an old /assets/x.js) is a real 404:
		// answering with index.html would hand HTML to a <script> tag.
		http.NotFound(w, r)
	default:
		// Anything else is a client-side route: serve the app shell.
		s.serveIndex(w, r)
	}
}

func isFile(fsys fs.FS, name string) bool {
	f, err := fs.Stat(fsys, name)
	return err == nil && !f.IsDir()
}

func (s *spa) serveIndex(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-cache") // always revalidate, so deploys take effect immediately
	h.Set("Content-Security-Policy", contentSecurityPolicy)
	h.Set("X-Frame-Options", "DENY")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(s.index))
}

func (s *spa) serveFile(w http.ResponseWriter, r *http.Request, name string) {
	b, err := fs.ReadFile(s.files, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(name, "assets/") {
		// Vite puts a content hash in every file name under assets/: safe to cache forever.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	// ServeContent sets Content-Type from the extension and handles Range/HEAD.
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(b))
}
