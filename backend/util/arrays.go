package util

import "math/rand"

func Contains[T comparable](s []T, e T) bool {
	for _, i := range s {
		if i == e {
			return true
		}
	}
	return false
}

func Shuffle[T any](s []T) []T {
	for i := range s {
		j := rand.Intn(i + 1)
		s[i], s[j] = s[j], s[i]
	}
	return s
}

// Should only be used if order doesn't matter, as it will change the order
// Taken from https://stackoverflow.com/a/37335777
func FastDelete[T any](s []T, index int) []T {
	last_index := len(s) - 1
	s[index] = s[last_index]
	var zero T
	// Clear for GC if T is a pointer type
	s[last_index] = zero
	return s[:last_index]
}
