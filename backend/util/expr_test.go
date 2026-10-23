package util

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestExprPrecedenceAndAssociativity(t *testing.T) {
	cases := map[string]float64{
		"10 - 4 - 3":               3,
		"100 / 10 / 5":             2,
		"2 + 3 * 4 - 6 / 2":        11,
		"-2 * -3":                  6,
		"- - 4":                    4,
		"+5 - -5":                  10,
		"1 + 2 == 3":               1,
		"(1 < 2) + (2 < 1) * 5":    1,
		"min(3, max(1, 2), 9) * 2": 4,
		"round(2.5) + round(-0.4)": 3,
		"(2 + 3) * (4 - 1)":        15,
	}
	for expr, want := range cases {
		if got, err := EvalExpr(expr, nil); err != nil || got != want {
			t.Errorf("EvalExpr(%q) = %v, %v; want %v", expr, got, err, want)
		}
	}
}

func TestExprEvaluatesLeftToRightAndStopsAtTheFirstError(t *testing.T) {
	var seen []string
	lookup := func(name string) (float64, error) {
		seen = append(seen, name)
		if name == "bad" {
			return 0, errTest
		}
		return 1, nil
	}
	if _, err := EvalExprWithLookup("var(a) + max(var(b), var(c)) * var(d)", nil, lookup); err != nil {
		t.Fatal(err)
	}
	if want := []string{"a", "b", "c", "d"}; !reflect.DeepEqual(seen, want) {
		t.Errorf("lookups in order %v, want %v", seen, want)
	}
	seen = nil
	if _, err := EvalExprWithLookup("var(a) + var(bad) + var(never)", nil, lookup); err != errTest {
		t.Errorf("error = %v, want the lookup error", err)
	}
	if want := []string{"a", "bad"}; !reflect.DeepEqual(seen, want) {
		t.Errorf("lookups before the failure %v, want %v", seen, want)
	}
}

type testError struct{}

func (testError) Error() string { return "lookup failed" }

var errTest error = testError{}

func TestExprDrawsRandomNumbersInTextOrder(t *testing.T) {
	draws := func(expr string) float64 {
		got, err := EvalExprWithRng(expr, nil, rand.New(rand.NewSource(7)))
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	rng := rand.New(rand.NewSource(7))
	first, second := rng.Float64(), rng.Float64()
	if got := draws("random() - random() * 10"); got != first-second*10 {
		t.Errorf("draw order: got %v, want %v", got, first-second*10)
	}
}

func TestExprRejectsMalformedInput(t *testing.T) {
	for _, expr := range []string{"1 +", "(1 + 2", "1 2", "1 < 2 < 3", "var(1)", "var(a", "round()", "min(1)", "random(1)", "nope(1)", "1 = 2", "$", ""} {
		if _, err := EvalExprWithLookupAndRng(expr, nil, func(string) (float64, error) { return 0, nil }, rand.New(rand.NewSource(1))); err == nil {
			t.Errorf("%q evaluated without error", expr)
		}
	}
}

func TestValidateExprVisitsReferencesInOrderWithoutEvaluating(t *testing.T) {
	var visited []string
	if err := ValidateExpr("var(a) / 0 + known + var(b) + random()", map[string]float64{"known": 1}, func(name string) { visited = append(visited, name) }); err != nil {
		t.Fatal(err)
	}
	if want := []string{"a", "b"}; !reflect.DeepEqual(visited, want) {
		t.Errorf("visited %v, want %v", visited, want)
	}
	if err := ValidateExpr("var(a) + unknown", nil, nil); err == nil {
		t.Error("an unknown plain variable validated")
	}
}

func TestDistanceCallMeasuresNamedThingsInSpaceOrZoneSteps(t *testing.T) {
	measure := func(a string, b string, zone_steps bool) (float64, error) {
		if zone_steps {
			return 10, nil
		}
		return float64(len(a) + len(b)), nil
	}
	cases := map[string]float64{"distance(ab, c)": 3, "distance(ab, c, 0)": 3, "distance(ab, c, 1) + 1": 11, "distance(a, b, 2 - 1) * 2": 20}
	for expr, want := range cases {
		if got, err := EvalExprWithDistance(expr, nil, nil, measure, nil); err != nil || got != want {
			t.Errorf("%q = %v, %v; want %v", expr, got, err, want)
		}
	}
	for _, expr := range []string{"distance(a)", "distance(1, b)", "distance(a b)", "distance(a, b"} {
		if _, err := EvalExprWithDistance(expr, nil, nil, measure, nil); err == nil {
			t.Errorf("%q evaluated without error", expr)
		}
	}
	if _, err := EvalExpr("distance(a, b)", nil); err == nil {
		t.Error("distance evaluated with no measure")
	}
	var visited []string
	if err := ValidateExpr("distance(a, b) + var(c)", nil, func(name string) { visited = append(visited, name) }); err != nil || !reflect.DeepEqual(visited, []string{"a.x", "b.x", "c"}) {
		t.Errorf("visited %v, %v", visited, err)
	}
}
