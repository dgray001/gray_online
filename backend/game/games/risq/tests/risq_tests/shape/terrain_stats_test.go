package shape

import (
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"testing"
)

func TestTerrainStatsWireValues(t *testing.T) {
	for name, terrainType := range map[string]defs.TerrainType{"flatlands": defs.TerrainType_FLATLANDS, "hilly": defs.TerrainType_HILLY, "mountainous": defs.TerrainType_MOUNTAINOUS, "water": defs.TerrainType_WATER} {
		t.Run(name, func(t *testing.T) {
			g := configuredGame(t, 3, func() {
				config := defs.TerrainConfigs[1]
				config.Terrain_type = terrainType
				defs.TerrainConfigs[1] = config
			})
			stats := checkShape(t, space(t, snapshot(g, g.Human(0), false), 0, 0)["terrain_stats"], "intra_cost:n inter_cost:n impassable:b build_speed_modifier:n")
			cost := terrainType.MoveCost()
			equal(t, stats["intra_cost"], float64(cost.Intra_cost))
			equal(t, stats["inter_cost"], float64(cost.Inter_cost))
			equal(t, stats["impassable"], cost.Impassable)
			equal(t, stats["build_speed_modifier"], cost.Build_speed_modifier)
		})
	}
}
