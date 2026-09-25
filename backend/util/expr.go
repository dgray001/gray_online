package util

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type exprTokenKind int

const (
	exprTokNum exprTokenKind = iota
	exprTokIdent
	exprTokOp
	exprTokLParen
	exprTokRParen
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
		case strings.ContainsRune("+-*/", c):
			toks = append(toks, exprToken{exprTokOp, string(c)})
			i++
		case unicode.IsDigit(c) || c == '.':
			j := i
			for j < len(s) && (unicode.IsDigit(rune(s[j])) || s[j] == '.') {
				j++
			}
			toks = append(toks, exprToken{exprTokNum, s[i:j]})
			i = j
		case unicode.IsLetter(c) || c == '_':
			j := i
			for j < len(s) && (unicode.IsLetter(rune(s[j])) || unicode.IsDigit(rune(s[j])) || s[j] == '_') {
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

type exprParser struct {
	toks []exprToken
	pos  int
	vars map[string]float64
}

func (p *exprParser) peek() exprToken {
	return p.toks[p.pos]
}

func (p *exprParser) next() exprToken {
	t := p.toks[p.pos]
	p.pos++
	return t
}

func (p *exprParser) parseExpr() (float64, error) {
	v, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for p.peek().kind == exprTokOp && (p.peek().text == "+" || p.peek().text == "-") {
		op := p.next().text
		rhs, err := p.parseTerm()
		if err != nil {
			return 0, err
		}
		if op == "+" {
			v += rhs
		} else {
			v -= rhs
		}
	}
	return v, nil
}

func (p *exprParser) parseTerm() (float64, error) {
	v, err := p.parseUnary()
	if err != nil {
		return 0, err
	}
	for p.peek().kind == exprTokOp && (p.peek().text == "*" || p.peek().text == "/") {
		op := p.next().text
		rhs, err := p.parseUnary()
		if err != nil {
			return 0, err
		}
		if op == "*" {
			v *= rhs
		} else {
			if rhs == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			v /= rhs
		}
	}
	return v, nil
}

func (p *exprParser) parseUnary() (float64, error) {
	if p.peek().kind == exprTokOp && p.peek().text == "-" {
		p.next()
		v, err := p.parseUnary()
		return -v, err
	}
	if p.peek().kind == exprTokOp && p.peek().text == "+" {
		p.next()
		return p.parseUnary()
	}
	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() (float64, error) {
	t := p.next()
	switch t.kind {
	case exprTokNum:
		return strconv.ParseFloat(t.text, 64)
	case exprTokIdent:
		v, ok := p.vars[t.text]
		if !ok {
			return 0, fmt.Errorf("unknown variable %q", t.text)
		}
		return v, nil
	case exprTokLParen:
		v, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		if p.next().kind != exprTokRParen {
			return 0, fmt.Errorf("expected closing paren")
		}
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected token %q", t.text)
	}
}

func EvalExpr(s string, vars map[string]float64) (float64, error) {
	toks, err := tokenizeExpr(s)
	if err != nil {
		return 0, err
	}
	p := &exprParser{toks: toks, vars: vars}
	v, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	if p.peek().kind != exprTokEOF {
		return 0, fmt.Errorf("unexpected trailing input in expression %q", s)
	}
	return v, nil
}
