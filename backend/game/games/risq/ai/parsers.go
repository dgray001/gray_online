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
	if exclude, _ := raw["exclude_buckets"].(bool); exclude {
		action = &unbucketedAction{inner: action}
	}
	return action, nil
}

func parseActionInner(raw map[string]any) (Action, error) {
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
			move_penalty := 1.0
			if p, ok := raw["move_penalty"].(float64); ok {
				move_penalty = p
			}
			return &balancedGatherAction{eligible: eligible, weight: weight, move_penalty: move_penalty}, nil
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
		target, err := parseAttackTarget(raw)
		if err != nil {
			return nil, err
		}
		return &attackAction{target: target, max: parseMax(raw), eligible: eligible}, nil
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
		return &garrisonAction{eligible: eligible, max: parseMax(raw)}, nil
	case "ungarrison":
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		return &ungarrisonAction{eligible: eligible, max: parseMax(raw)}, nil
	case "set_bucket":
		bucket, ok := raw["bucket"].(string)
		if !ok {
			return nil, fmt.Errorf("set_bucket action requires a string \"bucket\"")
		}
		size, ok := raw["size"].(float64)
		if !ok {
			return nil, fmt.Errorf("set_bucket action requires a numeric \"size\"")
		}
		task_raw, ok := raw["task"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("set_bucket action requires an object \"task\"")
		}
		task, err := parseAction(task_raw)
		if err != nil {
			return nil, err
		}
		return &setBucketAction{bucket: bucket, size: int(size), task: task}, nil
	case "fill_bucket":
		bucket, ok := raw["bucket"].(string)
		if !ok {
			return nil, fmt.Errorf("fill_bucket action requires a string \"bucket\"")
		}
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		return &fillBucketAction{bucket: bucket, eligible: eligible}, nil
	case "run_bucket":
		bucket, ok := raw["bucket"].(string)
		if !ok {
			return nil, fmt.Errorf("run_bucket action requires a string \"bucket\"")
		}
		when_full, _ := raw["when_full"].(bool)
		return &runBucketAction{bucket: bucket, when_full: when_full}, nil
	case "drain_bucket":
		to, ok := raw["to"].(string)
		if !ok {
			return nil, fmt.Errorf("drain_bucket action requires a string \"to\"")
		}
		action := &drainBucketAction{to: to, max: parseMax(raw)}
		switch from := raw["from"].(type) {
		case string:
			if from == "any" {
				action.from_any = true
			} else {
				action.from = []string{from}
			}
		case []any:
			for _, f := range from {
				s, ok := f.(string)
				if !ok {
					return nil, fmt.Errorf("drain_bucket action's \"from\" array must contain strings")
				}
				action.from = append(action.from, s)
			}
		default:
			return nil, fmt.Errorf("drain_bucket action requires a string or array \"from\"")
		}
		return action, nil
	case "build_foundations", "renew", "repair", "delete_unit":
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		switch action_type {
		case "renew":
			return &renewAction{eligible: eligible, max: parseMax(raw)}, nil
		case "repair":
			return &repairAction{eligible: eligible, max: parseMax(raw)}, nil
		case "delete_unit":
			return &deleteUnitAction{eligible: eligible, max: parseMax(raw)}, nil
		}
		return &buildFoundationsAction{eligible: eligible, max: parseMax(raw)}, nil
	case "delete_building":
		return &deleteBuildingAction{max: parseMax(raw)}, nil
	case "building_attack":
		target, err := parseAttackTarget(raw)
		if err != nil {
			return nil, err
		}
		return &buildingAttackAction{target: target, max: parseMax(raw)}, nil
	case "set_unit_behavior":
		return parseSetUnitBehavior(raw)
	case "set_building_behavior":
		return parseSetBuildingBehavior(raw)
	case "unit_stop", "building_stop":
		return parseStop(action_type, raw)
	case "empty_bucket":
		bucket, ok := raw["bucket"].(string)
		if !ok {
			return nil, fmt.Errorf("empty_bucket action requires a string \"bucket\"")
		}
		return &emptyBucketAction{bucket: bucket}, nil
	case "cancel_foundations":
		return &cancelFoundationsAction{}, nil
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
	types, _ := raw["unit_types"].([]any)
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
