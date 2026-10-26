package ai

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"

	"github.com/dgray001/gray_online/util"
)

// setVarAction stores a number for later rules to read with var(name): in the per-turn map (cleared
// every turn) or, with "persist", in the script's global map that survives between turns.
type setVarAction struct {
	name    string
	value   amount
	persist bool
}

// Load-time bookkeeping for var() names: every name read and every name set_var writes
var (
	var_names_mu  sync.Mutex
	var_names_log *varNameLog
)

type varNameLog struct {
	read   map[string]bool
	set    map[string]bool
	random bool
	// the for_each loops enclosing the action being parsed, innermost last, and what is wrong with the item fields read so far
	scopes []loopScope
	errors []string
}

type loopScope struct {
	name    string
	source  string
	extra   []string
	barrier string
}

func withScopeBarrier(label string, parse func() error) error {
	if var_names_log == nil {
		return parse()
	}
	var_names_log.scopes = append(var_names_log.scopes, loopScope{barrier: label})
	defer func() { var_names_log.scopes = var_names_log.scopes[:len(var_names_log.scopes)-1] }()
	return parse()
}

func (l *varNameLog) hasRoleInScope(role string) bool {
	for i := len(l.scopes) - 1; i >= 0; i-- {
		if l.scopes[i].barrier != "" {
			return false
		}
		if l.scopes[i].name == role {
			return true
		}
	}
	return false
}

func noteVarRead(name string) {
	if var_names_log == nil {
		return
	}
	if ref, isConfig := strings.CutPrefix(name, util.ConfigRefPrefix); isConfig {
		var_names_log.noteConfigRead(ref)
		return
	}
	if strings.Contains(name, ".") {
		var_names_log.noteItemRead(name)
		return
	}
	var_names_log.read[name] = true
}

// A name.field read must sit inside a for_each named name, over a source whose items have that field
func (l *varNameLog) noteItemRead(name string) {
	prefix, field, _ := strings.Cut(name, ".")
	for i := len(l.scopes) - 1; i >= 0; i-- {
		if l.scopes[i].barrier != "" {
			l.errors = append(l.errors, fmt.Sprintf("var(%s): %s is resolved once, before any item, so it cannot read %q; use a let with an aggregate for per-item values", name, l.scopes[i].barrier, prefix))
			return
		}
		if l.scopes[i].name != prefix {
			continue
		}
		if (field == "distance_to_actor" || field == "zone_distance_to_actor") && !l.hasRoleInScope("actor") {
			l.errors = append(l.errors, fmt.Sprintf("var(%s): %s items have no field %q", name, l.scopes[i].source, field))
			return
		}
		if (field == "distance_to_target" || field == "zone_distance_to_target") && !l.hasRoleInScope("target") {
			l.errors = append(l.errors, fmt.Sprintf("var(%s): %s items have no field %q", name, l.scopes[i].source, field))
			return
		}
		if !hasItemField(l.scopes[i].source, field) && !slices.Contains(l.scopes[i].extra, field) {
			l.errors = append(l.errors, fmt.Sprintf("var(%s): %s items have no field %q", name, l.scopes[i].source, field))
		}
		return
	}
	l.errors = append(l.errors, fmt.Sprintf("var(%s) is read outside a for_each named %q", name, prefix))
}

// Parses a for_each body with its item name in scope; without a name log (a parse outside a model load) nothing is checked
func withLoopScope(name string, source string, parse func() error) error {
	return withLoopScopeExtra(name, source, nil, parse)
}

func withLoopScopeExtra(name string, source string, extra []string, parse func() error) error {
	if var_names_log == nil {
		return parse()
	}
	var_names_log.scopes = append(var_names_log.scopes, loopScope{name: name, source: source, extra: extra})
	defer func() { var_names_log.scopes = var_names_log.scopes[:len(var_names_log.scopes)-1] }()
	return parse()
}

func noteVarSet(name string) {
	if var_names_log != nil {
		var_names_log.set[name] = true
	}
}

// Runs parse with var() bookkeeping on, then reports names no set_var ever sets
func withVarNameCheck(diagnostics io.Writer, parse func() error) error {
	var_names_mu.Lock()
	defer var_names_mu.Unlock()
	var_names_log = &varNameLog{read: map[string]bool{}, set: map[string]bool{}}
	defer func() { var_names_log = nil }()
	if err := parse(); err != nil {
		return err
	}
	if len(var_names_log.errors) > 0 {
		return fmt.Errorf("%s", strings.Join(var_names_log.errors, "; "))
	}
	for name := range var_names_log.read {
		if !var_names_log.set[name] {
			fmt.Fprintf(diagnostics, "ai config: var(%s) is never set by a set_var; it will read 0\n", name)
		}
	}
	return nil
}

func noteRandomUse() {
	if var_names_log != nil {
		var_names_log.random = true
	}
}

func trackRandom(parse func() (counter, error)) (counter, bool, error) {
	if var_names_log == nil {
		query, err := parse()
		return query, false, err
	}
	outer := var_names_log.random
	var_names_log.random = false
	query, err := parse()
	used := var_names_log.random
	var_names_log.random = outer || used
	return query, used, err
}

func (a *setVarAction) ToOrders(view View, internals *Internals) []Order {
	previous := internals.eval_errors
	v := a.value.float(view, internals)
	if internals.eval_errors != previous {
		return nil
	}
	if a.persist {
		if internals.vars == nil {
			internals.vars = make(map[string]float64)
		}
		internals.vars[a.name] = v
	} else {
		if internals.turn_vars == nil {
			internals.turn_vars = make(map[string]float64)
		}
		internals.turn_vars[a.name] = v
	}
	return nil
}

func parseSetVar(raw map[string]any) (Action, error) {
	if err := checkActionKeys(raw, "name", "value", "persist"); err != nil {
		return nil, fmt.Errorf("set_var: %w", err)
	}
	if value, present := raw["persist"]; present {
		if _, ok := value.(bool); !ok {
			return nil, fmt.Errorf("\"persist\" must be a boolean")
		}
	}
	name, ok := raw["name"].(string)
	if !ok || !validVarName(name) {
		return nil, fmt.Errorf("set_var requires a \"name\" of letters, digits and underscores")
	}
	value, err := parseAmount(raw["value"])
	if err != nil {
		return nil, fmt.Errorf("set_var %q: \"value\" %v", name, err)
	}
	persist, _ := raw["persist"].(bool)
	noteVarSet(name)
	return &setVarAction{name: name, value: value, persist: persist}, nil
}

func validVarName(name string) bool {
	if name == "" {
		return false
	}
	for i, c := range name {
		letter := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if !letter && (i == 0 || c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// var(name) resolution: this turn's variables, then global ones; anything else is 0
func (i *Internals) lookupVar(name string) float64 {
	if strings.Contains(name, ".") {
		return i.itemField(name)
	}
	if v, ok := i.turn_vars[name]; ok {
		return v
	}
	return i.vars[name]
}

func (i *Internals) warnOnce(key string, message string) {
	if i.warned == nil {
		i.warned = make(map[string]bool)
	}
	if !i.warned[key] {
		i.warned[key] = true
		fmt.Fprintln(i.diagnostics, message)
	}
}

func boolNumber(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// "value" counter: compares any expression, so rules can branch on script variables
func parseValueCounter(obj map[string]any) (counter, error) {
	value, err := parseAmount(obj["value"])
	if err != nil {
		return nil, fmt.Errorf("\"value\" %v", err)
	}
	return func(v View, i *Internals) float64 { return value.float(v, i) }, nil
}
