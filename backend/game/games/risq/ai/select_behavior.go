package ai

import "fmt"

func checkBehaviorOptions(options map[string]any, actor_kind string) error {
	allowed := []string{"interrupt_current", "target_priority"}
	if actor_kind == "units" {
		allowed = append(allowed, "stance", "attack_back")
	} else {
		allowed = append(allowed, "auto_attack")
	}
	if err := checkKeys(options, allowed...); err != nil {
		return err
	}
	if len(options) == 0 {
		return fmt.Errorf("set_behavior needs at least one setting")
	}
	return checkBehaviorTypes(options)
}

func checkBehaviorTypes(options map[string]any) error {
	for _, key := range []string{"stance", "target_priority", "interrupt_current", "attack_back", "auto_attack"} {
		if value, present := options[key]; present {
			var valid bool
			switch key {
			case "stance":
				_, valid = value.(string)
			case "target_priority":
				_, valid = value.([]any)
			default:
				_, valid = value.(bool)
			}
			if !valid {
				return fmt.Errorf("invalid set_behavior value for %q", key)
			}
		}
	}
	return nil
}

func parseBehaviorVerb(v verb, options map[string]any, actor_kind string) (verb, error) {
	if actor_kind == "units" {
		behavior, err := parseUnitBehavior(options)
		v.unit_behavior = &behavior
		return v, err
	}
	behavior, err := parseBuildingBehavior(options)
	v.building_behavior = &behavior
	return v, err
}

func (c *issueContext) setBehavior(v verb, actor loopItem) {
	switch a := unwrapItem(actor).(type) {
	case unitItem:
		behavior := *v.unit_behavior
		behavior.Subjects = []uint64{a.unit.InternalID}
		c.internals.behaviors = append(c.internals.behaviors, behavior)
	case buildingItem:
		behavior := *v.building_behavior
		behavior.Subjects = []uint64{a.building.InternalID}
		c.internals.building_behaviors = append(c.internals.building_behaviors, behavior)
	}
}
