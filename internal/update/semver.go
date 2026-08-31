package update

import (
	"strconv"
	"strings"
)

// NormalizeVersion trims space and a leading "v"/"V" prefix.
func NormalizeVersion(s string) string {
	s = strings.TrimSpace(s)
	return strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
}

// CompareSemver returns -1 if a < b, 0 if equal, 1 if a > b.
// "dev" is older than any release. Leading v is ignored. Compares major.minor.patch
// as ints (missing parts = 0). A non-numeric segment suffix marks a pre-release,
// which sorts below the same numeric version without a suffix.
func CompareSemver(a, b string) int {
	na, nb := NormalizeVersion(a), NormalizeVersion(b)
	if na == "dev" && nb == "dev" {
		return 0
	}
	if na == "dev" {
		return -1
	}
	if nb == "dev" {
		return 1
	}

	pa, prea := parseSemverParts(na)
	pb, preb := parseSemverParts(nb)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	switch {
	case prea && !preb:
		return -1
	case !prea && preb:
		return 1
	default:
		return 0
	}
}

func parseSemverParts(s string) (parts [3]int, prerelease bool) {
	segs := strings.Split(s, ".")
	for i := 0; i < 3 && i < len(segs); i++ {
		n, rest, ok := leadingInt(segs[i])
		if !ok {
			prerelease = true
			continue
		}
		parts[i] = n
		if rest != "" {
			prerelease = true
		}
	}
	if len(segs) > 3 {
		prerelease = true
	}
	return parts, prerelease
}

func leadingInt(s string) (n int, rest string, ok bool) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, s, false
	}
	v, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0, s, false
	}
	return v, s[i:], true
}
