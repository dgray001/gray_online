package mapgen

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

var uniformFoodIds = []uint32{1, 2}
var uniformTreeIds = []uint32{301, 302, 303, 304, 305, 306, 307, 308, 309, 310, 311, 312, 313}
var uniformGroveIds = []uint32{401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413}
var uniformForestIds = []uint32{501, 502, 503, 504, 505, 506, 507, 508, 509, 510, 511, 512, 513}
var uniformWoodIds = append(append(append([]uint32{}, uniformTreeIds...), uniformGroveIds...), uniformForestIds...)
var uniformStoneIds = []uint32{101, 102, 103}
var uniformGoldIds = []uint32{201}

func resourceZoneMatches(zone Zone, selector string) bool {
	return selector == "" || selector == "all" || (selector == "center" && zone.IsCenter()) || (selector == "edge" && !zone.IsCenter())
}

func resourceLocalMatches(local game_utils.Coordinate2D, selector string) bool {
	return selector == "" || selector == "all" || (selector == "center" && local.X == 0 && local.Y == 0) || (selector == "edge" && (local.X != 0 || local.Y != 0))
}

func validateResourceZone(selector string) bool {
	return selector == "" || selector == "all" || selector == "center" || selector == "edge"
}

type scatterEntry struct {
	ids    []uint32
	weight float64
}

// Fixed order keeps a seeded scatter reproducible regardless of json map ordering
var scatterCategoryPools = []struct {
	name string
	ids  []uint32
}{
	{"food", uniformFoodIds}, {"wood", uniformWoodIds}, {"grove", uniformGroveIds}, {"forest", uniformForestIds}, {"stone", uniformStoneIds},
	{"tree", uniformTreeIds},
}

func scatterEntries(config *defs.Config, category_weights map[string]float64, resource_weights map[uint32]float64) ([]scatterEntry, error) {
	entries := make([]scatterEntry, 0)
	known := make(map[string]bool, len(scatterCategoryPools))
	for _, pool := range scatterCategoryPools {
		known[pool.name] = true
		if w := category_weights[pool.name]; w > 0 {
			entries = append(entries, scatterEntry{pool.ids, w})
		}
	}
	for name := range category_weights {
		if !known[name] {
			return nil, fmt.Errorf("unknown resource category %q", name)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(resource_weights)) {
		if _, ok := config.Resources[id]; !ok {
			return nil, fmt.Errorf("unknown resource id %d", id)
		}
		if w := resource_weights[id]; w > 0 {
			entries = append(entries, scatterEntry{[]uint32{id}, w})
		}
	}
	return entries, nil
}

func pickScatterIds(entries []scatterEntry, rng *rand.Rand) []uint32 {
	total := 0.0
	for _, e := range entries {
		total += e.weight
	}
	if total <= 0 {
		return nil
	}
	roll := rng.Float64() * total
	for _, e := range entries {
		if roll < e.weight {
			return e.ids
		}
		roll -= e.weight
	}
	return entries[len(entries)-1].ids
}
