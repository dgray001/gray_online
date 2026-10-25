package ai

import (
	"fmt"
	"slices"
)

type matchSpec struct {
	where  Condition
	weight *amount
	score  []amount
}

type selectAction struct {
	pick, actors *selectorSpec
	verbs        []verb
	match        *matchSpec
	into         *selectSink
	queue        bool
	prepend      bool
	assign       string
	carry        string
}

var selectKeys = []string{"action", "pick", "actors", "assign", "match", "issue", "into", "mode"}

func parseSelect(raw map[string]any) (Action, error) {
	if raw["assign"] != nil {
		return nil, fmt.Errorf("select: assign was replaced by pick.capacity and match")
	}
	if err := checkKeys(raw, selectKeys...); err != nil {
		return nil, fmt.Errorf("select: %v", err)
	}
	if err := checkOptionTypes[map[string]any](raw, "pick", "actors", "assign", "match", "into"); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	if err := checkOptionTypes[[]any](raw, "issue"); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	if err := checkOptionTypes[string](raw, "mode"); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	if carry, isCarry, err := parseCarry(raw); isCarry {
		if err != nil {
			return nil, err
		}
		return &selectAction{carry: carry}, nil
	}
	a := &selectAction{}
	var err error
	var assigning bool
	if a.assign, assigning, err = parseAssignInto(raw); err != nil {
		return nil, err
	}
	if raw["pick"] != nil {
		if a.pick, err = parseSelectorSpecFor(raw["pick"], "target", assigning); err != nil {
			return nil, fmt.Errorf("select pick: %v", err)
		}
	}
	if raw["actors"] != nil {
		if a.actors, err = parseSelectorSpec(raw["actors"], "actor"); err != nil {
			return nil, fmt.Errorf("select actors: %v", err)
		}
	}
	if raw["issue"] != nil {
		actor_kind := "buildings"
		if a.actors != nil {
			actor_kind = a.actors.kind
		}
		parseIssue := func() error {
			return withLoopScopeExtra("actor", "select_actor_"+actor_kind, a.actorScopeNames(), func() (parsed error) {
				a.verbs, parsed = parseVerbs(raw["issue"], actor_kind)
				return parsed
			})
		}
		if a.pick != nil {
			err = withLoopScopeExtra("target", "select_target_"+a.pick.kind, a.pick.letNames(), parseIssue)
		} else {
			err = parseIssue()
		}
		if err != nil {
			return nil, fmt.Errorf("select: %v", err)
		}
	}
	if raw["into"] != nil && !assigning {
		if a.into, err = parseSink(raw["into"], a.pick); err != nil {
			return nil, fmt.Errorf("select into: %v", err)
		}
	}
	if a.pick != nil && (a.pick.kind == "regions" || a.pick.kind == "assignments") && len(a.verbs) > 0 {
		return nil, fmt.Errorf("select: %s cannot be the target of a verb", a.pick.kind)
	}
	if err = a.parseQueue(raw["mode"]); err != nil {
		return nil, err
	}
	if err = a.parseActors(raw); err != nil {
		return nil, err
	}
	if assigning {
		if err = a.checkAssignShape(); err != nil {
			return nil, err
		}
	}
	if a.pick != nil && a.pick.capacity != nil && (!a.needsTarget() || a.into != nil || slices.ContainsFunc(a.verbs, verb.actorless)) {
		return nil, fmt.Errorf("select: capacity is only valid on pick selectors with target-dependent verbs")
	}
	if a.pick == nil && (a.into != nil || a.needsTarget()) {
		return nil, fmt.Errorf("select needs a \"pick\"")
	}
	if a.pick == nil && len(a.verbs) == 0 {
		return nil, fmt.Errorf("select needs a \"pick\" or an \"issue\"")
	}
	return a, nil
}

func (a *selectAction) parseQueue(raw any) error {
	switch raw {
	case nil, "replace":
	case "append":
		a.queue = true
	case "prepend":
		a.prepend = true
	default:
		return fmt.Errorf("select: mode must be \"replace\", \"append\" or \"prepend\"")
	}
	return nil
}

func (a *selectAction) hiring() bool {
	return len(a.verbs) > 0 && a.verbs[0].name == "hire"
}

func (a *selectAction) cancelling() bool {
	return len(a.verbs) > 0 && a.verbs[0].name == "cancel_foundation"
}

func (a *selectAction) needsTarget() bool {
	for _, v := range a.verbs {
		if v.name != "delete" && v.name != "ungarrison" && !v.production() && !v.control() {
			return true
		}
	}
	return false
}

func (a *selectAction) parseActors(raw map[string]any) error {
	if len(a.verbs) == 0 {
		if raw["actors"] != nil || raw["match"] != nil || raw["mode"] != nil {
			return fmt.Errorf("select: \"actors\", \"match\" and \"mode\" need an \"issue\"")
		}
		return nil
	}
	if slices.ContainsFunc(a.verbs, verb.actorless) {
		if len(a.verbs) > 1 || raw["actors"] != nil || raw["match"] != nil || raw["mode"] != nil {
			return fmt.Errorf("select: hire and cancel_foundation must be the only verb and take no \"actors\", \"match\" or \"mode\"")
		}
		kind := "foundations"
		if a.hiring() {
			kind = "zones"
		}
		if a.pick == nil || a.pick.kind != kind {
			return fmt.Errorf("select: %s needs a %q pick", a.verbs[0].name, kind)
		}
		return nil
	}
	if a.actors != nil && a.actors.kind != "units" && a.actors.kind != "buildings" {
		return fmt.Errorf("select: actors must be units or buildings")
	}
	if slices.ContainsFunc(a.verbs, verb.production) {
		return a.parseProduction(raw)
	}
	if slices.ContainsFunc(a.verbs, verb.control) {
		return a.parseControl(raw)
	}
	if a.actors == nil && slices.ContainsFunc(a.verbs, func(v verb) bool { return v.name == "delete" }) {
		return fmt.Errorf("select: \"delete\" needs explicit \"actors\"")
	}
	if a.actors == nil {
		default_actors := map[string]any{"kind": "units", "where": map[string]any{"if": "var(actor.idle)"}}
		var err error
		if a.actors, err = parseSelectorSpec(default_actors, "actor"); err != nil {
			return fmt.Errorf("select actors: %v", err)
		}
	}
	if a.actors.kind == "buildings" && raw["mode"] != nil {
		return fmt.Errorf("select: building orders take no \"mode\"")
	}
	return a.parseMatchClause(raw)
}

func (a *selectAction) parseMatchClause(raw map[string]any) error {
	if raw["match"] == nil {
		return nil
	}
	if !a.needsTarget() {
		return fmt.Errorf("select: \"match\" needs a target-dependent verb")
	}
	if a.pick == nil {
		return fmt.Errorf("select: match requires both actors and pick")
	}
	extra := append(a.pick.letNames(), "assigned", "remaining", "already_targeted")
	err := withLoopScopeExtra("actor", "select_actor_"+a.actors.kind, a.actorScopeNames(), func() error {
		return withLoopScopeExtra("target", "select_target_"+a.pick.kind, extra, func() error {
			return a.parseMatchInner(raw["match"])
		})
	})
	if err != nil {
		return fmt.Errorf("select match: %v", err)
	}
	return nil
}

type matchWhere struct {
	conditions []amount
}

func (m *matchWhere) Evaluate(view View, internals *Internals) bool {
	for _, c := range m.conditions {
		if c.float(view, internals) <= 0 {
			return false
		}
	}
	return true
}

var matchKeys = []string{"where", "weight", "score"}

func (a *selectAction) parseMatchInner(raw any) error {
	obj, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("must be an object")
	}
	if err := checkKeys(obj, matchKeys...); err != nil {
		return err
	}
	m := &matchSpec{}
	if err := m.parseWhere(obj["where"]); err != nil {
		return err
	}
	if err := m.parseWeight(obj["weight"]); err != nil {
		return err
	}
	if err := m.parseScore(obj["score"]); err != nil {
		return err
	}
	a.match = m
	return nil
}

func (m *matchSpec) parseWhere(raw any) error {
	if raw == nil {
		m.where = &conditionAlways{}
		return nil
	}
	whereMap, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("where must be an object")
	}
	if err := checkKeys(whereMap, "if"); err != nil {
		return fmt.Errorf("where: %w", err)
	}
	if ifRaw, hasIf := whereMap["if"]; hasIf {
		return m.parseWhereIf(ifRaw)
	}
	m.where = &conditionAlways{}
	return nil
}

func (m *matchSpec) parseWhereIf(raw any) error {
	list, isList := raw.([]any)
	if !isList {
		list = []any{raw}
	}
	conds := make([]amount, 0, len(list))
	for _, entry := range list {
		c, err := parseAmount(entry)
		if err != nil {
			return fmt.Errorf("where if: %v", err)
		}
		if c.query != nil {
			return fmt.Errorf("match where cannot contain aggregates")
		}
		conds = append(conds, c)
	}
	m.where = &matchWhere{conditions: conds}
	return nil
}

func (m *matchSpec) parseWeight(raw any) error {
	if raw == nil {
		return nil
	}
	w, err := parseAmount(raw)
	if err != nil {
		return fmt.Errorf("weight: %v", err)
	}
	if w.query != nil {
		return fmt.Errorf("match weight cannot contain aggregates")
	}
	m.weight = &w
	return nil
}

func (m *matchSpec) parseScore(raw any) error {
	if raw == nil {
		return nil
	}
	list, isList := raw.([]any)
	if !isList {
		list = []any{raw}
	}
	for _, entry := range list {
		s, err := parseAmount(entry)
		if err != nil {
			return fmt.Errorf("score: %v", err)
		}
		if s.query != nil {
			return fmt.Errorf("match score cannot contain aggregates")
		}
		m.score = append(m.score, s)
	}
	return nil
}
