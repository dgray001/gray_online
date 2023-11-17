package util

func AbsInt(x int) int {
	return AbsDiffInt(x, 0)
}

func AbsDiffInt(x int, y int) int {
	if x < y {
		return y - x
	}
	return x - y
}
