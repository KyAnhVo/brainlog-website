// Package textnorm implements the case- and accent-insensitive comparison that
// GET /api/admin/tracking requires.
package textnorm

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// đ/Đ (Vietnamese) and a few other letters carry their "accent" as part of the
// base letter, so NFD does not split them off. Map them by hand.
var letterFold = strings.NewReplacer(
	"đ", "d", "Đ", "d",
	"ø", "o", "Ø", "o",
	"ł", "l", "Ł", "l",
	"ß", "ss",
)

// Fold lowercases s and strips diacritics: "Đà Nẵng" -> "da nang".
func Fold(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		out = s
	}
	return strings.ToLower(letterFold.Replace(out))
}
