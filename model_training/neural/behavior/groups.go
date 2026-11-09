package behavior

import "strings"

func comparisonGroup(pair Pair) string {
	a, b := strings.Split(pair.A, "-"), strings.Split(pair.B, "-")
	if a[0] == "rl" && b[0] == "rl" {
		return rlComparisonGroup(a, b)
	}
	if a[0] == b[0] && a[1] == b[1] && a[2] == b[2] {
		return "seed only"
	}
	if a[0] == b[0] && a[2] == b[2] && a[3] == b[3] {
		return "initialization only"
	}
	if a[0] == b[0] && a[1] == b[1] && a[3] == b[3] {
		return "learning rate only"
	}
	if a[1] == b[1] && a[2] == b[2] && a[3] == b[3] {
		return "target type only"
	}
	return ""
}
