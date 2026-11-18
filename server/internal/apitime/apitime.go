// Package apitime implements the wire format for times (requirements/Notes.md):
// UTC, RFC 3339, millisecond precision, `Z` suffix, exactly yyyy-mm-ddThh:mm:ss.xxxZ.
package apitime

import (
	"errors"
	"regexp"
	"time"
)

const layout = "2006-01-02T15:04:05.000Z"

var shape = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)

var ErrInvalid = errors.New("time must be of the form yyyy-mm-ddThh:mm:ss.xxxZ")

// Format renders t in the wire format.
func Format(t time.Time) string {
	return t.UTC().Format(layout)
}

// Parse accepts only the exact wire format. Out-of-range fields (e.g. Feb 30, 24:00)
// are rejected rather than rolled over.
func Parse(s string) (time.Time, error) {
	if !shape.MatchString(s) {
		return time.Time{}, ErrInvalid
	}
	t, err := time.Parse(layout, s)
	if err != nil || Format(t) != s {
		return time.Time{}, ErrInvalid
	}
	return t, nil
}

// IsValid reports whether s is exactly in the wire format.
func IsValid(s string) bool {
	_, err := Parse(s)
	return err == nil
}

// Normalize truncates to millisecond precision in UTC, the precision every stored time has.
func Normalize(t time.Time) time.Time {
	return t.UTC().Truncate(time.Millisecond)
}
