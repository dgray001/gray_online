package ai

import (
	"slices"
	"strings"
)

// Every field of these tables is readable as var(role.<path>.<field>) on items standing in that place
var locationPaths = map[string]func() []string{
	"space":  spaceFields.names,
	"zone":   zoneFields.names,
	"region": regionFields.names,
}

var locatedKinds = []string{"units", "buildings", "resources", "zones", "foundations", "gatherables", "attackables"}

func kindPaths(kind string) []string {
	switch {
	case slices.Contains(locatedKinds, kind):
		return []string{"space", "zone", "region"}
	case kind == "spaces":
		return []string{"region"}
	}
	return nil
}

func pathFieldNames(kind string) []string {
	names := []string{}
	for _, path := range kindPaths(kind) {
		for _, field := range locationPaths[path]() {
			names = append(names, path+"."+field)
		}
	}
	return names
}

// nested is false when name is not a path, so the caller reads it as a plain field
func (s selItem) pathField(name string) (value float64, ok bool, nested bool) {
	path, field, nested := strings.Cut(name, ".")
	if !nested {
		return 0, false, false
	}
	if item, located := s.pathItem(path); located {
		value, ok = item.field(field)
	}
	return value, ok, true
}

// Built the way the spaces, zones and regions sources build their items, so both read the same values
func (s selItem) pathItem(path string) (loopItem, bool) {
	loc, _, located := locOf(s.loopItem)
	if !located || s.census == nil {
		return nil, false
	}
	switch path {
	case "space":
		return newSpaceItem(loc.Space, s.census.spaces[loc.Space], s.census.ownUnitsBySpace(s.view)), true
	case "zone":
		s.census.loadZones(s.view)
		if position, known := s.census.zone_position[loc]; known {
			return newZoneItem(s.census.zones[position]), true
		}
		return newZoneItem(ZoneInfo{Location: loc, BuildingPlayer: -1}), true
	case "region":
		stat, _ := s.census.regions.at(loc.Space)
		return regionItem{stat: stat}, true
	}
	return nil, false
}
