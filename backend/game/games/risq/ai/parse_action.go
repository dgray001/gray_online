package ai

import "fmt"

// Inputs that take a number or an expression string
var numericActionKeys = []string{"max", "queue", "weight", "move_penalty", "depth", "size", "reserve", "launch", "retreat", "defend_radius", "strike", "score", "min_score"}

func parseActionInner(raw map[string]any) (Action, error) {
	action_type, ok := raw["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action must have a string \"action\" field")
	}
	for _, key := range numericActionKeys {
		if _, err := parseNumber(raw, key, 0); err != nil {
			return nil, fmt.Errorf("%s: %v", action_type, err)
		}
	}
	switch action_type {
	case "gather":
		eligible, err := parseEligible(raw["eligible"])
		if err != nil {
			return nil, err
		}
		if raw["category"] == nil {
			weight, _ := parseNumber(raw, "weight", 0)
			move_penalty, _ := parseNumber(raw, "move_penalty", 1)
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
		queue, _ := parseNumber(raw, "queue", 1)
		return &createAction{unit_id: uint32(id), queue: queue}, nil
	case "createNextInQ":
		weight, prioritize, depth := parseQueueParams(raw)
		return &createNextInQAction{weight: weight, prioritize: prioritize, depth: depth}, nil
	case "research":
		id, ok := raw["tech_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("research action requires a numeric \"tech_id\"")
		}
		queue, _ := parseNumber(raw, "queue", 1)
		return &researchAction{tech_id: uint32(id), queue: queue}, nil
	case "researchNextInQ":
		weight, prioritize, depth := parseQueueParams(raw)
		return &researchNextInQAction{weight: weight, prioritize: prioritize, depth: depth}, nil
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
		weight, prioritize, depth := parseQueueParams(raw)
		return &buildNextInQAction{eligible: eligible, weight: weight, prioritize: prioritize, depth: depth, max: parseMax(raw)}, nil
	case "produce":
		weight, prioritize, depth := parseQueueParams(raw)
		return &produceNextInQAction{weight: weight, prioritize: prioritize, depth: depth, max: parseMax(raw)}, nil
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
		picker, err := parseTargetPicker(raw)
		if err != nil {
			return nil, err
		}
		return &attackAction{picker: picker, target: target, max: parseMax(raw), eligible: eligible}, nil
	case "hire":
		id, ok := raw["unit_id"].(float64)
		if !ok {
			return nil, fmt.Errorf("hire action requires a numeric \"unit_id\"")
		}
		reserve, _ := parseNumber(raw, "reserve", 0)
		return &hireAction{unit_id: uint32(id), max: parseMax(raw), reserve: reserve}, nil
	case "army":
		ids, err := parseIDSet(raw, "assault_unit_ids")
		if err != nil {
			return nil, err
		}
		launch, _ := parseNumber(raw, "launch", 30)
		retreat, _ := parseNumber(raw, "retreat", 8)
		radius, _ := parseNumber(raw, "defend_radius", 2)
		strike, _ := parseNumber(raw, "strike", 0)
		return &armyAction{assault_ids: ids, launch: launch, retreat: retreat, defend_radius: radius, strike: strike}, nil
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
		if raw["size"] == nil {
			return nil, fmt.Errorf("set_bucket action requires a \"size\"")
		}
		size, _ := parseNumber(raw, "size", 0)
		task_raw, ok := raw["task"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("set_bucket action requires an object \"task\"")
		}
		task, err := parseAction(task_raw)
		if err != nil {
			return nil, err
		}
		return &setBucketAction{bucket: bucket, size: size, task: task}, nil
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
	case "move":
		return parseMove(raw)
	case "set_var":
		return parseSetVar(raw)
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
		weight, _ := parseNumber(raw, "weight", 1)
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
