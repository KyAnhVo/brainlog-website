// Package account validates and builds new accounts. Used by the createAccount endpoint
// and the create-account CLI command, so both enforce the same rules.
package account

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/KyAnhVo/brainlog-website/server/internal/ids"
	"github.com/KyAnhVo/brainlog-website/server/internal/model"
	"github.com/KyAnhVo/brainlog-website/server/internal/password"
)

const (
	MinPasswordLen = 8   // characters; matches the admin client
	MaxPasswordLen = 256 // bytes; bounds Argon2 input
	MaxNameLen     = 100
	MaxEmailLen    = 254
)

type Input struct {
	Email    string
	Password string
	Name     string
	Role     string
}

// Validate normalises in (trims email and name) and returns a client-facing message
// when it is invalid, or "" when it is valid.
func Validate(in Input) (Input, string) {
	in.Email = strings.TrimSpace(in.Email)
	in.Name = strings.TrimSpace(in.Name)
	if len(in.Email) > MaxEmailLen {
		return in, "email is too long"
	}
	// A bare address only: no display name ("Bob <bob@x>"), no surrounding junk.
	if addr, err := mail.ParseAddress(in.Email); err != nil || addr.Address != in.Email {
		return in, "email is not a valid address"
	}
	if utf8.RuneCountInString(in.Password) < MinPasswordLen {
		return in, "password must be at least 8 characters"
	}
	if len(in.Password) > MaxPasswordLen {
		return in, "password is too long"
	}
	if in.Name == "" {
		return in, "name must not be empty"
	}
	if utf8.RuneCountInString(in.Name) > MaxNameLen {
		return in, "name is too long"
	}
	if in.Role != string(model.RoleAdmin) && in.Role != string(model.RoleUser) {
		return in, `role must be "admin" or "user"`
	}
	return in, ""
}

// New builds the account row for a validated input: UUIDv7 id, Argon2id hash, creation time.
func New(in Input, now time.Time) model.Account {
	return model.Account{
		ID:             ids.NewUUIDv7(now),
		Email:          in.Email,
		PasswordHashed: password.Hash(in.Password),
		Name:           in.Name,
		CreateTime:     now,
		Role:           model.Role(in.Role),
	}
}
