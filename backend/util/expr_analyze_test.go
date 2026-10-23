package util

import (
	"reflect"
	"testing"
)

func TestEqualityFactors(t *testing.T) {
	cases := map[string][]EqualityFactor{
		"var(item.a) == var(target.b)":                                         {{"var ( item.a )", "var ( target.b )"}},
		"(var(item.a) == var(t.b)) * (food >= 50) * (var(item.c) == 0)":        {{"var ( item.a )", "var ( t.b )"}, {"var ( item.c )", "0"}},
		"((var(item.a) == 1))":                                                 {{"var ( item.a )", "1"}},
		"(var(item.a) == 1) + (var(item.b) == 2)":                              nil,
		"(var(item.a) == 1) / 2":                                               nil,
		"var(item.a) * 2 == var(t.b)":                                          {{"var ( item.a ) * 2", "var ( t.b )"}},
		"var(item.a) == 1 == 2":                                                nil,
		"var(item.a) < 1":                                                      nil,
		"1 - var(item.a)":                                                      nil,
		"(var(item.a) == var(t.b)) * (var(x.y) == 2) * (var(item.p) > 1)":      {{"var ( item.a )", "var ( t.b )"}, {"var ( x.y )", "2"}},
		"max(var(item.a), 2) == var(t.b)":                                      {{"max ( var ( item.a ) , 2 )", "var ( t.b )"}},
		"-(var(item.a) == 1)":                                                  nil,
		"(var(item.a) == var(t.b)) * (var(item.c) == var(t.d) + 1) * food > 0": nil,
	}
	for expr, want := range cases {
		if got := EqualityFactors(expr); !reflect.DeepEqual(got, want) && !(len(got) == 0 && len(want) == 0) {
			t.Errorf("EqualityFactors(%q) = %v, want %v", expr, got, want)
		}
	}
}

func TestAnalyzeExpr(t *testing.T) {
	reads, ok := AnalyzeExpr("var(item.a) + food * max(var(turn_flag), random())")
	if !ok || !reflect.DeepEqual(reads.Vars, []string{"item.a", "turn_flag"}) || !reflect.DeepEqual(reads.Bare, []string{"food"}) || !reads.Impure {
		t.Errorf("AnalyzeExpr = %+v, %v", reads, ok)
	}
	if reads, _ := AnalyzeExpr("var(item.a) + randomInt(1, 2)"); !reads.Impure {
		t.Error("randomInt must be impure")
	}
}
