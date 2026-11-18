// Package ids generates identifiers: UUIDv7 for rows, and 128-bit random session ids.
package ids

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// NewUUIDv7 returns an RFC 9562 UUIDv7: 48-bit Unix ms timestamp, version 7,
// variant 10, and 74 random bits. Lexicographic order follows creation time.
func NewUUIDv7(now time.Time) string {
	var b [16]byte
	_, _ = rand.Read(b[6:]) // crypto/rand.Read never fails (Go 1.24+)
	ms := uint64(now.UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	b[6] = (b[6] & 0x0f) | 0x70 // version 7
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10

	var out [36]byte
	hex.Encode(out[0:8], b[0:4])
	out[8] = '-'
	hex.Encode(out[9:13], b[4:6])
	out[13] = '-'
	hex.Encode(out[14:18], b[6:8])
	out[18] = '-'
	hex.Encode(out[19:23], b[8:10])
	out[23] = '-'
	hex.Encode(out[24:36], b[10:16])
	return string(out[:])
}

// SessionID is 128 random bits (requirements/Login, account.md).
type SessionID [16]byte

func NewSessionID() SessionID {
	var id SessionID
	_, _ = rand.Read(id[:])
	return id
}

// String is the cookie encoding: 32 lowercase hex characters.
func (id SessionID) String() string {
	return hex.EncodeToString(id[:])
}

// ParseSessionID decodes the cookie value; ok is false for anything malformed.
func ParseSessionID(s string) (id SessionID, ok bool) {
	if len(s) != 32 {
		return id, false
	}
	if _, err := hex.Decode(id[:], []byte(s)); err != nil {
		return id, false
	}
	return id, true
}

// Tracking numbers are public lookup keys (GET /api/tracking/:n needs no login), so they
// must be unguessable. 32 symbols, no look-alikes (no 0/O, 1/I): 8 random symbols = 40 bits.
const trackingAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// NewTrackingNumber returns a random number like "BLJ-7K3M-Q9TX".
func NewTrackingNumber() string {
	var r [8]byte
	_, _ = rand.Read(r[:])
	var sym [8]byte
	for i, b := range r {
		sym[i] = trackingAlphabet[b&31] // 32 divides 256: uniform
	}
	return "BLJ-" + string(sym[:4]) + "-" + string(sym[4:])
}
