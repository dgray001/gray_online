package ai

import (
	"fmt"
	"slices"
)

type selectAction struct {
	pick, actors *selectorSpec
	verbs        []verb
	assign       *spreadConfig
	into         *selectSink
	queue        bool
	prepend      bool
}

var selectKeys = []string{"action", "pick", "actors", "assign", "issue", "into", "mode"}

func parseSelect(raw map[string]any) (Action, error) {
	if err := checkKeys(raw, selectKeys...); err != nil {
		return nil, fmt.Errorf("select: %v", err)
	}
	if err := checkOptionTypes[map[string]any](raw, "pick", "actors", "assign", "into"); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	if err := checkOptionTypes[[]any](raw, "issue"); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	if err := checkOptionTypes[string](raw, "mode"); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	a := &selectAction{}
	var err error
	if raw["pick"] != nil {
		if a.pick, err = parseSelectorSpec(raw["pick"], "target"); err != nil {
			return nil, fmt.Errorf("select pick: %v", err)
		}
	}
	if raw["actors"] != nil {
		if a.actors, err = parseSelectorSpec(raw["actors"], "actor"); err != nil {
			return nil, fmt.Errorf("select actors: %v", err)
		}
	}
	if raw["issue"] != nil {
		target_kind, actor_kind := "units", "buildings"
		if a.pick != nil {
			target_kind = a.pick.kind
		}
		if a.actors != nil {
			actor_kind = a.actors.kind
		}
		err = withLoopScopeExtra("target", "select_"+target_kind, a.pick.letNames(), func() error {
			return withLoopScopeExtra("actor", "select_"+actor_kind, a.actors.letNames(), func() (parsed error) {
				a.verbs, parsed = parseVerbs(raw["issue"], actor_kind)
				return parsed
			})
		})
		if err != nil {
			return nil, fmt.Errorf("select: %v", err)
		}
	}
	if raw["into"] != nil {
		if a.into, err = parseSink(raw["into"], a.pick); err != nil {
			return nil, fmt.Errorf("select into: %v", err)
		}
	}
	if a.pick != nil && a.pick.kind == "regions" && len(a.verbs) > 0 {
		return nil, fmt.Errorf("select: regions cannot be the target of a verb")
	}
	if err = a.parseQueue(raw["mode"]); err != nil {
		return nil, err
	}
	if err = a.parseActors(raw); err != nil {
		return nil, err
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
		if raw["actors"] != nil || raw["assign"] != nil || raw["mode"] != nil {
			return fmt.Errorf("select: \"actors\", \"assign\" and \"mode\" need an \"issue\"")
		}
		return nil
	}
	if slices.ContainsFunc(a.verbs, verb.actorless) {
		if len(a.verbs) > 1 || raw["actors"] != nil || raw["assign"] != nil || raw["mode"] != nil {
			return fmt.Errorf("select: hire and cancel_foundation must be the only verb and take no \"actors\", \"assign\" or \"mode\"")
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
	return a.parseAssign(raw)
}

func (a *selectAction) parseAssign(raw map[string]any) error {
	if raw["assign"] == nil {
		return nil
	}
	if !a.needsTarget() {
		return fmt.Errorf("select: \"assign\" needs a target-dependent verb")
	}
	kind := "units"
	if a.pick != nil {
		kind = a.pick.kind
	}
	parse := func() error {
		var err error
		a.assign, err = parseSpread(raw["assign"])
		return err
	}
	extra := a.pick.letNames()
	err := withLoopScopeExtra("actor", "select_"+a.actors.kind, a.actors.letNames(), func() error {
		return withLoopScopeExtra("target", "select_"+kind, extra, parse)
	})
	if err == nil && a.assign.by_space {
		err = fmt.Errorf("group_by is not supported")
	}
	if err != nil {
		return fmt.Errorf("select assign: %v", err)
	}
	return nil
}
