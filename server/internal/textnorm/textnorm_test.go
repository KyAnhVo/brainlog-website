package textnorm

import "testing"

func TestFold(t *testing.T) {
	cases := map[string]string{
		"Đà Nẵng, VN":     "da nang, vn",
		"Hồ Chí Minh, VN": "ho chi minh, vn",
		"Hải Phòng":       "hai phong",
		"BLJ-Good-Code":   "blj-good-code",
		"Chloé":           "chloe",
		"":                "",
	}
	for in, want := range cases {
		if got := Fold(in); got != want {
			t.Errorf("Fold(%q) = %q, want %q", in, got, want)
		}
	}
}
