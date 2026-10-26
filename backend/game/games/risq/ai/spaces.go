package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type SpaceCondition struct {
	TerrainTypes              []string `json:"terrain_types,omitempty"`
	Passable                  *bool    `json:"passable,omitempty"`
	IntraCostAtLeast          *uint    `json:"intra_cost_at_least,omitempty"`
	IntraCostAtMost           *uint    `json:"intra_cost_at_most,omitempty"`
	InterCostAtLeast          *uint    `json:"inter_cost_at_least,omitempty"`
	InterCostAtMost           *uint    `json:"inter_cost_at_most,omitempty"`
	BuildSpeedModifierAtLeast *float64 `json:"build_speed_modifier_at_least,omitempty"`
	BuildSpeedModifierAtMost  *float64 `json:"build_speed_modifier_at_most,omitempty"`

	All              []SpaceCondition `json:"all,omitempty"`
	Any              []SpaceCondition `json:"any,omitempty"`
	Not              *SpaceCondition  `json:"not,omitempty"`
	Vision           *uint8           `json:"vision,omitempty"`
	VisionAtLeast    *uint8           `json:"vision_at_least,omitempty"`
	VisionAtMost     *uint8           `json:"vision_at_most,omitempty"`
	ResourceIDs      []uint32         `json:"resource_ids,omitempty"`
	BuildingIDs      []uint32         `json:"building_ids,omitempty"`
	Owner            string           `json:"owner,omitempty"`
	PlayerID         *int             `json:"player_id,omitempty"`
	BuildingOwner    string           `json:"building_owner,omitempty"`
	BuildingPlayerID *int             `json:"building_player_id,omitempty"`
	UnitCountAtLeast *int             `json:"unit_count_at_least,omitempty"`
	UnitCountAtMost  *int             `json:"unit_count_at_most,omitempty"`
}

func parseWhere(raw map[string]any) (SpaceCondition, error) {
	value, present := raw["where"]
	if !present {
		return SpaceCondition{}, nil
	}
	if value == nil {
		return SpaceCondition{}, fmt.Errorf("where must be an object")
	}
	return parseSpaceCondition(value)
}

func parseSpaceCondition(raw any) (SpaceCondition, error) {
	var condition SpaceCondition
	if raw == nil {
		return condition, nil
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return condition, fmt.Errorf("space condition must be an object")
	}
	for key, value := range obj {
		if value == nil {
			return condition, fmt.Errorf("space condition %q must not be null", key)
		}
		if key == "resource_ids" || key == "building_ids" {
			if _, err := parseIDSet(obj, key); err != nil {
				return condition, err
			}
		}
		if key == "owner" || key == "building_owner" {
			if owner, ok := value.(string); !ok || owner == "" {
				return condition, fmt.Errorf("%q must be an owner name", key)
			}
		}
		if key == "all" || key == "any" {
			children, ok := value.([]any)
			if !ok {
				return condition, fmt.Errorf("%q must be an array", key)
			}
			for _, child := range children {
				if _, ok := child.(map[string]any); !ok {
					return condition, fmt.Errorf("%q children must be objects", key)
				}
				if _, err := parseSpaceCondition(child); err != nil {
					return condition, err
				}
			}
		}
		if key == "not" {
			if _, ok := value.(map[string]any); !ok {
				return condition, fmt.Errorf("not must be an object")
			}
			if _, err := parseSpaceCondition(value); err != nil {
				return condition, err
			}
		}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return condition, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&condition); err != nil {
		return condition, err
	}
	return condition, condition.validate()
}

func (c SpaceCondition) validate() error {
	for _, terrain := range c.TerrainTypes {
		switch terrain {
		case "flatlands", "hilly", "mountainous", "swamp", "shallows", "water", "deep_water":
		default:
			return fmt.Errorf("unknown terrain type %q", terrain)
		}
	}
	for _, vision := range []*uint8{c.Vision, c.VisionAtLeast, c.VisionAtMost} {
		if vision != nil && *vision > 4 {
			return fmt.Errorf("vision must be between 0 and 4")
		}
	}
	for _, owner := range []string{c.Owner, c.BuildingOwner} {
		if owner != "" && owner != "own" && owner != "enemy" && owner != "unowned" && owner != "any" {
			return fmt.Errorf("owner must be own, enemy, unowned or any")
		}
	}
	for _, child := range append(append([]SpaceCondition{}, c.All...), c.Any...) {
		if err := child.validate(); err != nil {
			return err
		}
	}
	if c.Not != nil {
		return c.Not.validate()
	}
	return nil
}
