package ai

import "fmt"

func (s *selectorSpec) takeExtras(where map[string]any) (map[string]any, error) {
	if where == nil {
		return map[string]any{}, nil
	}
	if err := s.takeConditions(where); err != nil {
		return nil, err
	}
	if err := s.takeSpatial(where); err != nil {
		return nil, err
	}
	if err := s.takeProduction(where); err != nil {
		return nil, err
	}
	if raw, present := where["in"]; present {
		filters, err := parseInFilters(raw, s.kind, s.letNames())
		if err != nil {
			return nil, err
		}
		s.in = filters
		delete(where, "in")
	}
	for key, target := range map[string]*string{"in_set": &s.in_set, "not_in_set": &s.not_in_set, "bucket": &s.bucket, "region_in_set": &s.region_in_set} {
		if raw, present := where[key]; present {
			name, ok := raw.(string)
			if !ok || name == "" {
				return nil, fmt.Errorf("%q must be a nonempty name", key)
			}
			if key == "bucket" && s.kind != "units" {
				return nil, fmt.Errorf("\"bucket\" only filters units")
			}
			*target = name
			delete(where, key)
		}
	}
	return where, nil
}

func (s *selectorSpec) takeConditions(where map[string]any) error {
	raw, present := where["if"]
	if !present {
		return nil
	}
	delete(where, "if")
	list, isList := raw.([]any)
	if !isList {
		list = []any{raw}
	}
	for _, entry := range list {
		a, err := parseAmount(entry)
		if err != nil {
			return fmt.Errorf("if: %v", err)
		}
		s.conditions = append(s.conditions, a)
	}
	return nil
}

func parseNear(raw any) (*nearSpec, error) {
	obj, ok := raw.(map[string]any)
	if !ok || obj["of"] == nil {
		return nil, fmt.Errorf("near must be an object (or list of objects) with \"distance\" and \"of\"")
	}
	if err := checkKeys(obj, "distance", "unit", "of"); err != nil {
		return nil, fmt.Errorf("near: %w", err)
	}
	if err := checkOptionTypes[string](obj, "unit"); err != nil {
		return nil, fmt.Errorf("near: %w", err)
	}
	distance, err := parseNumber(obj, "distance", 0)
	if err != nil {
		return nil, fmt.Errorf("near: %v", err)
	}
	near := &nearSpec{distance: distance}
	err = withScopeBarrier("near.of", func() (parsed error) {
		origin, ok := obj["of"].(map[string]any)
		if !ok {
			return fmt.Errorf("near.of must be a selector or coordinate object")
		}
		if _, selector := origin["kind"]; selector {
			near.of, parsed = parseSelectorSpec(origin, "anchor")
		} else {
			near.coordinate, parsed = parseCoordinateSpec(origin)
		}
		return parsed
	})
	if err != nil {
		return nil, fmt.Errorf("near: %v", err)
	}
	unit, _ := obj["unit"].(string)
	if _, present := obj["unit"]; present && unit != "space" && unit != "zone" {
		return nil, fmt.Errorf("near unit must be \"space\" or \"zone\"")
	}
	near.zone_steps = unit == "zone"
	return near, nil
}

func (s *selectorSpec) takeSpatial(where map[string]any) error {
	if raw, present := where["near"]; present {
		entries, is_list := raw.([]any)
		if !is_list {
			entries = []any{raw}
		}
		for _, entry := range entries {
			near, err := parseNear(entry)
			if err != nil {
				return err
			}
			s.near = append(s.near, near)
		}
		delete(where, "near")
	}
	if raw, present := where["at"]; present {
		var at *selectorSpec
		err := withScopeBarrier("at", func() (parsed error) {
			at, parsed = parseSelectorSpec(map[string]any{"kind": "spaces", "where": raw}, "place")
			return parsed
		})
		if err != nil {
			return fmt.Errorf("at: %v", err)
		}
		s.at = at
		delete(where, "at")
	}
	return nil
}
