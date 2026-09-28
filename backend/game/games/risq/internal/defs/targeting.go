package defs

import "fmt"

type TargetCategory uint8

const (
	TargetCategory_NONE TargetCategory = iota
	TargetCategory_ECONOMIC
	TargetCategory_MILITARY
	TargetCategory_BUILDING
	TargetCategory_END
)

type TargetType struct {
	Orderable_type OrderableType
	Object_type    int // UnitType when orderable_type is OrderableType_UNIT; 0 means "any" within that orderable_type
}

func (tt TargetType) Matches(actual TargetType) bool {
	if tt.Orderable_type != actual.Orderable_type {
		return false
	}
	return tt.Object_type == 0 || tt.Object_type == actual.Object_type
}

func parseTargetType(s string) (TargetType, error) {
	switch s {
	case "building":
		return TargetType{Orderable_type: OrderableType_BUILDING}, nil
	case "unit":
		return TargetType{Orderable_type: OrderableType_UNIT}, nil
	default:
		unit_type, err := parseUnitType(s)
		if err != nil {
			return TargetType{}, fmt.Errorf("unknown target_type %q", s)
		}
		return TargetType{Orderable_type: OrderableType_UNIT, Object_type: int(unit_type)}, nil
	}
}

func targetTypeOf(orderable_type OrderableType, unit_type UnitType) TargetType {
	if orderable_type == OrderableType_UNIT {
		return TargetType{Orderable_type: OrderableType_UNIT, Object_type: int(unit_type)}
	}
	return TargetType{Orderable_type: orderable_type}
}
