package risq

import "fmt"

type RisqAiCondition interface {
	evaluate(p *RisqPlayer, r *GameRisq) bool
}

type risqAiConditionAll struct{ conditions []RisqAiCondition }

func (c *risqAiConditionAll) evaluate(p *RisqPlayer, r *GameRisq) bool {
	for _, condition := range c.conditions {
		if !condition.evaluate(p, r) {
			return false
		}
	}
	return true
}

type risqAiConditionAny struct{ conditions []RisqAiCondition }

func (c *risqAiConditionAny) evaluate(p *RisqPlayer, r *GameRisq) bool {
	for _, condition := range c.conditions {
		if condition.evaluate(p, r) {
			return true
		}
	}
	return false
}

type risqAiConditionNot struct{ condition RisqAiCondition }

func (c *risqAiConditionNot) evaluate(p *RisqPlayer, r *GameRisq) bool {
	return !c.condition.evaluate(p, r)
}

type risqAiConditionAlways struct{}

func (c *risqAiConditionAlways) evaluate(p *RisqPlayer, r *GameRisq) bool {
	return true
}

func parseRisqAiCondition(raw map[string]interface{}) (RisqAiCondition, error) {
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
		return parseRisqAiConditionList(value, func(cs []RisqAiCondition) RisqAiCondition {
			return &risqAiConditionAll{conditions: cs}
		})
	case "any":
		return parseRisqAiConditionList(value, func(cs []RisqAiCondition) RisqAiCondition {
			return &risqAiConditionAny{conditions: cs}
		})
	case "not":
		inner, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("not condition must be an object")
		}
		condition, err := parseRisqAiCondition(inner)
		if err != nil {
			return nil, err
		}
		return &risqAiConditionNot{condition: condition}, nil
	case "always":
		return &risqAiConditionAlways{}, nil
	default:
		return nil, fmt.Errorf("unknown condition type %q", key)
	}
}

func parseRisqAiConditionList(value interface{}, build func([]RisqAiCondition) RisqAiCondition) (RisqAiCondition, error) {
	list, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected a list of conditions")
	}
	conditions := make([]RisqAiCondition, 0, len(list))
	for _, item := range list {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("condition list item must be an object")
		}
		condition, err := parseRisqAiCondition(obj)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, condition)
	}
	return build(conditions), nil
}
