package mapgen

import (
	"fmt"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

// Restricts a step to spaces whose current terrain is listed by id, display name or terrain type; empty matches everything
type terrainFilterJSON struct {
	TerrainIds   []ScriptExpr `json:"terrain_ids"`
	TerrainNames []string     `json:"terrain_names"`
	TerrainTypes []string     `json:"terrain_types"`
}

func (f terrainFilterJSON) empty() bool {
	return len(f.TerrainIds) == 0 && len(f.TerrainNames) == 0 && len(f.TerrainTypes) == 0
}

func (f terrainFilterJSON) allowedIds(ctx *mapScriptContext) (map[uint32]bool, error) {
	allowed := make(map[uint32]bool)
	for _, expr := range f.TerrainIds {
		id, err := resolveTerrainId(expr, ctx)
		if err != nil {
			return nil, fmt.Errorf("terrain filter: %w", err)
		}
		for _, member := range defs.TerrainFamily(id) {
			allowed[member] = true
		}
	}
	for _, name := range f.TerrainNames {
		found := false
		for id, config := range defs.TerrainConfigs {
			if config.Display_name == name {
				allowed[id], found = true, true
			}
		}
		if !found {
			return nil, fmt.Errorf("terrain filter: unknown terrain name %q", name)
		}
	}
	return allowed, f.addTypes(allowed)
}

func (f terrainFilterJSON) addTypes(allowed map[uint32]bool) error {
	for _, name := range f.TerrainTypes {
		terrain_type, err := defs.ParseTerrainType(name)
		if err != nil {
			return err
		}
		for id, config := range defs.TerrainConfigs {
			if config.Terrain_type == terrain_type {
				allowed[id] = true
			}
		}
	}
	return nil
}

func filterSpaces(spaces []Space, matches func(Space) bool) []Space {
	kept := make([]Space, 0, len(spaces))
	for _, space := range spaces {
		if matches(space) {
			kept = append(kept, space)
		}
	}
	return kept
}

func (f terrainFilterJSON) matcher(ctx *mapScriptContext) (func(Space) bool, error) {
	if f.empty() {
		return func(Space) bool { return true }, nil
	}
	allowed, err := f.allowedIds(ctx)
	if err != nil {
		return nil, err
	}
	return func(space Space) bool { return allowed[space.Terrain()] }, nil
}
