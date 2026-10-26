package ai

import (
	"fmt"
	"math"
)

func parseCondition(raw map[string]any) (Condition, error) {
	if len(raw) != 1 {
		return nil, fmt.Errorf("condition must have exactly one key, got %d", len(raw))
	}
	var key string
	var value any
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
		inner, ok := value.(map[string]any)
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
	case "tech_researched":
		obj, _ := value.(map[string]any)
		tech_id := parseOptionalID(obj, "tech_id")
		if tech_id == nil {
			return nil, fmt.Errorf("tech_researched requires a numeric \"tech_id\"")
		}
		return &conditionTechResearched{tech_id: *tech_id}, nil
	default:
		return parseCountCondition(key, value)
	}
}

func parseConditionList(value any, build func([]Condition) Condition) (Condition, error) {
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a list of conditions")
	}
	conditions := make([]Condition, 0, len(list))
	for _, item := range list {
		obj, ok := item.(map[string]any)
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

func parseAction(raw map[string]any) (Action, error) {
	return parseActionInner(raw)
}

func parseOptionalID(raw map[string]any, key string) *uint32 {
	f, ok := raw[key].(float64)
	if !ok || f < 1 || f > math.MaxUint32 || f != math.Trunc(f) {
		return nil
	}
	id := uint32(f)
	return &id
}

// 0 means unlimited; validated by parseForEach
func parseMax(raw map[string]any) amount {
	m, _ := parseNumber(raw, "max", 0)
	return m
}

func parseIDSet(raw map[string]any, key string) (map[uint32]bool, error) {
	ids := map[uint32]bool{}
	list, ok := raw[key].([]any)
	if _, present := raw[key]; present && !ok {
		return nil, fmt.Errorf("%q must be an array of numbers", key)
	}
	for _, id := range list {
		f, ok := id.(float64)
		if !ok || f < 1 || f > math.MaxUint32 || f != math.Trunc(f) {
			return nil, fmt.Errorf("%q entries must be whole numbers in 1..4294967295", key)
		}
		ids[uint32(f)] = true
	}
	return ids, nil
}

func parseUnitFilter(raw map[string]any) (unitFilter, error) {
	unit_ids, err := parseIDSet(raw, "unit_ids")
	if err != nil {
		return unitFilter{}, err
	}
	filter := unitFilter{unit_ids: unit_ids, unit_types: map[UnitType]bool{}}
	types, ok := raw["unit_types"].([]any)
	if _, present := raw["unit_types"]; present && !ok {
		return filter, fmt.Errorf("\"unit_types\" must be an array of unit type names")
	}
	for _, t := range types {
		named := unitTypesNamed(fmt.Sprint(t))
		if named == nil {
			return filter, fmt.Errorf("unknown unit type %v", t)
		}
		for _, unit_type := range named {
			filter.unit_types[unit_type] = true
		}
	}
	return filter, nil
}

func parseResourceCategory(raw any) (ResourceCategory, error) {
	s, ok := raw.(string)
	if !ok {
		return 0, fmt.Errorf("\"category\" must be a string")
	}
	switch s {
	case "food":
		return ResourceFood, nil
	case "wood":
		return ResourceWood, nil
	case "stone":
		return ResourceStone, nil
	case "gold":
		return ResourceGold, nil
	default:
		return 0, fmt.Errorf("unknown resource category %q", s)
	}
}

func parseRules(raw []any) ([]Rule, error) {
	rules := make([]Rule, 0, len(raw))
	for i, item := range raw {
		rule, err := parseRule(item)
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i, err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func parseRule(item any) (Rule, error) {
	obj, ok := item.(map[string]any)
	if !ok {
		return Rule{}, fmt.Errorf("rule must be an object")
	}
	when_raw, ok := obj["when"].(map[string]any)
	if !ok {
		return Rule{}, fmt.Errorf("missing \"when\" object")
	}
	when, err := parseCondition(when_raw)
	if err != nil {
		return Rule{}, err
	}
	then_raw, ok := obj["then"].([]any)
	if !ok {
		return Rule{}, fmt.Errorf("missing \"then\" list")
	}
	then := make([]Action, 0, len(then_raw))
	for j, action_raw := range then_raw {
		action_obj, ok := action_raw.(map[string]any)
		if !ok {
			return Rule{}, fmt.Errorf("action %d: must be an object", j)
		}
		action, err := parseAction(action_obj)
		if err != nil {
			return Rule{}, fmt.Errorf("action %d: %w", j, err)
		}
		then = append(then, action)
	}
	return Rule{when: when, then: then}, nil
}

func parseUnitBehavior(raw map[string]any) (UnitBehavior, error) {
	var behavior UnitBehavior
	if s, ok := raw["stance"].(string); ok {
		stance, ok := unitStanceNames[s]
		if !ok {
			return behavior, fmt.Errorf("unknown stance %q", s)
		}
		behavior.Stance = &stance
	}
	if b, ok := raw["interrupt_current"].(bool); ok {
		behavior.InterruptCurrent = &b
	}
	if b, ok := raw["attack_back"].(bool); ok {
		behavior.AttackBack = &b
	}
	priority, err := parseTargetPriority(raw)
	if err != nil {
		return behavior, err
	}
	behavior.TargetPriority = priority
	return behavior, nil
}

func parseTargetPriority(raw map[string]any) ([]TargetCategory, error) {
	list, ok := raw["target_priority"].([]any)
	if !ok {
		return nil, nil
	}
	priority := make([]TargetCategory, 0, len(list))
	for _, item := range list {
		category, ok := targetCategoryNames[fmt.Sprint(item)]
		if !ok {
			return nil, fmt.Errorf("unknown target category %v", item)
		}
		priority = append(priority, category)
	}
	return priority, nil
}

func parseBuildingBehavior(raw map[string]any) (BuildingBehavior, error) {
	var behavior BuildingBehavior
	if b, ok := raw["auto_attack"].(bool); ok {
		behavior.AutoAttack = &b
	}
	if b, ok := raw["interrupt_current"].(bool); ok {
		behavior.InterruptCurrent = &b
	}
	priority, err := parseTargetPriority(raw)
	if err != nil {
		return behavior, err
	}
	behavior.TargetPriority = priority
	return behavior, nil
}
