package mapgen

import (
	"encoding/json"
	"fmt"
	"maps"
	"math/rand"
	"slices"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

var uniformFoodIds = []uint32{1, 2}
var uniformTreeIds = []uint32{11, 12, 13, 14, 15, 16}
var uniformGroveIds = []uint32{21, 22, 23, 24, 25, 26}
var uniformForestIds = []uint32{31, 32, 33, 34, 35, 36}
var uniformWoodIds = append(append(append([]uint32{}, uniformTreeIds...), uniformGroveIds...), uniformForestIds...)
var uniformStoneIds = []uint32{41, 42, 43}
var uniformGoldIds = []uint32{51}

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

func scatterEntries(p resourceScatterParams) ([]scatterEntry, error) {
	entries := make([]scatterEntry, 0)
	known := make(map[string]bool, len(scatterCategoryPools))
	for _, pool := range scatterCategoryPools {
		known[pool.name] = true
		if w := p.CategoryWeights[pool.name]; w > 0 {
			entries = append(entries, scatterEntry{pool.ids, w})
		}
	}
	for name := range p.CategoryWeights {
		if !known[name] {
			return nil, fmt.Errorf("unknown resource_scatter category %q", name)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(p.ResourceWeights)) {
		if _, ok := defs.ResourceConfigs[id]; !ok {
			return nil, fmt.Errorf("unknown resource_scatter resource id %d", id)
		}
		if w := p.ResourceWeights[id]; w > 0 {
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

type resourceScatterParams struct {
	Chance ScriptExpr `json:"chance"`
	Zone   string     `json:"zone,omitempty"`
	// keys: food, wood (trees + groves + forests), tree, grove, forest, stone
	CategoryWeights map[string]float64 `json:"category_weights,omitempty"`
	// exact resource ids, weighted against the categories
	ResourceWeights map[uint32]float64 `json:"resource_weights,omitempty"`
}

func stepResourceScatter(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[resourceScatterParams](raw, "resource_scatter")
	if err != nil {
		return err
	}
	if !validateResourceZone(p.Zone) {
		return fmt.Errorf("resource_scatter: invalid zone %q", p.Zone)
	}
	chance, err := p.Chance.resolve(ctx.vars)
	if err != nil {
		return err
	}
	entries, err := scatterEntries(p)
	if err != nil {
		return err
	}
	for _, space := range ctx.allSpaces() {
		for _, zone := range util.ShuffleFrom(ctx.rng, space.Zones()) {
			if zone.Occupied() || !resourceZoneMatches(zone, p.Zone) {
				continue
			}
			if ctx.rng.Float64() >= chance {
				continue
			}
			ids := pickScatterIds(entries, ctx.rng)
			if len(ids) == 0 {
				continue
			}
			resource_id := ids[ctx.rng.Intn(len(ids))]
			ctx.board.PlaceResource(zone, resource_id)
		}
	}
	return nil
}

type resourceClusterParams struct {
	ResourceId uint32     `json:"resource_id"`
	SeedCount  ScriptExpr `json:"seed_count"`
	Size       ScriptExpr `json:"size"`
	Zone       string     `json:"zone,omitempty"`
}

func stepResourceCluster(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[resourceClusterParams](raw, "resource_cluster")
	if err != nil {
		return err
	}
	if !validateResourceZone(p.Zone) {
		return fmt.Errorf("resource_cluster: invalid zone %q", p.Zone)
	}
	if _, ok := defs.ResourceConfigs[p.ResourceId]; !ok {
		return fmt.Errorf("resource_cluster: unknown resource id %d", p.ResourceId)
	}
	seed_count, err := p.SeedCount.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	size, err := p.Size.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	candidates := make([]Zone, 0)
	for _, z := range ctx.allZones() {
		if !z.Occupied() && resourceZoneMatches(z, p.Zone) {
			candidates = append(candidates, z)
		}
	}
	for i := 0; i < seed_count && len(candidates) > 0; i++ {
		seed := candidates[ctx.rng.Intn(len(candidates))]
		for _, z := range growZoneBlob(seed, size, ctx.rng) {
			if z.Occupied() || !resourceZoneMatches(z, p.Zone) {
				continue
			}
			ctx.board.PlaceResource(z, p.ResourceId)
		}
	}
	return nil
}

type resourceMinSpacingParams struct {
	Distance ScriptExpr `json:"distance"`
}

func stepResourceMinSpacing(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[resourceMinSpacingParams](raw, "resource_min_spacing")
	if err != nil {
		return err
	}
	distance, err := p.Distance.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	resource_zones := make([]Zone, 0)
	for _, z := range ctx.allZones() {
		if _, ok := z.ResourceId(); ok {
			resource_zones = append(resource_zones, z)
		}
	}
	kept := make([]Zone, 0, len(resource_zones))
	for _, z := range util.ShuffleFrom(ctx.rng, resource_zones) {
		too_close := false
		for _, k := range kept {
			if int(game_utils.AxialDistance(z.Space().Coordinate(), k.Space().Coordinate())) < distance {
				too_close = true
				break
			}
		}
		if too_close {
			z.RemoveResource()
			continue
		}
		kept = append(kept, z)
	}
	return nil
}

type resourcePlaceParams struct {
	ResourceId uint32     `json:"resource_id"`
	Count      ScriptExpr `json:"count"`
	Zone       string     `json:"zone,omitempty"`
	// keeps the resources out of every player start area (requires an earlier player_starts step)
	OutsidePlayerAreas bool `json:"outside_player_areas"`
}

// Places exactly count of one resource on random free zones, so every seed gets the same amount
func stepResourcePlace(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[resourcePlaceParams](raw, "resource_place")
	if err != nil {
		return err
	}
	if !validateResourceZone(p.Zone) {
		return fmt.Errorf("resource_place: invalid zone %q", p.Zone)
	}
	if _, ok := defs.ResourceConfigs[p.ResourceId]; !ok {
		return fmt.Errorf("resource_place: unknown resource id %d", p.ResourceId)
	}
	count, err := p.Count.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	if count <= 0 {
		return nil
	}
	excluded := make(map[uint]bool)
	if p.OutsidePlayerAreas {
		if len(ctx.player_starts) == 0 {
			return fmt.Errorf("resource_place: outside_player_areas needs an earlier player_starts step")
		}
		for _, start := range ctx.player_starts {
			for _, s := range hexRadiusSpaces(start.space, ctx.player_area_size) {
				excluded[s.Key()] = true
			}
		}
	}
	candidates := make([]Zone, 0)
	for _, z := range ctx.allZones() {
		if z.Occupied() || z.Space().Impassable() || excluded[z.Space().Key()] || !resourceZoneMatches(z, p.Zone) {
			continue
		}
		candidates = append(candidates, z)
	}
	if len(candidates) < count {
		return fmt.Errorf("resource_place: only %d free zones for %d of resource %d", len(candidates), count, p.ResourceId)
	}
	for _, z := range util.ShuffleFrom(ctx.rng, candidates)[:count] {
		ctx.board.PlaceResource(z, p.ResourceId)
	}
	return nil
}
