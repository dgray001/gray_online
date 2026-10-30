package ai

import (
	"fmt"
	"slices"
)

func parseCarry(raw map[string]any) (string, bool, error) {
	match, _ := raw["match"].(map[string]any)
	value, present := match["carry"]
	if !present {
		return "", false, nil
	}
	for key := range raw {
		if key != "action" && key != "match" {
			return "", true, fmt.Errorf("select carry takes no %q", key)
		}
	}
	name, ok := value.(string)
	if len(match) != 1 || !ok || !validVarName(name) {
		return "", true, fmt.Errorf("select carry needs only a valid assignment name in match")
	}
	return name, true, nil
}

func parseAssignInto(raw map[string]any) (string, bool, error) {
	into, _ := raw["into"].(map[string]any)
	value, present := into["assign"]
	if !present {
		return "", false, nil
	}
	name, ok := value.(string)
	if len(into) != 1 || !ok || !validVarName(name) {
		return "", true, fmt.Errorf("select into assign takes only a valid assignment name")
	}
	for _, key := range []string{"pick", "actors", "match", "issue"} {
		if raw[key] == nil {
			return "", true, fmt.Errorf("select into assign needs %q", key)
		}
	}
	if mode, present := raw["mode"]; present && mode != "replace" && mode != "append" {
		return "", true, fmt.Errorf("select into assign takes only replace or append mode")
	}
	return name, true, nil
}

func (a *selectAction) checkAssignShape() error {
	untargeted := func(v verb) bool { return !v.needsTargetVerb() }
	if slices.ContainsFunc(a.verbs, untargeted) {
		return fmt.Errorf("select into assign needs target-dependent order verbs only")
	}
	return nil
}

func (v verb) needsTargetVerb() bool {
	return v.name != "delete" && v.name != "ungarrison" && !v.production() && !v.control() && !v.actorless()
}

func (a *selectAction) actorScopeNames() []string {
	names := a.actors.letNames()
	if a.assign != "" {
		names = append(names, "assign_round")
	}
	return names
}
