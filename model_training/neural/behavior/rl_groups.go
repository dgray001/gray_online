package behavior

func rlComparisonGroup(a []string, b []string) string {
	seedA, seedB := a[len(a)-1], b[len(b)-1]
	widthA, widthB := "h0", "h0"
	if len(a) == 6 {
		widthA = a[4]
	}
	if len(b) == 6 {
		widthB = b[4]
	}
	if widthA != widthB {
		if a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && seedA == seedB {
			return "architecture only"
		}
		return ""
	}
	if a[1] == b[1] && a[2] == b[2] && a[3] == b[3] {
		return "seed only"
	}
	if a[2] == b[2] && a[3] == b[3] && seedA == seedB {
		return "initialization only"
	}
	if a[1] == b[1] && a[3] == b[3] && seedA == seedB {
		return "learning rate only"
	}
	if a[1] == b[1] && a[2] == b[2] && seedA == seedB {
		return "entropy only"
	}
	return ""
}
