package ai

import "fmt"

type coordinateSpec struct {
	x, y, zone_x, zone_y amount
}

func parseCoordinateSpec(obj map[string]any) (*coordinateSpec, error) {
	if err := checkKeys(obj, "x", "y", "zone_x", "zone_y"); err != nil {
		return nil, err
	}
	if obj["x"] == nil || obj["y"] == nil {
		return nil, fmt.Errorf("coordinate needs \"x\" and \"y\"")
	}
	_, has_x := obj["zone_x"]
	_, has_y := obj["zone_y"]
	if has_x != has_y {
		return nil, fmt.Errorf("coordinate needs both \"zone_x\" and \"zone_y\"")
	}
	return parseCoordinateFields(obj)
}

func parseCoordinateFields(obj map[string]any) (*coordinateSpec, error) {
	c := &coordinateSpec{}
	for _, field := range []struct {
		name  string
		value *amount
	}{{"x", &c.x}, {"y", &c.y}, {"zone_x", &c.zone_x}, {"zone_y", &c.zone_y}} {
		value, err := parseNumber(obj, field.name, 0)
		if err != nil {
			return nil, err
		}
		*field.value = value
	}
	return c, nil
}

func (c *coordinateSpec) resolve(view View, internals *Internals) ZoneRef {
	return ZoneRef{
		Space: Coordinate{X: c.x.int(view, internals), Y: c.y.int(view, internals)},
		Zone:  Coordinate{X: c.zone_x.int(view, internals), Y: c.zone_y.int(view, internals)},
	}
}
