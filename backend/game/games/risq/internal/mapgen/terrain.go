package mapgen

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type weightedTerrainIdJSON struct {
	TerrainId ScriptExpr `json:"terrain_id"`
	Weight    ScriptExpr `json:"weight"`
}

type terrainPickJSON struct {
	TerrainType string                  `json:"terrain_type,omitempty"`
	TerrainId   ScriptExpr              `json:"terrain_id,omitempty"`
	TerrainIds  []weightedTerrainIdJSON `json:"terrain_ids,omitempty"`
}

func resolveTerrainId(expr ScriptExpr, ctx *mapScriptContext) (uint32, error) {
	value, err := expr.resolveInt(ctx.vars)
	if err != nil {
		return 0, err
	}
	if _, ok := ctx.config().Terrains[uint32(value)]; !ok {
		return 0, fmt.Errorf("unknown terrain id %d", value)
	}
	return uint32(value), nil
}

func (p terrainPickJSON) resolve(ctx *mapScriptContext) (uint32, error) {
	rng := ctx.rng
	if len(p.TerrainIds) > 0 {
		ids, weights, total := make([]uint32, len(p.TerrainIds)), make([]float64, len(p.TerrainIds)), 0.0
		for i, w := range p.TerrainIds {
			id, err := resolveTerrainId(w.TerrainId, ctx)
			if err != nil {
				return 0, fmt.Errorf("terrain_ids: %w", err)
			}
			weight, err := w.Weight.resolve(ctx.vars)
			if err != nil {
				return 0, err
			}
			ids[i], weights[i], total = id, weight, total+weight
		}
		if total <= 0 {
			return 0, fmt.Errorf("terrain_ids weights must sum to more than 0")
		}
		roll := rng.Float64() * total
		for i, weight := range weights {
			if roll < weight {
				return ids[i], nil
			}
			roll -= weight
		}
		return ids[len(ids)-1], nil
	}
	if p.TerrainId.provided() {
		id, err := resolveTerrainId(p.TerrainId, ctx)
		if err != nil {
			return 0, fmt.Errorf("terrain_id: %w", err)
		}
		return id, nil
	}
	if p.TerrainType != "" {
		terrain_type, err := defs.ParseTerrainType(p.TerrainType)
		if err != nil {
			return 0, err
		}
		return ctx.config().RandomTerrainId(terrain_type, rng), nil
	}
	return 0, fmt.Errorf("must specify terrain_type, terrain_id, or terrain_ids")
}

const defaultBlobRoundness = 0.6

// With probability roundness picks among the candidates nearest the seed, preferring those touching the most blob cells; else uniformly among all
func pickCandidateIndex[T any](candidates []T, distance func(T) int, touching func(T) int, roundness float64, rng *rand.Rand) int {
	if rng.Float64() >= roundness {
		return rng.Intn(len(candidates))
	}
	nearest := distance(candidates[0])
	for _, c := range candidates[1:] {
		nearest = min(nearest, distance(c))
	}
	pool, most := make([]int, 0, len(candidates)), 0
	for i, c := range candidates {
		if distance(c) != nearest {
			continue
		}
		if t := touching(c); t > most {
			pool, most = pool[:0], t
		}
		if touching(c) == most {
			pool = append(pool, i)
		}
	}
	return pool[rng.Intn(len(pool))]
}

func bfsDistances[T any](start T, limit int, key func(T) uint, neighbors func(T) []T) map[uint]int {
	dist := map[uint]int{key(start): 0}
	queue := []T{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if dist[key(cur)] >= limit {
			continue
		}
		for _, n := range neighbors(cur) {
			if _, seen := dist[key(n)]; !seen {
				dist[key(n)] = dist[key(cur)] + 1
				queue = append(queue, n)
			}
		}
	}
	return dist
}

func growBlob[T any](start T, size int, roundness float64, rng *rand.Rand, key func(T) uint, neighbors func(T) []T, excluded func(T) bool) []T {
	dist := bfsDistances(start, size, key, neighbors)
	distance := func(c T) int {
		if d, ok := dist[key(c)]; ok {
			return d
		}
		return size + 1
	}
	seen := map[uint]bool{key(start): true}
	members := map[uint]bool{key(start): true}
	touching := func(c T) int {
		count := 0
		for _, n := range neighbors(c) {
			if members[key(n)] {
				count++
			}
		}
		return count
	}
	result := []T{start}
	candidates := []T{}
	addCandidates := func(cell T) {
		for _, n := range neighbors(cell) {
			if !excluded(n) && !seen[key(n)] {
				seen[key(n)] = true
				candidates = append(candidates, n)
			}
		}
	}
	addCandidates(start)
	for len(result) < size && len(candidates) > 0 {
		idx := pickCandidateIndex(candidates, distance, touching, roundness, rng)
		cell := candidates[idx]
		candidates = append(candidates[:idx], candidates[idx+1:]...)
		result = append(result, cell)
		members[key(cell)] = true
		addCandidates(cell)
	}
	return result
}

func growSpaceBlob(start Space, size int, roundness float64, excluded func(Space) bool, rng *rand.Rand) []Space {
	return growBlob(start, size, roundness, rng,
		func(s Space) uint { return s.Key() },
		func(s Space) []Space { return s.SortedAdjacent() },
		excluded,
	)
}

func growZoneBlob(start Zone, size int, rng *rand.Rand) []Zone {
	return growBlob(start, size, defaultBlobRoundness, rng,
		func(z Zone) uint { return z.Key() },
		func(z Zone) []Zone { return z.Adjacent() },
		func(Zone) bool { return false },
	)
}

type terrainFillParams struct {
	terrainPickJSON
	On     terrainFilterJSON `json:"on"`
	Region string            `json:"region,omitempty"`
}

func stepTerrainFill(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[terrainFillParams](raw, "terrain_fill")
	if err != nil {
		return err
	}
	region := ctx.region(p.Region)
	matches, err := p.On.matcher(ctx)
	if err != nil {
		return err
	}
	for _, space := range ctx.allSpaces() {
		if !matches(space) {
			continue
		}
		terrain_id, err := p.resolve(ctx)
		if err != nil {
			return err
		}
		space.SetTerrain(terrain_id)
		if region != nil {
			region[space.Key()] = true
		}
	}
	return nil
}

type terrainBlobParams struct {
	terrainPickJSON
	On         terrainFilterJSON `json:"on"`
	SeedCount  ScriptExpr        `json:"seed_count"`
	Size       BlobSize          `json:"size"`
	MinSpacing ScriptExpr        `json:"min_spacing"`
	Roundness  ScriptExpr        `json:"roundness"`
	// seeds first try this spacing, then relax down to min_spacing (the hard floor)
	PreferredSpacing ScriptExpr        `json:"preferred_spacing"`
	Placement        blobPlacementJSON `json:"placement"`
	Region           string            `json:"region,omitempty"`
	// "hilly" or "mountainous": converts each space to that version of its terrain, skipping terrain without one
	Variant string `json:"variant,omitempty"`
}

// The terrain type a variant blob converts spaces to, or NONE when the blob paints terrain_id or terrain_ids instead
func (p terrainBlobParams) variantKind() (defs.TerrainType, error) {
	if p.Variant == "" {
		return defs.TerrainType_NONE, nil
	}
	if p.TerrainId.provided() || len(p.TerrainIds) > 0 || p.TerrainType != "" {
		return defs.TerrainType_NONE, fmt.Errorf("terrain_blob: variant can't be combined with terrain_id, terrain_ids or terrain_type")
	}
	kind, err := defs.ParseTerrainType(p.Variant)
	if err != nil || (kind != defs.TerrainType_HILLY && kind != defs.TerrainType_MOUNTAINOUS) {
		return defs.TerrainType_NONE, fmt.Errorf("terrain_blob: variant must be \"hilly\" or \"mountainous\", got %q", p.Variant)
	}
	return kind, nil
}

func stepTerrainBlob(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[terrainBlobParams](raw, "terrain_blob")
	if err != nil {
		return err
	}
	variant, err := p.variantKind()
	if err != nil {
		return err
	}
	seed_count, err := p.SeedCount.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	min_size, max_size, err := p.Size.resolve(ctx.vars)
	if err != nil {
		return err
	}
	min_spacing, preferred_spacing, err := resolveBlobSpacing(ctx, p.MinSpacing, p.PreferredSpacing)
	if err != nil {
		return err
	}
	if seed_count < 0 {
		return fmt.Errorf("map script terrain_blob: seed_count must be non-negative, got %d", seed_count)
	}
	roundness := defaultBlobRoundness
	if p.Roundness.provided() {
		if roundness, err = p.Roundness.resolve(ctx.vars); err != nil {
			return err
		}
		if roundness < 0 || roundness > 1 {
			return fmt.Errorf("map script terrain_blob: roundness must be in [0, 1], got %v", roundness)
		}
	}
	candidates, err := blobSeedCandidates(ctx, p.Placement)
	if err != nil {
		return err
	}
	matches, err := p.On.matcher(ctx)
	if err != nil {
		return err
	}
	if variant != defs.TerrainType_NONE {
		filter := matches
		matches = func(space Space) bool {
			_, convertible := ctx.config().TerrainVariant(space.Terrain(), variant)
			return convertible && filter(space)
		}
	}
	seeds, err := blobSeeds(ctx, p.Placement, filterSpaces(candidates, matches), seed_count, min_spacing, preferred_spacing)
	if err != nil {
		return err
	}
	inside := blobGrowthLimit(candidates, p.Placement.KeepInside)
	excluded := func(space Space) bool { return !matches(space) || inside(space) }
	region := ctx.region(p.Region)
	for _, seed := range seeds {
		blob := growSpaceBlob(seed, pickBlobSize(ctx.rng, min_size, max_size), roundness, excluded, ctx.rng)
		for _, space := range blob {
			var terrain_id uint32
			if variant != defs.TerrainType_NONE {
				terrain_id, _ = ctx.config().TerrainVariant(space.Terrain(), variant)
			} else if terrain_id, err = p.resolve(ctx); err != nil {
				return err
			}
			space.SetTerrain(terrain_id)
			if region != nil {
				region[space.Key()] = true
			}
		}
	}
	return nil
}

type axialParamJSON struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func shortestPathToSet(from Space, target map[uint]bool) []Space {
	prev := map[uint]Space{from.Key(): nil}
	queue := []Space{from}
	var end Space
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if target[cur.Key()] {
			end = cur
			break
		}
		for _, adj := range cur.SortedAdjacent() {
			if _, seen := prev[adj.Key()]; seen {
				continue
			}
			prev[adj.Key()] = cur
			queue = append(queue, adj)
		}
	}
	if end == nil {
		return nil
	}
	path := make([]Space, 0)
	for cell := prev[end.Key()]; cell != nil; cell = prev[cell.Key()] {
		path = append(path, cell)
	}
	return path
}

type ensureConnectivityParams struct {
	terrainPickJSON
}

func stepEnsureConnectivity(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[ensureConnectivityParams](raw, "ensure_connectivity")
	if err != nil {
		return err
	}
	all := ctx.allSpaces()
	var start Space
	for _, s := range all {
		if !s.Impassable() {
			start = s
			break
		}
	}
	if start == nil {
		return nil
	}
	reached := map[uint]bool{start.Key(): true}
	queue := []Space{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, adj := range cur.SortedAdjacent() {
			if adj.Impassable() || reached[adj.Key()] {
				continue
			}
			reached[adj.Key()] = true
			queue = append(queue, adj)
		}
	}
	for _, s := range all {
		if s.Impassable() || reached[s.Key()] {
			continue
		}
		for _, cell := range shortestPathToSet(s, reached) {
			if reached[cell.Key()] {
				continue
			}
			terrain_id, err := p.resolve(ctx)
			if err != nil {
				return err
			}
			cell.SetTerrain(terrain_id)
			reached[cell.Key()] = true
		}
	}
	return nil
}
