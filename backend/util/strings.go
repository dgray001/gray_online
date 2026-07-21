package util

import "strconv"

func ParseFloat(s string) float64 {
	i, e := strconv.ParseFloat(s, 64)
	if e != nil {
		return 0
	}
	return i
}

func ParseInt(s string) int {
	i, e := strconv.Atoi(s)
	if e != nil {
		return 0
	}
	return i
}
