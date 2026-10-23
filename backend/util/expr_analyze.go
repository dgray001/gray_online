package util

import "strings"

// One `left == right` factor of a condition; both sides are expression text
type EqualityFactor struct {
	Left, Right string
}

// What an expression reads: var(name) names, plain identifiers, and whether it draws random numbers
type ExprReads struct {
	Vars   []string
	Bare   []string
	Impure bool
}

func tokenTexts(toks []exprToken) string {
	texts := make([]string, len(toks))
	for i, t := range toks {
		texts[i] = t.text
	}
	return strings.Join(texts, " ")
}

// Indexes of tokens at parenthesis depth zero with the given kind
func topLevel(toks []exprToken, kind exprTokenKind) []int {
	depth, found := 0, []int{}
	for i, t := range toks {
		switch {
		case t.kind == exprTokLParen:
			depth++
		case t.kind == exprTokRParen:
			depth--
		case depth == 0 && t.kind == kind:
			found = append(found, i)
		}
	}
	return found
}

func stripOuterParens(toks []exprToken) []exprToken {
	for len(toks) >= 2 && toks[0].kind == exprTokLParen && toks[len(toks)-1].kind == exprTokRParen {
		depth := 0
		for i, t := range toks {
			if t.kind == exprTokLParen {
				depth++
			} else if t.kind == exprTokRParen {
				depth--
			}
			if depth == 0 && i < len(toks)-1 {
				return toks
			}
		}
		toks = toks[1 : len(toks)-1]
	}
	return toks
}

func equalityOf(toks []exprToken) (EqualityFactor, bool) {
	toks = stripOuterParens(toks)
	compares := topLevel(toks, exprTokCompare)
	if len(compares) != 1 || toks[compares[0]].text != "==" || compares[0] == 0 || compares[0] == len(toks)-1 {
		return EqualityFactor{}, false
	}
	at := compares[0]
	return EqualityFactor{Left: tokenTexts(toks[:at]), Right: tokenTexts(toks[at+1:])}, true
}

// The `==` factors that must all hold for the expression to be positive: the expression itself, or each
// parenthesized factor of a product of comparisons. Anything else (sums, quotients) yields none.
func EqualityFactors(expr string) []EqualityFactor {
	toks, err := cachedTokens(expr)
	if err != nil || len(toks) < 2 {
		return nil
	}
	toks = toks[:len(toks)-1]
	if len(topLevel(toks, exprTokCompare)) > 0 {
		if factor, ok := equalityOf(toks); ok {
			return []EqualityFactor{factor}
		}
		return nil
	}
	for _, i := range topLevel(toks, exprTokOp) {
		if toks[i].text != "*" {
			return nil
		}
	}
	factors, start := []EqualityFactor{}, 0
	cuts := append(topLevel(toks, exprTokOp), len(toks))
	for _, cut := range cuts {
		if cut > start {
			if factor, ok := equalityOf(toks[start:cut]); ok {
				factors = append(factors, factor)
			}
		}
		start = cut + 1
	}
	return factors
}

func AnalyzeExpr(expr string) (ExprReads, bool) {
	toks, err := cachedTokens(expr)
	if err != nil {
		return ExprReads{}, false
	}
	reads := ExprReads{}
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if t.kind != exprTokIdent {
			continue
		}
		if toks[i+1].kind != exprTokLParen {
			reads.Bare = append(reads.Bare, t.text)
			continue
		}
		switch t.text {
		case "var":
			if i+2 < len(toks) && toks[i+2].kind == exprTokIdent {
				reads.Vars = append(reads.Vars, toks[i+2].text)
				i += 2
			}
		case "distance":
			i += 4 // the two item names are not variables
		case "random", "randomInt":
			reads.Impure = true
		}
	}
	return reads, true
}
