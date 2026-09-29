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
	case "tech_researched":
		obj, _ := value.(map[string]any)
		tech_id := parseOptionalID(obj, "tech_id")
		if tech_id == nil {
			return nil, fmt.Errorf("tech_researched requires a numeric \"tech_id\"")
		}
		return &conditionTechResearched{tech_id: *tech_id}, nil
	case "bucket_full":
		obj, _ := value.(map[string]any)
		bucket, ok := obj["bucket"].(string)
		if !ok {
			return nil, fmt.Errorf("bucket_full requires a string \"bucket\"")
		}
		return &conditionBucketFull{bucket: bucket}, nil
	case "resource_available":
		obj, _ := value.(map[string]any)
		category, err := parseResourceCategory(obj["category"])
		if err != nil {
			return nil, err
		}
		return &conditionResourceAvailable{category: category}, nil
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
	action, err := parseActionInner(raw)
	if err != nil {
		return nil, err
	}
	filter, err := parseUnitFilter(raw)
	if err != nil {
		return nil, err
	}
	if f, ok := action.(interface{ setFilter(unitFilter) }); ok {
		f.setFilter(filter)
	} else if len(filter.unit_ids) > 0 || len(filter.unit_types) > 0 {
		return nil, fmt.Errorf("action %q does not take unit filters", raw["action"])
	}
	building_ids, err := parseIDSet(raw, "building_ids")
	if err != nil {
		return nil, err
	}
	if f, ok := action.(interface{ setBuildingFilter(buildingFilter) }); ok {
		f.setBuildingFilter(buildingFilter{ids: building_ids})
	} else if len(building_ids) > 0 {
		return nil, fmt.Errorf("action %q does not take building filters", raw["action"])
	}
	if raw["unit_when"] != nil {
		obj, ok := raw["unit_when"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("\"unit_when\" must be a condition object")
		}
		when, err := parseCondition(obj)
		if err != nil {
			return nil, fmt.Errorf("\"unit_when\": %v", err)
		}
		action = &unitWhenAction{when: when, inner: action}
	}
	if bucket, ok := raw["in_bucket"].(string); ok {
		action = &inBucketAction{bucket: bucket, inner: action}
	}
	if exclude, _ := raw["exclude_buckets"].(bool); exclude {
		action = &unbucketedAction{inner: action}
	}
	return action, nil
}

func parseQueueParams(raw map[string]any) (weight amount, prioritize bool, depth amount) {
	weight, _ = parseNumber(raw, "weight", 0)
	prioritize, _ = raw["prioritize"].(bool)
	depth, _ = parseNumber(raw, "depth", 1)
	return weight, prioritize, depth
}

func parseAttackTarget(raw map[string]any) (attackTarget, error) {
	t, ok := raw["target"].(string)
	if !ok {
		return attackTargetMilitary, nil
	}
	switch t {
	case "military":
		return attackTargetMilitary, nil
	case "economic":
		return attackTargetEconomic, nil
	case "any":
		return attackTargetAny, nil
	}
	return 0, fmt.Errorf("unknown attack target %q", t)
}

func parseOptionalID(raw map[string]any, key string) *uint32 {
	f, ok := raw[key].(float64)
	if !ok {
		return nil
	}
	id := uint32(f)
	return &id
}

// 0 means unlimited; validated by parseActionInner
func parseMax(raw map[string]any) amount {
	m, _ := parseNumber(raw, "max", 0)
	return m
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

func parseIDSet(raw map[string]any, key string) (map[uint32]bool, error) {
	ids := map[uint32]bool{}
	list, ok := raw[key].([]any)
	if raw[key] != nil && !ok {
		return nil, fmt.Errorf("%q must be an array of numbers", key)
	}
	for _, id := range list {
		f, ok := id.(float64)
		if !ok {
			return nil, fmt.Errorf("%q entries must be numbers", key)
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
	if raw["unit_types"] != nil && !ok {
		return filter, fmt.Errorf("\"unit_types\" must be an array of unit type names")
	}
	for _, t := range types {
		unit_type, ok := unitTypeNames[fmt.Sprint(t)]
		if !ok {
			return filter, fmt.Errorf("unknown unit type %v", t)
		}
		filter.unit_types[unit_type] = true
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

func parseSetUnitBehavior(raw map[string]any) (Action, error) {
	action := &setUnitBehaviorAction{}
	if s, ok := raw["stance"].(string); ok {
		stance, ok := unitStanceNames[s]
		if !ok {
			return nil, fmt.Errorf("unknown stance %q", s)
		}
		action.behavior.Stance = &stance
	}
	if b, ok := raw["interrupt_current"].(bool); ok {
		action.behavior.InterruptCurrent = &b
	}
	if b, ok := raw["attack_back"].(bool); ok {
		action.behavior.AttackBack = &b
	}
	priority, err := parseTargetPriority(raw)
	if err != nil {
		return nil, err
	}
	action.behavior.TargetPriority = priority
	return action, nil
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

func parseStop(action_type string, raw map[string]any) (Action, error) {
	if action_type == "unit_stop" {
		kinds, err := parseEligible(raw["orders"])
		if err != nil {
			return nil, err
		}
		orders := make(map[OrderKind]bool, len(kinds))
		for _, k := range kinds {
			orders[k] = true
		}
		return &unitStopAction{orders: orders, max: parseMax(raw)}, nil
	}
	names, ok := raw["orders"].([]any)
	if raw["orders"] != nil && !ok {
		return nil, fmt.Errorf("\"orders\" must be a list of strings")
	}
	orders := make(map[BuildingOrderKind]bool, len(names))
	for _, name := range names {
		kind, ok := buildingOrderKindNames[fmt.Sprint(name)]
		if !ok {
			return nil, fmt.Errorf("unknown building order kind %v", name)
		}
		orders[kind] = true
	}
	item_ids, err := parseIDSet(raw, "item_ids")
	if err != nil {
		return nil, err
	}
	return &buildingStopAction{orders: orders, item_ids: item_ids, max: parseMax(raw)}, nil
}

func parseSetBuildingBehavior(raw map[string]any) (Action, error) {
	action := &setBuildingBehaviorAction{}
	if b, ok := raw["auto_attack"].(bool); ok {
		action.behavior.AutoAttack = &b
	}
	if b, ok := raw["interrupt_current"].(bool); ok {
		action.behavior.InterruptCurrent = &b
	}
	priority, err := parseTargetPriority(raw)
	if err != nil {
		return nil, err
	}
	action.behavior.TargetPriority = priority
	return action, nil
}
