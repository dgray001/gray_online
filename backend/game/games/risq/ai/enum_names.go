package ai

import (
	"fmt"
	"slices"
)

type enumValues map[string][]float64

var enumNames = map[string]enumValues{
	"order_kind":      orderKindEnum(),
	"unit_type":       unitTypeEnum(),
	"stance":          singleEnum(unitStanceNames),
	"attack_range":    {"none": {0}, "zone": {1}, "space": {2}, "adjacent": {3}, "secondary": {4}},
	"category":        {"food": {0}, "wood": {1}, "stone": {2}, "gold": {3}},
	"terrain_type":    {"none": {0}, "flatlands": {1}, "hilly": {2}, "mountainous": {3}, "swamp": {4}, "shallows": {5}, "water": {6}, "deep_water": {7}},
	"vision":          {"unexplored": {0}, "fog": {1}, "poor": {2}, "good": {3}, "spy": {4}},
	"target_category": singleEnum(targetCategoryNames),
}

func init() {
	enumNames["space_vision"] = enumNames["vision"]
	enumNames["resource_category"] = enumNames["category"]
}

func singleEnum[T ~uint8](names map[string]T) enumValues {
	values := make(enumValues, len(names))
	for name, value := range names {
		values[name] = []float64{float64(value)}
	}
	return values
}

func orderKindEnum() enumValues {
	values := singleEnum(orderKindNames)
	values["none"] = []float64{-1}
	return values
}

func unitTypeEnum() enumValues {
	values := singleEnum(unitTypeNames)
	for _, t := range unitTypesNamed("military") {
		values["military"] = append(values["military"], float64(t))
	}
	return values
}

type inFilter struct {
	field  string
	values []float64
}

func (f inFilter) allows(value float64) bool {
	return slices.Contains(f.values, value)
}

func parseInFilters(raw any, kind string, let_names []string) ([]inFilter, error) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("\"in\" must map field names to lists of values")
	}
	filters := make([]inFilter, 0, len(obj))
	for field, entry := range obj {
		if !hasItemField("select_"+kind, field) && !slices.Contains(let_names, field) {
			return nil, fmt.Errorf("in: %s items have no field %q", kind, field)
		}
		list, ok := entry.([]any)
		if !ok || len(list) == 0 {
			return nil, fmt.Errorf("in %s: needs a nonempty list of values", field)
		}
		filter := inFilter{field: field}
		for _, item := range list {
			values, err := enumValue(field, item)
			if err != nil {
				return nil, err
			}
			filter.values = append(filter.values, values...)
		}
		filters = append(filters, filter)
	}
	return filters, nil
}

func enumValue(field string, item any) ([]float64, error) {
	switch v := item.(type) {
	case float64:
		return []float64{v}, nil
	case string:
		if values, known := enumNames[field][v]; known {
			return values, nil
		}
		return nil, fmt.Errorf("in %s: unknown value %q", field, v)
	}
	return nil, fmt.Errorf("in %s: values must be numbers or names", field)
}
