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

func ParseBool(s string) bool {
	trues := []string{"True", "true"}
	for _, t := range trues {
		if t == s {
			return true
		}
	}
	return false
}
