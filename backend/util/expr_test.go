package util

import (
	"fmt"
	"testing"
)

func TestEvalExprFunctions(t *testing.T) {
	vars := map[string]float64{"a": 3, "b": 7}
	lookup := func(name string) (float64, error) {
		if name == "x" {
			return 10, nil
		}
		return 0, fmt.Errorf("unknown %s", name)
	}
	cases := map[string]float64{
		"min(a, b)":                       3,
		"max(a, b, 12) - 1":               11,
		"min(max(a, 5), b)":               5,
		"var(x) * 2 + min(1, 2)":          21,
		"max(0, 24 - var(x)) / max(1, a)": 14.0 / 3,
	}
	for expr, want := range cases {
		got, err := EvalExprWithLookup(expr, vars, lookup)
		if err != nil || got != want {
			t.Errorf("%s = %v (err %v), want %v", expr, got, err, want)
		}
	}
	for _, bad := range []string{"min(a)", "var(y)", "foo(a, b)", "min(a b)", "var(1)"} {
		if _, err := EvalExprWithLookup(bad, vars, lookup); err == nil {
			t.Errorf("%s should fail", bad)
		}
	}
	if _, err := EvalExpr("var(x)", vars); err == nil {
		t.Errorf("var() without a lookup should fail")
	}
}
