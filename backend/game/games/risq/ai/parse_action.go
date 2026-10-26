package ai

import "fmt"

func parseActionInner(raw map[string]any) (Action, error) {
	action_type, ok := raw["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action must have a string \"action\" field")
	}
	switch action_type {
	case "select":
		return parseSelect(raw)
	case "set_var":
		return parseSetVar(raw)
	case "for_each":
		return parseForEach(raw)
	case "if":
		return parseIf(raw)
	default:
		return nil, fmt.Errorf("unknown action type %q", action_type)
	}
}
