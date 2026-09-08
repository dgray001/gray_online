package ai

import "fmt"

type conditionAll struct{ conditions []Condition }

func (c *conditionAll) Evaluate(view View) bool {
	for _, condition := range c.conditions {
		if !condition.Evaluate(view) {
			return false
		}
	}
	return true
}

type conditionAny struct{ conditions []Condition }

func (c *conditionAny) Evaluate(view View) bool {
	for _, condition := range c.conditions {
		if condition.Evaluate(view) {
			return true
		}
	}
	return false
}

type conditionNot struct{ condition Condition }

func (c *conditionNot) Evaluate(view View) bool {
	return !c.condition.Evaluate(view)
}

type conditionAlways struct{}

func (c *conditionAlways) Evaluate(View) bool {
	return true
}

func parseCondition(raw map[string]interface{}) (Condition, error) {
	if len(raw) != 1 {
		return nil, fmt.Errorf("condition must have exactly one key, got %d", len(raw))
	}
	var key string
	var value interface{}
	for k, v := range raw {
		key, value = k, v
	}
	switch key {
	case "all":
		return parseConditionList(value, func(cs []Condition) Condition {
			return &conditionAll{conditions: cs}
		})
	case "any":
		return parseConditionList(value, func(cs []Condition) Condition {
			return &conditionAny{conditions: cs}
		})
	case "not":
		inner, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("not condition must be an object")
		}
		condition, err := parseCondition(inner)
		if err != nil {
			return nil, err
		}
		return &conditionNot{condition: condition}, nil
	case "always":
		return &conditionAlways{}, nil
	default:
		return nil, fmt.Errorf("unknown condition type %q", key)
	}
}

func parseConditionList(value interface{}, build func([]Condition) Condition) (Condition, error) {
	list, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected a list of conditions")
	}
	conditions := make([]Condition, 0, len(list))
	for _, item := range list {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("condition list item must be an object")
		}
		condition, err := parseCondition(obj)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, condition)
	}
	return build(conditions), nil
}
