package util

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"unicode"
)

type exprTokenKind int

const (
	exprTokNum exprTokenKind = iota
	exprTokIdent
	exprTokOp
	exprTokLParen
	exprTokRParen
	exprTokComma
	exprTokCompare
	exprTokEOF
)

type exprToken struct {
	kind exprTokenKind
	text string
}

func tokenizeExpr(s string) ([]exprToken, error) {
	toks := make([]exprToken, 0)
	i := 0
	for i < len(s) {
		c := rune(s[i])
		switch {
		case unicode.IsSpace(c):
			i++
		case c == '(':
			toks = append(toks, exprToken{exprTokLParen, "("})
			i++
		case c == ')':
			toks = append(toks, exprToken{exprTokRParen, ")"})
			i++
		case c == ',':
			toks = append(toks, exprToken{exprTokComma, ","})
			i++
		case strings.ContainsRune("+-*/", c):
			toks = append(toks, exprToken{exprTokOp, string(c)})
			i++
		case strings.ContainsRune("<>=!", c):
			end := i + 1
			if end < len(s) && s[end] == '=' {
				end++
			}
			op := s[i:end]
			if op == "=" || op == "!" {
				return nil, fmt.Errorf("invalid comparison operator %q", op)
			}
			toks = append(toks, exprToken{exprTokCompare, op})
			i = end
		case unicode.IsDigit(c) || c == '.':
			j := i
			for j < len(s) && (unicode.IsDigit(rune(s[j])) || s[j] == '.') {
				j++
			}
			toks = append(toks, exprToken{exprTokNum, s[i:j]})
			i = j
		case unicode.IsLetter(c) || c == '_':
			j := i
			for j < len(s) && (unicode.IsLetter(rune(s[j])) || unicode.IsDigit(rune(s[j])) || s[j] == '_' || s[j] == '.') {
				j++
			}
			toks = append(toks, exprToken{exprTokIdent, s[i:j]})
			i = j
		default:
			return nil, fmt.Errorf("unexpected character %q in expression %q", c, s)
		}
	}
	toks = append(toks, exprToken{exprTokEOF, ""})
	return toks, nil
}

func EvalExpr(s string, vars map[string]float64) (float64, error) {
	return EvalExprWithLookup(s, vars, nil)
}

// Like EvalExpr, but var(name) is resolved through lookup
func EvalExprWithLookup(s string, vars map[string]float64, lookup func(name string) (float64, error)) (float64, error) {
	return evalExpr(s, &exprEnv{vars: vars, lookup: lookup})
}

func EvalExprWithLookupAndRng(s string, vars map[string]float64, lookup func(name string) (float64, error), rng *rand.Rand) (float64, error) {
	return EvalExprWithDistance(s, vars, lookup, nil, rng)
}

// Like EvalExprWithLookupAndRng, but distance(a, b) is measured through distance
func EvalExprWithDistance(s string, vars map[string]float64, lookup func(name string) (float64, error), distance DistanceFunc, rng *rand.Rand) (float64, error) {
	return EvalExprWithConfig(s, vars, lookup, distance, nil, rng)
}

func EvalExprWithConfig(s string, vars map[string]float64, lookup func(name string) (float64, error), distance DistanceFunc, config ConfigFunc, rng *rand.Rand) (float64, error) {
	return EvalExprWithFunctions(s, vars, lookup, distance, config, nil, nil, nil, rng)
}

func EvalExprWithFunctions(s string, vars map[string]float64, lookup func(name string) (float64, error), distance DistanceFunc, config ConfigFunc, game_var GameVarFunc, researched ResearchedFunc, membership MembershipFunc, rng *rand.Rand) (float64, error) {
	return evalExpr(s, &exprEnv{vars: vars, lookup: lookup, distance: distance, config: config, game_var: game_var, researched: researched, membership: membership, rng: rng})
}

// Like EvalExpr, but random() and randomInt(a, b) draw from rng
func EvalExprWithRng(s string, vars map[string]float64, rng *rand.Rand) (float64, error) {
	return evalExpr(s, &exprEnv{vars: vars, rng: rng})
}

var exprTokenCache sync.Map // expression text -> read-only []exprToken

func cachedTokens(s string) ([]exprToken, error) {
	if toks, ok := exprTokenCache.Load(s); ok {
		return toks.([]exprToken), nil
	}
	toks, err := tokenizeExpr(s)
	if err == nil {
		exprTokenCache.Store(s, toks)
	}
	return toks, err
}
