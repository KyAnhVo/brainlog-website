package apitime

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	valid := []string{
		"2026-10-15T09:11:00.000Z",
		"1970-01-01T00:00:00.000Z",
		"2024-02-29T23:59:59.999Z", // leap day
	}
	for _, s := range valid {
		if !IsValid(s) {
			t.Errorf("IsValid(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"",
		"2026-10-15T09:11:00Z",          // no millis
		"2026-10-15T09:11:00.00Z",       // 2-digit millis
		"2026-10-15T09:11:00.0000Z",     // 4-digit millis
		"2026-10-15T09:11:00.000+00:00", // offset instead of Z
		"2026-10-15 09:11:00.000Z",      // space separator
		"2026-10-15t09:11:00.000z",      // lowercase
		"2026-02-30T00:00:00.000Z",      // rolls over
		"2025-02-29T00:00:00.000Z",      // not a leap year
		"2026-10-15T24:00:00.000Z",      // hour 24
		"2026-13-01T00:00:00.000Z",      // month 13
		"2026-10-15T09:60:00.000Z",      // minute 60
		" 2026-10-15T09:11:00.000Z",     // leading space
	}
	for _, s := range invalid {
		if IsValid(s) {
			t.Errorf("IsValid(%q) = true, want false", s)
		}
	}
}

func TestFormatRoundTrip(t *testing.T) {
	in := time.Date(2026, 10, 15, 16, 11, 0, 123_456_789, time.FixedZone("ICT", 7*3600))
	s := Format(in)
	if s != "2026-10-15T09:11:00.123Z" {
		t.Fatalf("Format = %q", s)
	}
	got, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(Normalize(in)) {
		t.Fatalf("round trip: got %v want %v", got, Normalize(in))
	}
}
