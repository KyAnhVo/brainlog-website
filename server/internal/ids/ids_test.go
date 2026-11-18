package ids

import (
	"fmt"
	"regexp"
	"testing"
	"time"
)

var uuidv7Shape = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewUUIDv7(t *testing.T) {
	now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	a := NewUUIDv7(now)
	if !uuidv7Shape.MatchString(a) {
		t.Fatalf("bad shape: %s", a)
	}
	// Timestamp prefix encodes now in ms.
	if got, want := a[0:8]+a[9:13], fmt.Sprintf("%012x", now.UnixMilli()); got != want {
		t.Fatalf("timestamp prefix = %s, want %s", got, want)
	}
	if b := NewUUIDv7(now.Add(time.Millisecond)); b <= a {
		t.Fatalf("not time-ordered: %s <= %s", b, a)
	}
}

func TestSessionIDRoundTrip(t *testing.T) {
	id := NewSessionID()
	got, ok := ParseSessionID(id.String())
	if !ok || got != id {
		t.Fatalf("round trip failed: %v %v", got, ok)
	}
	for _, bad := range []string{"", "abc", id.String() + "0", "zz" + id.String()[2:]} {
		if _, ok := ParseSessionID(bad); ok {
			t.Errorf("ParseSessionID(%q) ok", bad)
		}
	}
}

func TestNewTrackingNumber(t *testing.T) {
	if len(trackingAlphabet) != 32 {
		t.Fatalf("alphabet has %d symbols; must be 32 for an unbiased b&31", len(trackingAlphabet))
	}
	shape := regexp.MustCompile(`^BLJ-[2-9A-HJ-NP-Z]{4}-[2-9A-HJ-NP-Z]{4}$`)
	seen := map[string]bool{}
	for range 1000 {
		n := NewTrackingNumber()
		if !shape.MatchString(n) {
			t.Fatalf("bad shape: %s", n)
		}
		if seen[n] {
			t.Fatalf("duplicate in 1000 draws: %s", n)
		}
		seen[n] = true
	}
}
