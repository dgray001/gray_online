package mapgen

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

type resourcePlaceParams struct {
	ResourceId      uint32             `json:"resource_id"`
	CategoryWeights map[string]float64 `json:"category_weights,omitempty"`
	ResourceWeights map[string]float64 `json:"resource_weights,omitempty"`
	Count           ScriptExpr         `json:"count"`
	// zones grouped per placement; resources land on up to this many adjacent zones
	ClumpSize ScriptExpr `json:"clump_size"`
	// hard floor on the space distance between this step's clumps
	MinSpacing ScriptExpr `json:"min_spacing"`
	// soft spacing target, relaxed down to min_spacing before the terrain preference is given up
	PreferredSpacing ScriptExpr `json:"preferred_spacing"`
	// chance each clump takes the zone farthest from earlier clumps instead of a random one
	Evenness ScriptExpr        `json:"evenness"`
	Zone     string            `json:"zone,omitempty"`
	On       terrainFilterJSON `json:"on"`
	// keeps the resources out of every player start area (requires an earlier player_starts step)
	OutsidePlayerAreas bool `json:"outside_player_areas"`
}

// Preferences give way in this order: preferred spacing (down to min_spacing), then the terrain filter, then the zone selector
const resourceTierCount = 3

type resourcePlacer struct {
	ctx       *mapScriptContext
	entries   []placerZone
	zone      string
	matches   func(Space) bool
	hard      int
	preferred int
	evenness  float64
	// space key to its distance from the nearest zone this step already placed; absent until one is placed
	nearest  map[uint]int
	insets   map[uint]int
	excluded map[uint]bool
	// scratch buffers reused across seed searches
	open, free []Zone
}

// A zone with the checks that cannot change during the step already evaluated
type placerZone struct {
	zone                        Zone
	usable, selected, onTerrain bool
}

func (r *resourcePlacer) classify(z Zone) placerZone {
	space := z.Space()
	return placerZone{zone: z, usable: !space.Impassable() && !r.excluded[space.Key()], selected: resourceZoneMatches(z, r.zone), onTerrain: r.matches(space)}
}

// A zone the resource may take at this tier; occupied, impassable and excluded zones are never allowed
func (r *resourcePlacer) entryAllowed(e placerZone, tier int) bool {
	if !e.usable || e.zone.Occupied() {
		return false
	}
	if tier < 2 && !e.selected {
		return false
	}
	return tier >= 1 || e.onTerrain
}

func (r *resourcePlacer) allowed(z Zone, tier int) bool {
	return r.entryAllowed(r.classify(z), tier)
}

func (r *resourcePlacer) spaced(z Zone, spacing int) bool {
	distance, placed := r.nearest[z.Space().Key()]
	return !placed || distance >= spacing
}

func (r *resourcePlacer) allowedZones(tier int) []Zone {
	r.open = r.open[:0]
	for _, e := range r.entries {
		if r.entryAllowed(e, tier) {
			r.open = append(r.open, e.zone)
		}
	}
	return r.open
}

func (r *resourcePlacer) spacedZones(open []Zone, spacing int) []Zone {
	r.free = r.free[:0]
	for _, z := range open {
		if r.spaced(z, spacing) {
			r.free = append(r.free, z)
		}
	}
	return r.free
}

// Places up to size resources on the zones around seed and returns how many landed
func (r *resourcePlacer) placeClump(seed Zone, size int, tier int, entries []scatterEntry) int {
	placed := []Zone{}
	for _, z := range growZoneBlob(seed, size, r.ctx.rng) {
		if len(placed) >= size || !r.allowed(z, tier) {
			continue
		}
		ids := pickScatterIds(entries, r.ctx.rng)
		r.ctx.board.PlaceResource(z, ids[r.ctx.rng.Intn(len(ids))])
		placed = append(placed, z)
	}
	r.recordPlaced(placed)
	return len(placed)
}

// Updates every space's distance to the nearest zone this step has placed
func (r *resourcePlacer) recordPlaced(zones []Zone) {
	sources := make(map[uint]Space, len(zones))
	for _, z := range zones {
		sources[z.Space().Key()] = z.Space()
	}
	for _, space := range r.ctx.allSpaces() {
		for _, source := range sources {
			distance := r.ctx.board.Distance(source, space)
			if current, placed := r.nearest[space.Key()]; !placed || distance < current {
				r.nearest[space.Key()] = distance
			}
		}
	}
}

// Resolves each key as an expression over script variables, summing weights of keys that land on one id
func resolveResourceWeights(ctx *mapScriptContext, weights map[string]float64) (map[uint32]float64, error) {
	resolved := make(map[uint32]float64, len(weights))
	for key, weight := range weights {
		value, err := util.EvalExpr(key, ctx.vars)
		if err != nil {
			return nil, fmt.Errorf("resource_weights key %q: %w", key, err)
		}
		resolved[uint32(math.Round(value))] += weight
	}
	return resolved, nil
}

func placeEntries(ctx *mapScriptContext, resource_id uint32, category_weights map[string]float64, resource_weights map[string]float64) ([]scatterEntry, error) {
	if resource_id != 0 {
		if _, ok := defs.ResourceConfigs[resource_id]; !ok {
			return nil, fmt.Errorf("unknown resource id %d", resource_id)
		}
		return []scatterEntry{{[]uint32{resource_id}, 1}}, nil
	}
	ids, err := resolveResourceWeights(ctx, resource_weights)
	if err != nil {
		return nil, err
	}
	entries, err := scatterEntries(category_weights, ids)
	if err == nil && len(entries) == 0 {
		err = fmt.Errorf("needs a resource_id or positive weights")
	}
	return entries, err
}

func playerAreaKeys(ctx *mapScriptContext) (map[uint]bool, error) {
	if len(ctx.player_starts) == 0 && !ctx.nomad_starts {
		return nil, fmt.Errorf("resource_place: outside_player_areas needs an earlier player_starts step")
	}
	keys := make(map[uint]bool)
	for _, start := range ctx.player_starts {
		for _, space := range hexRadiusSpaces(start.space, ctx.player_area_size) {
			keys[space.Key()] = true
		}
	}
	return keys, nil
}

// With probability evenness the zone farthest from earlier clumps and, more softly, the map edge; otherwise a random one
func (r *resourcePlacer) pick(free []Zone) Zone {
	if len(r.nearest) == 0 || r.ctx.rng.Float64() >= r.evenness {
		return free[r.ctx.rng.Intn(len(free))]
	}
	var best []Zone
	farthest := -1.0
	for _, z := range free {
		key := z.Space().Key()
		score := min(blobEdgeSoftening*float64(r.insets[key]+1), float64(r.nearest[key]))
		if score > farthest {
			best, farthest = best[:0], score
		}
		if score == farthest {
			best = append(best, z)
		}
	}
	return best[r.ctx.rng.Intn(len(best))]
}

// The next clump seed: every preference at the preferred spacing first, then spacing relaxes to the hard floor
// before the terrain filter goes, and the zone selector goes last; false when no free zone is left
func (r *resourcePlacer) nextSeed() (Zone, int, bool) {
	for tier := 0; tier < resourceTierCount; tier++ {
		open := r.allowedZones(tier)
		for spacing := max(r.preferred, r.hard); len(open) > 0 && spacing >= r.hard; spacing-- {
			if free := r.spacedZones(open, spacing); len(free) > 0 {
				return r.pick(free), tier, true
			}
		}
	}
	return nil, 0, false
}

func newResourcePlacer(ctx *mapScriptContext, p resourcePlaceParams) (*resourcePlacer, error) {
	hard, preferred, err := resolveBlobSpacing(ctx, p.MinSpacing, p.PreferredSpacing)
	if err != nil {
		return nil, err
	}
	evenness, err := p.Evenness.resolve(ctx.vars)
	if err != nil {
		return nil, err
	}
	if evenness < 0 || evenness > 1 {
		return nil, fmt.Errorf("resource_place: evenness must be in [0, 1], got %v", evenness)
	}
	matches, err := p.On.matcher(ctx)
	if err != nil {
		return nil, err
	}
	r := &resourcePlacer{ctx: ctx, zone: p.Zone, matches: matches, hard: hard, preferred: preferred, evenness: evenness,
		nearest: make(map[uint]int), insets: ctx.edgeInsets(), excluded: make(map[uint]bool)}
	if p.OutsidePlayerAreas {
		if r.excluded, err = playerAreaKeys(ctx); err != nil {
			return nil, err
		}
	}
	zones := ctx.allZones()
	r.entries = make([]placerZone, len(zones))
	for i, z := range zones {
		r.entries[i] = r.classify(z)
	}
	return r, nil
}

// Places count resources, each preference relaxed only when no free zone satisfies it
func stepResourcePlace(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[resourcePlaceParams](raw, "resource_place")
	if err != nil {
		return err
	}
	if !validateResourceZone(p.Zone) {
		return fmt.Errorf("resource_place: invalid zone %q", p.Zone)
	}
	entries, err := placeEntries(ctx, p.ResourceId, p.CategoryWeights, p.ResourceWeights)
	if err != nil {
		return fmt.Errorf("resource_place: %w", err)
	}
	count, err := p.Count.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	clump, err := p.ClumpSize.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	placer, err := newResourcePlacer(ctx, p)
	if err != nil {
		return err
	}
	for placed := 0; placed < count; {
		seed, tier, ok := placer.nextSeed()
		if !ok {
			break
		}
		placed += placer.placeClump(seed, min(max(clump, 1), count-placed), tier, entries)
	}
	return nil
}
