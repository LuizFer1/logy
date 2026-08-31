package update

import "testing"

func TestNormalizeVersion(t *testing.T) {
	if got := NormalizeVersion("v1.2.0"); got != "1.2.0" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeVersion("1.2.0"); got != "1.2.0" {
		t.Fatalf("got %q", got)
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.2.0", "1.2.0", 0},
		{"v1.2.0", "1.1.9", 1},
		{"1.2.0", "v1.3.0", -1},
		{"1.2.0", "1.2.0-rc.1", 1}, // non-numeric suffix treated as less
		{"dev", "1.0.0", -1},
		{"1.0.0", "dev", 1},
		{"dev", "dev", 0},
	}
	for _, tc := range cases {
		got := CompareSemver(tc.a, tc.b)
		if got != tc.want {
			t.Fatalf("CompareSemver(%q,%q)=%d want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
