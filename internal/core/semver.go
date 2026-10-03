package core

import (
	"regexp"
	"strconv"
)

var semverRE = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)

// ParseSemver parses MAJOR.MINOR.PATCH; ok is false for anything else.
func ParseSemver(s string) (v [3]int, ok bool) {
	m := semverRE.FindStringSubmatch(s)
	if m == nil {
		return v, false
	}
	for i := 0; i < 3; i++ {
		v[i], _ = strconv.Atoi(m[i+1])
	}
	return v, true
}

// CompareSemver returns -1, 0 or 1.
func CompareSemver(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
