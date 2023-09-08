package util

func Contains[T comparable](s []T, e T) bool {
	for _, i := range s {
		if i == e {
			return true
		}
	}
	return false
}
