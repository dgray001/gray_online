package ai

import "fmt"

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
	case "building_count_at_least":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("building_count_at_least must be an object")
		}
		count, ok := obj["count"].(float64)
		if !ok {
			return nil, fmt.Errorf("building_count_at_least requires a numeric \"count\"")
		}
		condition := &conditionBuildingCountAtLeast{count: int(count)}
		if id, ok := obj["building_id"].(float64); ok {
			building_id := uint32(id)
			condition.building_id = &building_id
		}
		return condition, nil
	case "building_count_at_most":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("building_count_at_most must be an object")
		}
		count, ok := obj["count"].(float64)
		if !ok {
			return nil, fmt.Errorf("building_count_at_most requires a numeric \"count\"")
		}
		condition := &conditionBuildingCountAtMost{count: int(count)}
		if id, ok := obj["building_id"].(float64); ok {
			building_id := uint32(id)
			condition.building_id = &building_id
		}
		return condition, nil
	case "population_headroom_at_least":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("population_headroom_at_least must be an object")
		}
		amount, ok := obj["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("population_headroom_at_least requires a numeric \"amount\"")
		}
		return &conditionPopulationHeadroomAtLeast{amount: int(amount)}, nil
	case "population_headroom_at_most":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("population_headroom_at_most must be an object")
		}
		amount, ok := obj["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("population_headroom_at_most requires a numeric \"amount\"")
		}
		return &conditionPopulationHeadroomAtMost{amount: int(amount)}, nil
	case "population_at_least":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("population_at_least must be an object")
		}
		amount, ok := obj["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("population_at_least requires a numeric \"amount\"")
		}
		condition := &conditionPopulationAtLeast{amount: int(amount)}
		if id, ok := obj["unit_id"].(float64); ok {
			unit_id := uint32(id)
			condition.unit_id = &unit_id
		}
		return condition, nil
	case "population_at_most":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("population_at_most must be an object")
		}
		amount, ok := obj["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("population_at_most requires a numeric \"amount\"")
		}
		condition := &conditionPopulationAtMost{amount: int(amount)}
		if id, ok := obj["unit_id"].(float64); ok {
			unit_id := uint32(id)
			condition.unit_id = &unit_id
		}
		return condition, nil
	default:
		return nil, fmt.Errorf("unknown condition type %q", key)
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
	action_type, ok := raw["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action must have a string \"action\" field")
	}
	switch action_type {
	case "gather":
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		if raw["category"] == nil {
			return &balancedGatherAction{eligible: eligible}, nil
		}
		category, err := parseResourceCategory(raw["category"])
		if err != nil {
			return nil, err
		}
		return &gatherAction{category: category, eligible: eligible}, nil
	case "produce":
		id, ok := raw["unit_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("produce action requires a numeric \"unit_id\"")
		}
		return &produceAction{unit_id: uint32(id)}, nil
	case "research":
		id, ok := raw["tech_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("research action requires a numeric \"tech_id\"")
		}
		return &researchAction{tech_id: uint32(id)}, nil
	case "build":
		id, ok := raw["building_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("build action requires a numeric \"building_id\"")
		}
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		return &buildAction{building_id: uint32(id), eligible: eligible}, nil
	case "explore":
		return &exploreAction{}, nil
	case "attack":
		action := &attackAction{target: attackTargetMilitary}
		if t, ok := raw["target"].(string); ok {
			switch t {
			case "military":
				action.target = attackTargetMilitary
			case "economic":
				action.target = attackTargetEconomic
			case "any":
				action.target = attackTargetAny
			default:
				return nil, fmt.Errorf("unknown attack target %q", t)
			}
		}
		return action, nil
	default:
		return nil, fmt.Errorf("unknown action type %q", action_type)
	}
}

func parseEligible(raw any) ([]OrderKind, error) {
	if raw == nil {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("\"eligible\" must be a list of strings")
	}
	kinds := make([]OrderKind, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("\"eligible\" entries must be strings")
		}
		kind, ok := orderKindNames[s]
		if !ok {
			return nil, fmt.Errorf("unknown order kind %q", s)
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}

func parseResourceCategory(raw any) (ResourceCategory, error) {
	s, ok := raw.(string)
	if !ok {
		return 0, fmt.Errorf("gather action requires a string \"category\"")
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
	for _, item := range raw {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("rule must be an object")
		}
		when_raw, ok := obj["when"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("rule missing \"when\" object")
		}
		when, err := parseCondition(when_raw)
		if err != nil {
			return nil, err
		}
		then_raw, ok := obj["then"].([]any)
		if !ok {
			return nil, fmt.Errorf("rule missing \"then\" list")
		}
		then := make([]Action, 0, len(then_raw))
		for _, action_raw := range then_raw {
			action_obj, ok := action_raw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("action must be an object")
			}
			action, err := parseAction(action_obj)
			if err != nil {
				return nil, err
			}
			then = append(then, action)
		}
		rules = append(rules, Rule{when: when, then: then})
	}
	return rules, nil
}
