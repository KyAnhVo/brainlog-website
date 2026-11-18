package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/KyAnhVo/brainlog-website/server/internal/ids"
	"github.com/KyAnhVo/brainlog-website/server/internal/model"
	"github.com/KyAnhVo/brainlog-website/server/internal/password"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
)

const (
	SessionCookieName = "brainlog_session"
	// requirements/Login, account.md: "A cookie is valid for 7 days" ...
	cookieLifetime = 7 * 24 * time.Hour
	// ... and the stored session has ends_time = start_time + 30 days.
	// Both are implemented as written, so a browser session effectively lasts 7 days.
	sessionLifetime = 30 * 24 * time.Hour
)

// The cookie is only ever needed by the admin API, so browsers don't send it to the
// customer site, the static files, or the customer API on the same host.
const cookiePath = "/api/admin"

type ctxKey int

const accountKey ctxKey = iota

// currentAccount returns the account attached by requireSession.
func currentAccount(ctx context.Context) model.Account {
	a, _ := ctx.Value(accountKey).(model.Account)
	return a
}

func (s *Server) setSessionCookie(w http.ResponseWriter, id ids.SessionID) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    id.String(),
		Path:     cookiePath,
		MaxAge:   int(cookieLifetime / time.Second),
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     cookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func sessionIDFromRequest(r *http.Request) (ids.SessionID, bool) {
	c, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ids.SessionID{}, false
	}
	return ids.ParseSessionID(c.Value)
}

// authenticate resolves the request's cookie to an account. ok is false when there is
// no cookie, the session is unknown/invalid/expired, or its account no longer exists.
func (s *Server) authenticate(r *http.Request) (model.Account, bool, error) {
	sid, ok := sessionIDFromRequest(r)
	if !ok {
		return model.Account{}, false, nil
	}
	sess, err := s.store.SessionByID(r.Context(), sid)
	if errors.Is(err, store.ErrNotFound) {
		return model.Account{}, false, nil
	}
	if err != nil {
		return model.Account{}, false, err
	}
	if !sess.Active(s.now()) {
		return model.Account{}, false, nil
	}
	acc, err := s.store.AccountByID(r.Context(), sess.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return model.Account{}, false, nil
	}
	if err != nil {
		return model.Account{}, false, err
	}
	return acc, true, nil
}

func (s *Server) requireSession(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acc, ok, err := s.authenticate(r)
		if err != nil {
			s.internalError(w, r, err)
			return
		}
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in or session expired")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), accountKey, acc)))
	})
}

// requireAdmin answers 401 for non-admins, as the account endpoints specify.
func (s *Server) requireAdmin(next http.HandlerFunc) http.Handler {
	return s.requireSession(func(w http.ResponseWriter, r *http.Request) {
		if currentAccount(r.Context()).Role != model.RoleAdmin {
			writeError(w, http.StatusUnauthorized, "Admin role required")
			return
		}
		next(w, r)
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// POST /api/admin/login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	acc, err := s.store.AccountByEmail(r.Context(), req.Email)
	if errors.Is(err, store.ErrNotFound) {
		password.BurnTime(req.Password) // same cost as a real check: don't leak which emails exist
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	ok, err := password.Verify(req.Password, acc.PasswordHashed)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Logging in replaces whatever session this browser had.
	if old, ok := sessionIDFromRequest(r); ok {
		_ = s.store.InvalidateSession(r.Context(), old)
	}

	now := s.now()
	sess := model.Session{
		ID:        ids.NewSessionID(),
		UserID:    acc.ID,
		StartTime: now,
		EndsTime:  now.Add(sessionLifetime),
		IsValid:   true,
	}
	if err := s.store.CreateSession(r.Context(), sess); err != nil {
		s.internalError(w, r, err)
		return
	}
	s.setSessionCookie(w, sess.ID)
	writeJSON(w, http.StatusOK, toLoginSuccess(acc))
}

// POST /api/admin/logout. Always succeeds, so a stale client can always log out.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if sid, ok := sessionIDFromRequest(r); ok {
		if err := s.store.InvalidateSession(r.Context(), sid); err != nil && !errors.Is(err, store.ErrNotFound) {
			s.internalError(w, r, err)
			return
		}
	}
	s.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, empty{})
}

// GET /api/admin/me
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, toLoginSuccess(currentAccount(r.Context())))
}
