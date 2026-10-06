package util

import "testing"

func TestExprDottedVariableNames(t *testing.T) {
	lookup := func(name string) (float64, error) {
		if name == "b.space_x" {
			return 4, nil
		}
		return 0, nil
	}
	got, err := EvalExprWithLookup("var(b.space_x) * 2 + 1", nil, lookup)
	if err != nil || got != 9 {
		t.Fatalf("got %v, %v; want 9", got, err)
	}
}
