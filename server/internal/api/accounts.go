package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/KyAnhVo/brainlog-website/server/internal/account"
	"github.com/KyAnhVo/brainlog-website/server/internal/model"
	"github.com/KyAnhVo/brainlog-website/server/internal/store"
)

// GET /api/admin/account/{id}: resolves a creatorId to a display name.
func (s *Server) handleAccountLookup(w http.ResponseWriter, r *http.Request) {
	acc, err := s.store.AccountByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Account not found")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, accountPublicDTO{ID: acc.ID, Name: acc.Name})
}

type accountCreateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

// POST /api/admin/createAccount (admin only)
func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	var req accountCreateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	in, msg := account.Validate(account.Input{Email: req.Email, Password: req.Password, Name: req.Name, Role: req.Role})
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	// Cheap pre-check so a duplicate doesn't pay for an Argon2 hash. The store's
	// uniqueness check below is still the authority (races, DB constraint).
	if _, err := s.store.AccountByEmail(r.Context(), in.Email); err == nil {
		writeError(w, http.StatusConflict, "An account with this email already exists")
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		s.internalError(w, r, err)
		return
	}

	acc := account.New(in, s.now())
	err := s.store.CreateAccount(r.Context(), acc)
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "An account with this email already exists")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, accountPublicDTO{ID: acc.ID, Name: acc.Name})
}

type accountDeleteRequest struct {
	Email *string `json:"email"`
	ID    *string `json:"id"`
}

// POST /api/admin/deleteAccount (admin only). Body is { email } or { id }, not both.
// Deleting an account also deletes all of its sessions (enforced by the store).
func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	var req accountDeleteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	var err error
	switch {
	case req.Email != nil && req.ID == nil:
		err = s.store.DeleteAccountByEmail(r.Context(), *req.Email)
	case req.ID != nil && req.Email == nil:
		err = s.store.DeleteAccountByID(r.Context(), *req.ID)
	default:
		writeError(w, http.StatusBadRequest, "Provide exactly one of email or id")
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Account not found")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	// If admins delete themselves, their cookie now points at nothing; clear it.
	if targetsSelf(req, currentAccount(r.Context())) {
		s.clearSessionCookie(w)
	}
	writeJSON(w, http.StatusOK, empty{})
}

// targetsSelf reports whether the delete request named the caller's own account.
func targetsSelf(req accountDeleteRequest, me model.Account) bool {
	if req.ID != nil {
		return *req.ID == me.ID
	}
	return req.Email != nil && strings.EqualFold(strings.TrimSpace(*req.Email), me.Email)
}
