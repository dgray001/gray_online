package invariants

import (
	"math"
	"testing"
)

func bound(t *testing.T, label string, value, low, high float64) {
	t.Helper()
	if math.IsNaN(value) || math.IsInf(value, 0) || value < low || value > high {
		t.Errorf("%s = %v, want [%v, %v]", label, value, low, high)
	}
}

func number(data object, key string) float64 { return data[key].(float64) }

func objects(data object, key string) []object {
	var result []object
	for _, value := range data[key].([]any) {
		result = append(result, value.(object))
	}
	return result
}
