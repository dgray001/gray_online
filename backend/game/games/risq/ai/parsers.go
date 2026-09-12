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
	case "building_count_equals":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("building_count_equals must be an object")
		}
		count, ok := obj["count"].(float64)
		if !ok {
			return nil, fmt.Errorf("building_count_equals requires a numeric \"count\"")
		}
		condition := &conditionBuildingCountEquals{count: int(count)}
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
	case "population_headroom_equals":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("population_headroom_equals must be an object")
		}
		amount, ok := obj["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("population_headroom_equals requires a numeric \"amount\"")
		}
		return &conditionPopulationHeadroomEquals{amount: int(amount)}, nil
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
	case "population_equals":
		obj, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("population_equals must be an object")
		}
		amount, ok := obj["amount"].(float64)
		if !ok {
			return nil, fmt.Errorf("population_equals requires a numeric \"amount\"")
		}
		condition := &conditionPopulationEquals{amount: int(amount)}
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
			weight := 0.0
			if w, ok := raw["weight"].(float64); ok {
				weight = w
			}
			return &balancedGatherAction{eligible: eligible, weight: weight}, nil
		}
		if _, has_weight := raw["weight"]; has_weight {
			return nil, fmt.Errorf("gather action with a category must not specify \"weight\"")
		}
		category, err := parseResourceCategory(raw["category"])
		if err != nil {
			return nil, err
		}
		return &gatherAction{category: category, eligible: eligible, max: parseMax(raw)}, nil
	case "create":
		id, ok := raw["unit_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("create action requires a numeric \"unit_id\"")
		}
		return &createAction{unit_id: uint32(id)}, nil
	case "createNextInQ":
		weight, prioritize := parseQueueParams(raw)
		return &createNextInQAction{weight: weight, prioritize: prioritize}, nil
	case "research":
		id, ok := raw["tech_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("research action requires a numeric \"tech_id\"")
		}
		return &researchAction{tech_id: uint32(id)}, nil
	case "researchNextInQ":
		weight, prioritize := parseQueueParams(raw)
		return &researchNextInQAction{weight: weight, prioritize: prioritize}, nil
	case "build":
		id, ok := raw["building_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("build action requires a numeric \"building_id\"")
		}
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		return &buildAction{building_id: uint32(id), eligible: eligible, max: parseMax(raw)}, nil
	case "buildNextInQ":
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		weight, prioritize := parseQueueParams(raw)
		return &buildNextInQAction{eligible: eligible, weight: weight, prioritize: prioritize, max: parseMax(raw)}, nil
	case "produce":
		weight, prioritize := parseQueueParams(raw)
		return &produceNextInQAction{weight: weight, prioritize: prioritize, max: parseMax(raw)}, nil
	case "explore":
		action := &exploreAction{max: parseMax(raw)}
		if a, ok := raw["anchor"].(string); ok {
			switch a {
			case "self":
				action.anchor = exploreAnchorSelf
			case "home":
				action.anchor = exploreAnchorHome
			case "center":
				action.anchor = exploreAnchorCenter
			default:
				return nil, fmt.Errorf("unknown explore anchor %q", a)
			}
		}
		return action, nil
	case "attack":
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		action := &attackAction{target: attackTargetMilitary, max: parseMax(raw), eligible: eligible}
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
	case "attack_space":
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		return &attackSpaceAction{max: parseMax(raw), eligible: eligible}, nil
	case "attack_zone":
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		return &attackZoneAction{max: parseMax(raw), eligible: eligible}, nil
	case "garrison":
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		action := &garrisonAction{eligible: eligible, max: parseMax(raw)}
		if id, ok := raw["building_id"].(float64); ok {
			building_id := uint32(id)
			action.building_id = &building_id
		}
		return action, nil
	case "ungarrison":
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		return &ungarrisonAction{eligible: eligible, max: parseMax(raw)}, nil
	case "add_q":
		type_str, ok := raw["type"].(string)
		if !ok {
			return nil, fmt.Errorf("add_q action requires a string \"type\"")
		}
		weight := 1.0
		if w, ok := raw["weight"].(float64); ok {
			weight = w
		}
		if type_str == "resource" {
			if _, has_id := raw["id"]; has_id {
				return nil, fmt.Errorf("add_q action with type \"resource\" must not specify \"id\"")
			}
			cost, err := parseCost(raw["cost"])
			if err != nil {
				return nil, err
			}
			return &addQAction{q_type: QResource, cost: cost, weight: weight}, nil
		}
		var q_type QKind
		switch type_str {
		case "unit":
			q_type = QUnit
		case "building":
			q_type = QBuilding
		case "technology":
			q_type = QTech
		default:
			return nil, fmt.Errorf("unknown add_q type %q", type_str)
		}
		if _, has_cost := raw["cost"]; has_cost {
			return nil, fmt.Errorf("add_q action with type %q must not specify \"cost\"", type_str)
		}
		id_f, ok := raw["id"].(float64)
		if !ok {
			return nil, fmt.Errorf("add_q action with type %q requires a numeric \"id\"", type_str)
		}
		id := uint32(id_f)
		return &addQAction{q_type: q_type, id: &id, weight: weight}, nil
	default:
		return nil, fmt.Errorf("unknown action type %q", action_type)
	}
}

func parseQueueParams(raw map[string]any) (weight float64, prioritize bool) {
	if w, ok := raw["weight"].(float64); ok {
		weight = w
	}
	prioritize, _ = raw["prioritize"].(bool)
	return weight, prioritize
}

// 0 means unlimited
func parseMax(raw map[string]any) int {
	if m, ok := raw["max"].(float64); ok {
		return int(m)
	}
	return 0
}

func parseCost(raw any) (Cost, error) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return Cost{}, fmt.Errorf("\"cost\" must be an object")
	}
	cost := Cost{}
	if v, ok := obj["food"].(float64); ok {
		cost.Food = v
	}
	if v, ok := obj["wood"].(float64); ok {
		cost.Wood = v
	}
	if v, ok := obj["stone"].(float64); ok {
		cost.Stone = v
	}
	if v, ok := obj["gold"].(float64); ok {
		cost.Gold = v
	}
	return cost, nil
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
