package mapgen

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

type weightedTerrainIdJSON struct {
	TerrainId uint32  `json:"terrain_id"`
	Weight    float64 `json:"weight"`
}

type terrainPickJSON struct {
	TerrainType string                  `json:"terrain_type,omitempty"`
	TerrainId   uint32                  `json:"terrain_id,omitempty"`
	TerrainIds  []weightedTerrainIdJSON `json:"terrain_ids,omitempty"`
}

func (p terrainPickJSON) resolve(rng *rand.Rand) (uint32, error) {
	if len(p.TerrainIds) > 0 {
		total := 0.0
		for _, w := range p.TerrainIds {
			if _, ok := defs.TerrainConfigs[w.TerrainId]; !ok {
				return 0, fmt.Errorf("terrain_ids: unknown terrain id %d", w.TerrainId)
			}
			total += w.Weight
		}
		if total <= 0 {
			return 0, fmt.Errorf("terrain_ids weights must sum to more than 0")
		}
		roll := rng.Float64() * total
		for _, w := range p.TerrainIds {
			if roll < w.Weight {
				return w.TerrainId, nil
			}
			roll -= w.Weight
		}
		return p.TerrainIds[len(p.TerrainIds)-1].TerrainId, nil
	}
	if p.TerrainId != 0 {
		if _, ok := defs.TerrainConfigs[p.TerrainId]; !ok {
			return 0, fmt.Errorf("terrain_id: unknown terrain id %d", p.TerrainId)
		}
		return p.TerrainId, nil
	}
	if p.TerrainType != "" {
		terrain_type, err := defs.ParseTerrainType(p.TerrainType)
		if err != nil {
			return 0, err
		}
		return defs.RandomTerrainId(terrain_type, rng), nil
	}
	return 0, fmt.Errorf("must specify terrain_type, terrain_id, or terrain_ids")
}

func growBlob[T any](start T, size int, rng *rand.Rand, key func(T) uint, neighbors func(T) []T, excluded func(T) bool) []T {
	seen := map[uint]bool{key(start): true}
	result := []T{start}
	frontier := []T{start}
	for len(result) < size && len(frontier) > 0 {
		idx := rng.Intn(len(frontier))
		cur := frontier[idx]
		frontier = append(frontier[:idx], frontier[idx+1:]...)
		for _, n := range util.ShuffleFrom(rng, neighbors(cur)) {
			if len(result) >= size {
				break
			}
			if excluded(n) || seen[key(n)] {
				continue
			}
			seen[key(n)] = true
			result = append(result, n)
			frontier = append(frontier, n)
		}
	}
	return result
}

func growSpaceBlob(start Space, size int, rng *rand.Rand) []Space {
	return growBlob(start, size, rng,
		func(s Space) uint { return s.Key() },
		func(s Space) []Space { return s.SortedAdjacent() },
		func(Space) bool { return false },
	)
}

func growZoneBlob(start Zone, size int, rng *rand.Rand) []Zone {
	return growBlob(start, size, rng,
		func(z Zone) uint { return z.Key() },
		func(z Zone) []Zone { return z.Adjacent() },
		func(Zone) bool { return false },
	)
}

type terrainFillParams struct {
	terrainPickJSON
	Region string `json:"region,omitempty"`
}

func stepTerrainFill(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[terrainFillParams](raw, "terrain_fill")
	if err != nil {
		return err
	}
	region := ctx.region(p.Region)
	for _, space := range ctx.allSpaces() {
		terrain_id, err := p.resolve(ctx.rng)
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
	SeedCount  ScriptExpr `json:"seed_count"`
	Size       ScriptExpr `json:"size"`
	MinSpacing ScriptExpr `json:"min_spacing"`
	Region     string     `json:"region,omitempty"`
}

func stepTerrainBlob(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[terrainBlobParams](raw, "terrain_blob")
	if err != nil {
		return err
	}
	seed_count, err := p.SeedCount.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	size, err := p.Size.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	min_spacing, err := p.MinSpacing.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	if seed_count < 0 {
		return fmt.Errorf("map script terrain_blob: seed_count must be non-negative, got %d", seed_count)
	}
	all := ctx.allSpaces()
	if len(all) == 0 {
		return nil
	}
	seeds := make([]Space, 0, seed_count)
	for attempt := 0; attempt < seed_count*20 && len(seeds) < seed_count; attempt++ {
		candidate := all[ctx.rng.Intn(len(all))]
		ok := true
		for _, s := range seeds {
			if int(game_utils.AxialDistance(candidate.Coordinate(), s.Coordinate())) < min_spacing {
				ok = false
				break
			}
		}
		if ok {
			seeds = append(seeds, candidate)
		}
	}
	region := ctx.region(p.Region)
	for _, seed := range seeds {
		blob := growSpaceBlob(seed, size, ctx.rng)
		for _, space := range blob {
			terrain_id, err := p.resolve(ctx.rng)
			if err != nil {
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

type terrainLineParams struct {
	terrainPickJSON
	Width  ScriptExpr     `json:"width"`
	From   axialParamJSON `json:"from"`
	To     axialParamJSON `json:"to"`
	Region string         `json:"region,omitempty"`
}

func stepTerrainLine(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[terrainLineParams](raw, "terrain_line")
	if err != nil {
		return err
	}
	width, err := p.Width.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	from := game_utils.Coordinate2D{X: p.From.X, Y: p.From.Y}
	to := game_utils.Coordinate2D{X: p.To.X, Y: p.To.Y}
	path := hexLine(from, to, ctx.board)
	region := ctx.region(p.Region)
	seen := make(map[uint]bool)
	for _, cell := range path {
		strip := map[uint]Space{cell.Key(): cell}
		frontier := []Space{cell}
		for depth := 1; depth < width; depth++ {
			next := make([]Space, 0)
			for _, cur := range frontier {
				for _, adj := range cur.SortedAdjacent() {
					if _, in := strip[adj.Key()]; !in {
						strip[adj.Key()] = adj
						next = append(next, adj)
					}
				}
			}
			frontier = next
		}
		// Sorted so which space consumes which rng draw below doesn't depend on map iteration order
		strip_keys := make([]uint, 0, len(strip))
		for key := range strip {
			strip_keys = append(strip_keys, key)
		}
		sort.Slice(strip_keys, func(i, j int) bool { return strip_keys[i] < strip_keys[j] })
		for _, key := range strip_keys {
			if seen[key] {
				continue
			}
			seen[key] = true
			terrain_id, err := p.resolve(ctx.rng)
			if err != nil {
				return err
			}
			strip[key].SetTerrain(terrain_id)
			if region != nil {
				region[key] = true
			}
		}
	}
	return nil
}

type terrainBorderParams struct {
	terrainPickJSON
	Width  ScriptExpr `json:"width"`
	Region string     `json:"region,omitempty"`
}

func stepTerrainBorder(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[terrainBorderParams](raw, "terrain_border")
	if err != nil {
		return err
	}
	width, err := p.Width.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	region := ctx.region(p.Region)
	center := game_utils.Coordinate2D{X: 0, Y: 0}
	threshold := int(ctx.board_size) - width
	for _, space := range ctx.allSpaces() {
		if int(game_utils.AxialDistance(space.Coordinate(), center)) < threshold {
			continue
		}
		terrain_id, err := p.resolve(ctx.rng)
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

func stepEnsureConnectivity(ctx *mapScriptContext, raw json.RawMessage) error {
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
			cell.SetTerrain(defs.DefaultTerrainId)
			reached[cell.Key()] = true
		}
	}
	return nil
}
