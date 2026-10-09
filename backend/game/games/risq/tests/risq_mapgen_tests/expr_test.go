package risq_mapgen_tests

import (
	"errors"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/util"
)

func TestExprEvaluates(t *testing.T) {
	vars := map[string]float64{"x": 4, "num_players": 3}
	cases := []struct {
		expr string
		want float64
	}{
		{"1 + 2 * 3", 7},
		{"(1 + 2) * 3", 9},
		{"10 / 4", 2.5},
		{"-2 + 5", 3},
		{"--2", 2},
		{"+3", 3},
		{"1.5 * 2", 3},
		{"x * x - 1", 15},
		{"2 + num_players / 3", 3},
		{"min(3, 1, 2)", 1},
		{"max(1, x, 2)", 4},
		{"max(min(5, 2), 1) * 2", 4},
		{"round(2.49)", 2},
		{"round(2.5)", 3},
		{"round(-2.5)", -3},
		{"5 * round(14 / 5)", 15},
		{"  7  ", 7},
		{"8 - 3 - 2", 3},
		{"8 / 4 / 2", 1},
		{"2 * -3", -6},
		{"2 - -3", 5},
	}
	for _, c := range cases {
		got, err := util.EvalExpr(c.expr, vars)
		if err != nil || got != c.want {
			t.Errorf("%q = %v, %v; want %v", c.expr, got, err, c.want)
		}
	}
}

func TestExprRejectsBadInput(t *testing.T) {
	for _, expr := range []string{"1 / 0", "1 +", "foo", "1 2", "min(1)", "round()", "round(1, 2)", "(1", "1 $ 2", "bar(1, 2)", "var(x)", "min(1,", "x +* 2", "", "   ", "()", "1 +* 2"} {
		if got, err := util.EvalExpr(expr, map[string]float64{"x": 1}); err == nil {
			t.Errorf("%q should fail, got %v", expr, got)
		}
	}
}

func TestExprLookupErrorsPropagate(t *testing.T) {
	failing := func(name string) (float64, error) { return 0, errors.New("no such var " + name) }
	if _, err := util.EvalExprWithLookup("1 + var(missing)", nil, failing); err == nil || !strings.Contains(err.Error(), "no such var missing") {
		t.Errorf("lookup failure was swallowed: %v", err)
	}
}

func TestExprLookupResolvesVar(t *testing.T) {
	lookup := func(name string) (float64, error) { return map[string]float64{"gold": 40}[name], nil }
	got, err := util.EvalExprWithLookup("var(gold) / 2 + 1", nil, lookup)
	if err != nil || got != 21 {
		t.Errorf("var() = %v, %v; want 21", got, err)
	}
}
