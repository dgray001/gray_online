package util

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"sync"
)

// Measures the distance between the things bound under two names, in space steps or zone steps
type DistanceFunc func(a string, b string, zone_steps bool) (float64, error)

type ConfigFunc func(table string, field string, id uint32) (float64, error)

type GameVarFunc func(field string) (float64, error)

type ResearchedFunc func(tech_id uint32) (bool, error)

const ConfigRefPrefix = "config:"

// What an evaluation reads from outside the expression
type exprEnv struct {
	vars       map[string]float64
	lookup     func(name string) (float64, error)
	distance   DistanceFunc
	config     ConfigFunc
	game_var   GameVarFunc
	researched ResearchedFunc
	rng        *rand.Rand
}

type exprNode func(env *exprEnv) (float64, error)

// A name an expression reads, in the order it appears
type exprRef struct {
	name        string
	is_var      bool
	is_item     bool // a thing named in distance(), which must have a location
	is_config   bool
	is_game_var bool
}

// An expression parsed once; evaluating it never re-reads the text
type exprProgram struct {
	run  exprNode
	refs []exprRef
}

type exprCompiler struct {
	toks []exprToken
	pos  int
	refs []exprRef
}

func (c *exprCompiler) peek() exprToken {
	return c.toks[c.pos]
}

func (c *exprCompiler) next() exprToken {
	t := c.toks[c.pos]
	c.pos++
	return t
}

func (c *exprCompiler) compileExpr() (exprNode, error) {
	left, err := c.compileSum()
	if err != nil || c.peek().kind != exprTokCompare {
		return left, err
	}
	op := c.next().text
	right, err := c.compileSum()
	if err != nil {
		return nil, err
	}
	return func(env *exprEnv) (float64, error) {
		l, err := left(env)
		if err != nil {
			return 0, err
		}
		r, err := right(env)
		if err != nil {
			return 0, err
		}
		if compareExpr(op, l, r) {
			return 1, nil
		}
		return 0, nil
	}, nil
}

func compareExpr(op string, left float64, right float64) bool {
	switch op {
	case "==":
		return left == right
	case "!=":
		return left != right
	case "<":
		return left < right
	case ">":
		return left > right
	case "<=":
		return left <= right
	case ">=":
		return left >= right
	}
	return false
}

// One chain of left-associative operators over operands, such as a + b - c
func (c *exprCompiler) compileChain(operators string, operand func() (exprNode, error), apply func(op string, left float64, right float64) (float64, error)) (exprNode, error) {
	first, err := operand()
	if err != nil {
		return nil, err
	}
	var ops []string
	var rest []exprNode
	for c.peek().kind == exprTokOp && (c.peek().text == string(operators[0]) || c.peek().text == string(operators[1])) {
		ops = append(ops, c.next().text)
		node, err := operand()
		if err != nil {
			return nil, err
		}
		rest = append(rest, node)
	}
	if len(ops) == 0 {
		return first, nil
	}
	return func(env *exprEnv) (float64, error) {
		v, err := first(env)
		if err != nil {
			return 0, err
		}
		for i, node := range rest {
			rhs, err := node(env)
			if err != nil {
				return 0, err
			}
			if v, err = apply(ops[i], v, rhs); err != nil {
				return 0, err
			}
		}
		return v, nil
	}, nil
}

func (c *exprCompiler) compileSum() (exprNode, error) {
	return c.compileChain("+-", c.compileTerm, func(op string, left float64, right float64) (float64, error) {
		if op == "+" {
			return left + right, nil
		}
		return left - right, nil
	})
}

func (c *exprCompiler) compileTerm() (exprNode, error) {
	return c.compileChain("*/", c.compileUnary, func(op string, left float64, right float64) (float64, error) {
		if op == "*" {
			return left * right, nil
		}
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	})
}

func (c *exprCompiler) compileUnary() (exprNode, error) {
	if c.peek().kind == exprTokOp && c.peek().text == "-" {
		c.next()
		operand, err := c.compileUnary()
		if err != nil {
			return nil, err
		}
		return func(env *exprEnv) (float64, error) {
			v, err := operand(env)
			return -v, err
		}, nil
	}
	if c.peek().kind == exprTokOp && c.peek().text == "+" {
		c.next()
		return c.compileUnary()
	}
	return c.compilePrimary()
}

func (c *exprCompiler) compilePrimary() (exprNode, error) {
	t := c.next()
	switch t.kind {
	case exprTokNum:
		value, err := strconv.ParseFloat(t.text, 64)
		if err != nil {
			return nil, err
		}
		return func(*exprEnv) (float64, error) { return value, nil }, nil
	case exprTokIdent:
		if c.peek().kind == exprTokLParen {
			return c.compileCall(t.text)
		}
		name := t.text
		c.refs = append(c.refs, exprRef{name: name})
		return func(env *exprEnv) (float64, error) {
			v, ok := env.vars[name]
			if !ok {
				return 0, fmt.Errorf("unknown variable %q", name)
			}
			return v, nil
		}, nil
	case exprTokLParen:
		inner, err := c.compileExpr()
		if err != nil {
			return nil, err
		}
		if c.next().kind != exprTokRParen {
			return nil, fmt.Errorf("expected closing paren")
		}
		return inner, nil
	}
	return nil, fmt.Errorf("unexpected token %q", t.text)
}

func (c *exprCompiler) compileVarCall() (exprNode, error) {
	arg := c.next()
	if arg.kind != exprTokIdent {
		return nil, fmt.Errorf("var() takes a bare name")
	}
	if c.next().kind != exprTokRParen {
		return nil, fmt.Errorf("expected closing paren after var(%s", arg.text)
	}
	name := arg.text
	c.refs = append(c.refs, exprRef{name: name, is_var: true})
	return func(env *exprEnv) (float64, error) {
		if env.lookup == nil {
			return 0, fmt.Errorf("var() is not available here")
		}
		return env.lookup(name)
	}, nil
}

// distance(a, b) in space steps, or distance(a, b, 1) in zone steps, between two named things
func (c *exprCompiler) compileDistanceCall() (exprNode, error) {
	names := [2]string{}
	for i := range names {
		arg := c.next()
		if arg.kind != exprTokIdent || (i == 0 && c.peek().kind != exprTokComma) {
			return nil, fmt.Errorf("distance() takes two bare names")
		}
		names[i] = arg.text
		if i == 0 {
			c.next() // ,
		}
	}
	zone_steps := func(*exprEnv) (float64, error) { return 0, nil }
	if c.peek().kind == exprTokComma {
		c.next()
		flag, err := c.compileExpr()
		if err != nil {
			return nil, err
		}
		zone_steps = flag
	}
	if c.next().kind != exprTokRParen {
		return nil, fmt.Errorf("expected closing paren after distance(%s, %s", names[0], names[1])
	}
	c.refs = append(c.refs, exprRef{name: names[0], is_item: true}, exprRef{name: names[1], is_item: true})
	return func(env *exprEnv) (float64, error) {
		if env.distance == nil {
			return 0, fmt.Errorf("distance() is not available here")
		}
		flag, err := zone_steps(env)
		if err != nil {
			return 0, err
		}
		return env.distance(names[0], names[1], flag != 0)
	}, nil
}

func (c *exprCompiler) compileConfigCall() (exprNode, error) {
	names := [2]string{}
	for i := range names {
		arg := c.next()
		if arg.kind != exprTokIdent || c.next().kind != exprTokComma {
			return nil, fmt.Errorf("config() takes a table name, a field name and an id")
		}
		names[i] = arg.text
	}
	id, err := c.compileExpr()
	if err != nil {
		return nil, err
	}
	if c.next().kind != exprTokRParen {
		return nil, fmt.Errorf("expected closing paren after config(%s, %s", names[0], names[1])
	}
	c.refs = append(c.refs, exprRef{name: names[0] + "." + names[1], is_config: true})
	return func(env *exprEnv) (float64, error) {
		if env.config == nil {
			return 0, fmt.Errorf("config() is not available here")
		}
		value, err := id(env)
		if err != nil {
			return 0, err
		}
		rounded := math.Round(value)
		if math.IsNaN(rounded) || rounded < 0 || rounded > math.MaxUint32 {
			return 0, fmt.Errorf("config(%s, %s, %v): id out of range", names[0], names[1], value)
		}
		return env.config(names[0], names[1], uint32(rounded))
	}, nil
}

var gameVarFields = map[string]bool{
	"food": true, "wood": true, "stone": true, "gold": true,
	"population": true, "population_limit": true, "population_max": true,
	"turn": true, "map_size": true, "num_players": true,
}

func (c *exprCompiler) compileGameVarCall() (exprNode, error) {
	arg := c.next()
	if arg.kind != exprTokIdent {
		return nil, fmt.Errorf("gameVar() takes a field name")
	}
	if !gameVarFields[arg.text] {
		return nil, fmt.Errorf("unknown gameVar field %q", arg.text)
	}
	if c.next().kind != exprTokRParen {
		return nil, fmt.Errorf("expected closing paren after gameVar(%s", arg.text)
	}
	c.refs = append(c.refs, exprRef{name: arg.text, is_game_var: true})
	field := arg.text
	return func(env *exprEnv) (float64, error) {
		if env.game_var == nil {
			return 0, fmt.Errorf("gameVar() is not available here")
		}
		return env.game_var(field)
	}, nil
}

func (c *exprCompiler) compileResearchedCall() (exprNode, error) {
	id, err := c.compileExpr()
	if err != nil {
		return nil, err
	}
	if c.next().kind != exprTokRParen {
		return nil, fmt.Errorf("expected closing paren after researched(...)")
	}
	return func(env *exprEnv) (float64, error) {
		if env.researched == nil {
			return 0, fmt.Errorf("researched() is not available here")
		}
		value, err := id(env)
		if err != nil {
			return 0, err
		}
		rounded := math.Round(value)
		if math.IsNaN(rounded) || rounded < 0 || rounded > math.MaxUint32 {
			return 0, fmt.Errorf("researched(%v): id out of range", value)
		}
		researched, err := env.researched(uint32(rounded))
		if err != nil || !researched {
			return 0, err
		}
		return 1, nil
	}, nil
}

// Supports round, random, randomInt, min and max; var(name) is resolved through the lookup
func (c *exprCompiler) compileCall(name string) (exprNode, error) {
	c.next() // (
	if name == "var" {
		return c.compileVarCall()
	}
	if name == "distance" {
		return c.compileDistanceCall()
	}
	if name == "config" {
		return c.compileConfigCall()
	}
	if name == "gameVar" {
		return c.compileGameVarCall()
	}
	if name == "researched" {
		return c.compileResearchedCall()
	}
	var args []exprNode
	if c.peek().kind == exprTokRParen {
		c.next()
	} else {
		for {
			arg, err := c.compileExpr()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			t := c.next()
			if t.kind == exprTokRParen {
				break
			}
			if t.kind != exprTokComma {
				return nil, fmt.Errorf("expected ',' or ')' in %s()", name)
			}
		}
	}
	apply, err := functionOf(name, len(args))
	if err != nil {
		return nil, err
	}
	return func(env *exprEnv) (float64, error) {
		values := make([]float64, len(args))
		for i, arg := range args {
			v, err := arg(env)
			if err != nil {
				return 0, err
			}
			values[i] = v
		}
		return apply(env, values)
	}, nil
}

func functionOf(name string, arity int) (func(env *exprEnv, args []float64) (float64, error), error) {
	switch name {
	case "round":
		if arity != 1 {
			return nil, fmt.Errorf("round() needs exactly one argument")
		}
		return func(_ *exprEnv, args []float64) (float64, error) { return math.Round(args[0]), nil }, nil
	case "random":
		if arity != 0 {
			return nil, fmt.Errorf("random() takes no arguments")
		}
		return func(env *exprEnv, _ []float64) (float64, error) {
			if env.rng == nil {
				return 0, fmt.Errorf("random() is not available here")
			}
			return env.rng.Float64(), nil
		}, nil
	case "randomInt":
		if arity != 2 {
			return nil, fmt.Errorf("randomInt() needs a min and a max")
		}
		return func(env *exprEnv, args []float64) (float64, error) {
			if env.rng == nil {
				return 0, fmt.Errorf("randomInt() is not available here")
			}
			return float64(RandomIntFrom(env.rng, int(math.Round(args[0])), int(math.Round(args[1])))), nil
		}, nil
	case "min", "max":
		if arity < 2 {
			return nil, fmt.Errorf("%s() needs at least two arguments", name)
		}
		return func(_ *exprEnv, args []float64) (float64, error) {
			v := args[0]
			for _, a := range args[1:] {
				if (name == "min" && a < v) || (name == "max" && a > v) {
					v = a
				}
			}
			return v, nil
		}, nil
	}
	return nil, fmt.Errorf("unknown function %q", name)
}

var exprProgramCache sync.Map // expression text -> read-only *exprProgram

func compiledExpr(s string) (*exprProgram, error) {
	if program, ok := exprProgramCache.Load(s); ok {
		return program.(*exprProgram), nil
	}
	toks, err := cachedTokens(s)
	if err != nil {
		return nil, err
	}
	c := &exprCompiler{toks: toks}
	run, err := c.compileExpr()
	if err != nil {
		return nil, err
	}
	if c.peek().kind != exprTokEOF {
		return nil, fmt.Errorf("unexpected trailing input in expression %q", s)
	}
	program := &exprProgram{run: run, refs: c.refs}
	exprProgramCache.Store(s, program)
	return program, nil
}

func evalExpr(s string, env *exprEnv) (float64, error) {
	program, err := compiledExpr(s)
	if err != nil {
		return 0, err
	}
	return program.run(env)
}

// ValidateExpr checks syntax and visits var() references without evaluating arithmetic or drawing randomness.
func ValidateExpr(s string, vars map[string]float64, visit func(name string)) error {
	program, err := compiledExpr(s)
	if err != nil {
		return err
	}
	for _, ref := range program.refs {
		if ref.is_config {
			if visit != nil {
				visit(ConfigRefPrefix + ref.name)
			}
		} else if ref.is_item {
			if visit != nil {
				visit(ref.name + ".x")
			}
		} else if ref.is_var {
			if visit != nil {
				visit(ref.name)
			}
		} else if ref.is_game_var {
			continue
		} else if _, ok := vars[ref.name]; !ok {
			return fmt.Errorf("unknown variable %q", ref.name)
		}
	}
	return nil
}
