package risq_mapgen_tests

import (
	"errors"
	"math/rand"
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
	for _, expr := range []string{"1 / 0", "1 +", "foo", "1 2", "min(1)", "round()", "round(1, 2)", "(1", "1 $ 2", "bar(1, 2)", "var(x)", "min(1,", "x +* 2", "", "   ", "()"} {
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

func TestExprRandomIntDrawsInclusiveRange(t *testing.T) {
	seen := map[float64]bool{}
	rng := rand.New(rand.NewSource(1))
	for range 200 {
		got, err := util.EvalExprWithRng("randomInt(0, 3)", nil, rng)
		if err != nil || got < 0 || got > 3 || got != float64(int(got)) {
			t.Fatalf("randomInt(0, 3) = %v, %v; want integer in [0, 3]", got, err)
		}
		seen[got] = true
	}
	if len(seen) != 4 {
		t.Errorf("randomInt(0, 3) only produced %v", seen)
	}
}

func TestExprRandomIsUniformFloat(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	fractional := false
	for range 200 {
		got, err := util.EvalExprWithRng("random()", nil, rng)
		if err != nil || got < 0 || got >= 1 {
			t.Fatalf("random() = %v, %v; want float in [0, 1)", got, err)
		}
		fractional = fractional || got != float64(int(got))
	}
	if !fractional {
		t.Error("random() never produced a fraction")
	}
}

func TestExprRandomIsSeedReproducible(t *testing.T) {
	draw := func(seed int64) float64 {
		got, _ := util.EvalExprWithRng("randomInt(0, 1000) + random()", nil, rand.New(rand.NewSource(seed)))
		return got
	}
	if draw(7) != draw(7) {
		t.Error("same seed gave different draws")
	}
}

func TestExprRandomRejectsMisuse(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, expr := range []string{"random(1)", "randomInt()", "randomInt(3)"} {
		if got, err := util.EvalExprWithRng(expr, nil, rng); err == nil {
			t.Errorf("%q should fail, got %v", expr, got)
		}
	}
	if got, err := util.EvalExpr("random()", nil); err == nil {
		t.Errorf("random without an rng should fail, got %v", got)
	}
}
