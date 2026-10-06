package ai

import "fmt"

// Runs a list of actions in order and gathers their orders
func runActions(actions []Action, view View, internals *Internals) []Order {
	orders := make([]Order, 0)
	for _, action := range actions {
		orders = append(orders, action.ToOrders(view, internals)...)
	}
	return orders
}

func parseActionList(raw any) ([]Action, error) {
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a list of actions")
	}
	actions := make([]Action, 0, len(list))
	for i, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("action %d: must be an object", i)
		}
		action, err := parseAction(obj)
		if err != nil {
			return nil, fmt.Errorf("action %d: %w", i, err)
		}
		actions = append(actions, action)
	}
	return actions, nil
}

type ifAction struct {
	when      Condition
	then      []Action
	otherwise []Action
}

func (a *ifAction) ToOrders(view View, internals *Internals) []Order {
	if a.when.Evaluate(view, internals) {
		return runActions(a.then, view, internals)
	}
	return runActions(a.otherwise, view, internals)
}

const maxNesting = 4

// How many if/for_each bodies deep the parser is; configs load one at a time, under the var-name lock
var nesting int

func parseNested(label string, raw any) ([]Action, error) {
	if nesting >= maxNesting {
		return nil, fmt.Errorf("%s: actions nested more than %d deep", label, maxNesting)
	}
	nesting++
	defer func() { nesting-- }()
	actions, err := parseActionList(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return actions, nil
}

func parseIf(raw map[string]any) (Action, error) {
	when_raw, ok := raw["when"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("if action requires a \"when\" condition object")
	}
	when, err := parseCondition(when_raw)
	if err != nil {
		return nil, fmt.Errorf("if: %v", err)
	}
	a := &ifAction{when: when}
	if a.then, err = parseNested("then", raw["then"]); err != nil {
		return nil, err
	}
	if raw["else"] != nil {
		a.otherwise, err = parseNested("else", raw["else"])
	}
	return a, err
}
