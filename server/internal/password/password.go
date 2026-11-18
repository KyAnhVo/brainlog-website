// Package password hashes passwords with Argon2id using the OWASP-recommended
// configuration from requirements/Login, account.md: 19 MiB memory, 2 iterations,
// parallelism 1. Hashes are stored as PHC strings so parameters can change later
// without breaking existing hashes.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	memoryKiB   = 19 * 1024
	iterations  = 2
	parallelism = 1
	saltLen     = 16
	keyLen      = 32
)

var b64 = base64.RawStdEncoding

var ErrMalformedHash = errors.New("malformed password hash")

// Hash returns "$argon2id$v=19$m=19456,t=2,p=1$<salt>$<key>".
func Hash(password string) string {
	salt := make([]byte, saltLen)
	_, _ = rand.Read(salt)
	key := argon2.IDKey([]byte(password), salt, iterations, memoryKiB, parallelism, keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryKiB, iterations, parallelism, b64.EncodeToString(salt), b64.EncodeToString(key))
}

// Verify reports whether password matches the PHC-encoded hash, using the
// parameters stored in the hash. The comparison is constant-time.
func Verify(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=..,t=..,p=..", salt, key]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrMalformedHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrMalformedHash
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil || m == 0 || t == 0 || p == 0 {
		return false, ErrMalformedHash
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, ErrMalformedHash
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, ErrMalformedHash
	}
	got := argon2.IDKey([]byte(password), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// dummyHash lets a login for an unknown email cost the same as a real one,
// so response time does not reveal which emails exist.
var dummyHash = Hash("dummy password for timing equalization")

// BurnTime runs one verification against a throwaway hash.
func BurnTime(password string) {
	_, _ = Verify(password, dummyHash)
}
