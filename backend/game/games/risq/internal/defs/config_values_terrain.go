package defs

var resourceFields = fieldTable[ResourceConfig]{
	"category":           func(r ResourceConfig) float64 { return float64(r.Category) },
	"starting_resources": func(r ResourceConfig) float64 { return r.Starting_resources },
	"gather_speed":       func(r ResourceConfig) float64 { return float64(r.Base_gather_speed) },
	"gather_capacity":    func(r ResourceConfig) float64 { return float64(r.Gather_capacity) },
}

type terrainValues struct {
	TerrainConfig
	TerrainMoveCost
}

func terrainEntry(c *Config, id uint32) (terrainValues, bool) {
	t, ok := c.Terrains[id]
	return terrainValues{t, c.MoveCost(t.Terrain_type)}, ok
}

var terrainFields = fieldTable[terrainValues]{
	"terrain_type":         func(t terrainValues) float64 { return float64(t.Terrain_type) },
	"intra_cost":           func(t terrainValues) float64 { return float64(t.Intra_cost) },
	"inter_cost":           func(t terrainValues) float64 { return float64(t.Inter_cost) },
	"impassable":           func(t terrainValues) float64 { return boolValue(t.Impassable) },
	"build_speed_modifier": func(t terrainValues) float64 { return t.Build_speed_modifier },
}
