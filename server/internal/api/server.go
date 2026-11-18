// Package api is the HTTP layer: routing, auth, validation, and JSON shapes.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/KyAnhVo/brainlog-website/server/internal/apitime"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
)

// AcceptedBuffer is the clock-skew allowance for "in the future" checks.
const AcceptedBuffer = 5 * time.Minute

const maxBodyBytes = 64 << 10

type Config struct {
	Store store.Store
	// Now returns the current server time; tests inject a fixed clock. Defaults to time.Now.
	Now func() time.Time
	// CookieSecure sets the Secure flag on the session cookie. Must be true behind HTTPS.
	CookieSecure bool
	Logger       *slog.Logger
}

type Server struct {
	store        store.Store
	clock        func() time.Time
	cookieSecure bool
	log          *slog.Logger
}

// New returns the API as an http.Handler.
func New(cfg Config) http.Handler {
	s := &Server{
		store:        cfg.Store,
		clock:        cfg.Now,
		cookieSecure: cfg.CookieSecure,
		log:          cfg.Logger,
	}
	if s.clock == nil {
		s.clock = time.Now
	}
	if s.log == nil {
		s.log = slog.Default()
	}

	mux := http.NewServeMux()

	// Customer-facing (public).
	mux.HandleFunc("GET /api/tracking/{trackingNumber}", s.handleCustomerTracking)

	// Admin auth.
	mux.HandleFunc("POST /api/admin/login", s.handleLogin)
	mux.HandleFunc("POST /api/admin/logout", s.handleLogout)
	mux.Handle("GET /api/admin/me", s.requireSession(s.handleMe))

	// Admin trackings.
	mux.Handle("GET /api/admin/tracking", s.requireSession(s.handleListTrackings))
	mux.Handle("GET /api/admin/tracking/{trackingNumber}", s.requireSession(s.handleAdminTracking))
	mux.Handle("POST /api/admin/tracking/{trackingNumber}/status", s.requireSession(s.handleAddStatus))

	// Admin accounts.
	mux.Handle("GET /api/admin/account/{id}", s.requireSession(s.handleAccountLookup))
	mux.Handle("POST /api/admin/createAccount", s.requireAdmin(s.handleCreateAccount))
	mux.Handle("POST /api/admin/deleteAccount", s.requireAdmin(s.handleDeleteAccount))

	// Anything else under /api is a JSON 404, so clients always get an ApiError.
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "No such endpoint")
	})

	return s.recoverer(s.logRequests(securityHeaders(mux)))
}

// now is the server clock at millisecond precision, the precision of every stored time.
func (s *Server) now() time.Time {
	return apitime.Normalize(s.clock())
}

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, apiError{ErrorCode: status, ErrorMsg: msg})
}

// errBadBody carries a message that is safe to show the client.
type errBadBody struct {
	status int
	msg    string
}

func (e *errBadBody) Error() string { return e.msg }

// decodeJSON reads exactly one JSON object into dst. Unknown fields, trailing data,
// wrong types, oversized bodies, and non-JSON content types are all rejected.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		return &errBadBody{http.StatusUnsupportedMediaType, "Content-Type must be application/json"}
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		var typeErr *json.UnmarshalTypeError
		switch {
		case errors.As(err, &tooBig):
			return &errBadBody{http.StatusRequestEntityTooLarge, "Request body too large"}
		case errors.As(err, &typeErr):
			return &errBadBody{http.StatusBadRequest, "Field " + typeErr.Field + " has the wrong type"}
		default:
			return &errBadBody{http.StatusBadRequest, "Malformed JSON body: " + err.Error()}
		}
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return &errBadBody{http.StatusBadRequest, "Body must contain a single JSON object"}
	}
	return nil
}

// writeDecodeError responds to an error from decodeJSON.
func writeDecodeError(w http.ResponseWriter, err error) {
	var bad *errBadBody
	if errors.As(err, &bad) {
		writeError(w, bad.status, bad.msg)
		return
	}
	writeError(w, http.StatusBadRequest, "Bad request body")
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.ErrorContext(r.Context(), "internal error", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "Internal server error")
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		// Never log bodies or query strings' secrets: only method, path, status, latency.
		s.log.InfoContext(r.Context(), "request",
			"method", r.Method, "path", r.URL.Path, "status", rec.status,
			"duration_ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.log.ErrorContext(r.Context(), "panic", "method", r.Method, "path", r.URL.Path, "panic", v)
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
