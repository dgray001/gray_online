package risq

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"

	"github.com/dgray001/gray_online/game/game_utils"
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
		return p.TerrainId, nil
	}
	if p.TerrainType != "" {
		terrain_type, err := parseTerrainType(p.TerrainType)
		if err != nil {
			return 0, err
		}
		return randomTerrainId(terrain_type, rng), nil
	}
	return 0, fmt.Errorf("must specify terrain_type, terrain_id, or terrain_ids")
}

func growSpaceBlob(start *RisqSpace, size int, rng *rand.Rand) map[uint]*RisqSpace {
	blob := map[uint]*RisqSpace{start.coordinate_key: start}
	frontier := []*RisqSpace{start}
	for len(blob) < size && len(frontier) > 0 {
		idx := rng.Intn(len(frontier))
		cur := frontier[idx]
		frontier = append(frontier[:idx], frontier[idx+1:]...)
		neighbors := make([]*RisqSpace, 0, len(cur.adjacent_spaces))
		for _, adj := range cur.adjacent_spaces {
			if _, in := blob[adj.coordinate_key]; !in {
				neighbors = append(neighbors, adj)
			}
		}
		for _, n := range util.ShuffleFrom(rng, neighbors) {
			if len(blob) >= size {
				break
			}
			if _, in := blob[n.coordinate_key]; in {
				continue
			}
			blob[n.coordinate_key] = n
			frontier = append(frontier, n)
		}
	}
	return blob
}

func growZoneBlob(start *RisqZone, size int, rng *rand.Rand) []*RisqZone {
	blob := map[uint]*RisqZone{start.coordinate_key: start}
	frontier := []*RisqZone{start}
	for len(blob) < size && len(frontier) > 0 {
		idx := rng.Intn(len(frontier))
		cur := frontier[idx]
		frontier = append(frontier[:idx], frontier[idx+1:]...)
		neighbors := append([]*RisqZone{}, cur.adjacent_zones...)
		for _, n := range util.ShuffleFrom(rng, neighbors) {
			if len(blob) >= size || n.isCenter() {
				continue
			}
			if _, in := blob[n.coordinate_key]; in {
				continue
			}
			blob[n.coordinate_key] = n
			frontier = append(frontier, n)
		}
	}
	result := make([]*RisqZone, 0, len(blob))
	for _, z := range blob {
		result = append(result, z)
	}
	return result
}

func cubeRound(x float64, y float64, z float64) (int, int, int) {
	rx, ry, rz := math.Round(x), math.Round(y), math.Round(z)
	dx, dy, dz := math.Abs(rx-x), math.Abs(ry-y), math.Abs(rz-z)
	if dx > dy && dx > dz {
		rx = -ry - rz
	} else if dy > dz {
		ry = -rx - rz
	} else {
		rz = -rx - ry
	}
	return int(rx), int(ry), int(rz)
}

func hexLine(from game_utils.Coordinate2D, to game_utils.Coordinate2D, risq *GameRisq) []*RisqSpace {
	dist := int(game_utils.AxialDistance(from, to))
	if dist == 0 {
		if space := risq.getSpace(&from); space != nil {
			return []*RisqSpace{space}
		}
		return nil
	}
	x1, z1 := float64(from.X), float64(from.Y)
	y1 := -x1 - z1
	x2, z2 := float64(to.X), float64(to.Y)
	y2 := -x2 - z2
	spaces := make([]*RisqSpace, 0, dist+1)
	seen := make(map[uint]bool)
	for i := 0; i <= dist; i++ {
		t := float64(i) / float64(dist)
		rx, _, rz := cubeRound(x1+(x2-x1)*t, y1+(y2-y1)*t, z1+(z2-z1)*t)
		c := game_utils.Coordinate2D{X: rx, Y: rz}
		space := risq.getSpace(&c)
		if space == nil || seen[space.coordinate_key] {
			continue
		}
		seen[space.coordinate_key] = true
		spaces = append(spaces, space)
	}
	return spaces
}

type terrainFillParams struct {
	terrainPickJSON
	Region string `json:"region,omitempty"`
}

func stepTerrainFill(ctx *mapScriptContext, raw json.RawMessage) error {
	var p terrainFillParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script terrain_fill: %v", err))
	}
	region := ctx.region(p.Region)
	for _, space := range ctx.allSpaces() {
		terrain_id, err := p.resolve(ctx.rng)
		if err != nil {
			return err
		}
		space.terrain_id = terrain_id
		if region != nil {
			region[space.coordinate_key] = true
		}
	}
	return nil
}

type terrainBlobParams struct {
	terrainPickJSON
	SeedCount  int    `json:"seed_count"`
	Size       int    `json:"size"`
	MinSpacing int    `json:"min_spacing"`
	Region     string `json:"region,omitempty"`
}

func stepTerrainBlob(ctx *mapScriptContext, raw json.RawMessage) error {
	var p terrainBlobParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script terrain_blob: %v", err))
	}
	all := ctx.allSpaces()
	if len(all) == 0 {
		return nil
	}
	seeds := make([]*RisqSpace, 0, p.SeedCount)
	for attempt := 0; attempt < p.SeedCount*20 && len(seeds) < p.SeedCount; attempt++ {
		candidate := all[ctx.rng.Intn(len(all))]
		ok := true
		for _, s := range seeds {
			if int(game_utils.AxialDistance(candidate.coordinate, s.coordinate)) < p.MinSpacing {
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
		blob := growSpaceBlob(seed, p.Size, ctx.rng)
		for key, space := range blob {
			terrain_id, err := p.resolve(ctx.rng)
			if err != nil {
				return err
			}
			space.terrain_id = terrain_id
			if region != nil {
				region[key] = true
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
	Width  int            `json:"width"`
	From   axialParamJSON `json:"from"`
	To     axialParamJSON `json:"to"`
	Region string         `json:"region,omitempty"`
}

func stepTerrainLine(ctx *mapScriptContext, raw json.RawMessage) error {
	var p terrainLineParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script terrain_line: %v", err))
	}
	from := game_utils.Coordinate2D{X: p.From.X, Y: p.From.Y}
	to := game_utils.Coordinate2D{X: p.To.X, Y: p.To.Y}
	path := hexLine(from, to, ctx.risq)
	region := ctx.region(p.Region)
	seen := make(map[uint]bool)
	for _, cell := range path {
		strip := map[uint]*RisqSpace{cell.coordinate_key: cell}
		frontier := []*RisqSpace{cell}
		for depth := 1; depth < p.Width; depth++ {
			next := make([]*RisqSpace, 0)
			for _, cur := range frontier {
				for _, adj := range cur.adjacent_spaces {
					if _, in := strip[adj.coordinate_key]; !in {
						strip[adj.coordinate_key] = adj
						next = append(next, adj)
					}
				}
			}
			frontier = next
		}
		for key, space := range strip {
			if seen[key] {
				continue
			}
			seen[key] = true
			terrain_id, err := p.resolve(ctx.rng)
			if err != nil {
				return err
			}
			space.terrain_id = terrain_id
			if region != nil {
				region[key] = true
			}
		}
	}
	return nil
}

type terrainBorderParams struct {
	terrainPickJSON
	Width  int    `json:"width"`
	Region string `json:"region,omitempty"`
}

func stepTerrainBorder(ctx *mapScriptContext, raw json.RawMessage) error {
	var p terrainBorderParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script terrain_border: %v", err))
	}
	region := ctx.region(p.Region)
	center := game_utils.Coordinate2D{X: 0, Y: 0}
	threshold := int(ctx.board_size) - p.Width
	for _, space := range ctx.allSpaces() {
		if int(game_utils.AxialDistance(space.coordinate, center)) < threshold {
			continue
		}
		terrain_id, err := p.resolve(ctx.rng)
		if err != nil {
			return err
		}
		space.terrain_id = terrain_id
		if region != nil {
			region[space.coordinate_key] = true
		}
	}
	return nil
}

func shortestPathToSet(from *RisqSpace, target map[uint]bool) []*RisqSpace {
	prev := map[uint]*RisqSpace{from.coordinate_key: nil}
	queue := []*RisqSpace{from}
	var end *RisqSpace
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if target[cur.coordinate_key] {
			end = cur
			break
		}
		for _, adj := range cur.adjacent_spaces {
			if _, seen := prev[adj.coordinate_key]; seen {
				continue
			}
			prev[adj.coordinate_key] = cur
			queue = append(queue, adj)
		}
	}
	if end == nil {
		return nil
	}
	path := make([]*RisqSpace, 0)
	for cell := prev[end.coordinate_key]; cell != nil; cell = prev[cell.coordinate_key] {
		path = append(path, cell)
	}
	return path
}

func stepEnsureConnectivity(ctx *mapScriptContext, raw json.RawMessage) error {
	all := ctx.allSpaces()
	var start *RisqSpace
	for _, s := range all {
		if !s.impassable() {
			start = s
			break
		}
	}
	if start == nil {
		return nil
	}
	reached := map[uint]bool{start.coordinate_key: true}
	queue := []*RisqSpace{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, adj := range cur.adjacent_spaces {
			if adj.impassable() || reached[adj.coordinate_key] {
				continue
			}
			reached[adj.coordinate_key] = true
			queue = append(queue, adj)
		}
	}
	for _, s := range all {
		if s.impassable() || reached[s.coordinate_key] {
			continue
		}
		for _, cell := range shortestPathToSet(s, reached) {
			if reached[cell.coordinate_key] {
				continue
			}
			cell.terrain_id = defaultTerrainId
			reached[cell.coordinate_key] = true
		}
	}
	return nil
}

func weightedResourceCategory(weights map[string]float64, rng *rand.Rand) []uint32 {
	type entry struct {
		ids    []uint32
		weight float64
	}
	entries := []entry{
		{uniformFoodIds, weights["food"]},
		{uniformWoodIds, weights["wood"]},
		{uniformStoneIds, weights["stone"]},
	}
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
	Chance          float64            `json:"chance"`
	CategoryWeights map[string]float64 `json:"category_weights"`
}

func stepResourceScatter(ctx *mapScriptContext, raw json.RawMessage) error {
	var p resourceScatterParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script resource_scatter: %v", err))
	}
	for _, space := range ctx.allSpaces() {
		for _, zone := range space.getZonesAsRandomArray(false, ctx.rng) {
			if zone.resource != nil || zone.building != nil {
				continue
			}
			if ctx.rng.Float64() >= p.Chance {
				continue
			}
			ids := weightedResourceCategory(p.CategoryWeights, ctx.rng)
			if len(ids) == 0 {
				continue
			}
			resource_id := ids[ctx.rng.Intn(len(ids))]
			space.setResource(&zone.coordinate, createRisqResource(ctx.risq.nextResourceInternalId(), resource_id))
		}
	}
	return nil
}

type resourceClusterParams struct {
	ResourceId uint32 `json:"resource_id"`
	SeedCount  int    `json:"seed_count"`
	Size       int    `json:"size"`
}

func stepResourceCluster(ctx *mapScriptContext, raw json.RawMessage) error {
	var p resourceClusterParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script resource_cluster: %v", err))
	}
	candidates := make([]*RisqZone, 0)
	for _, z := range ctx.allZones() {
		if !z.isCenter() && z.resource == nil && z.building == nil {
			candidates = append(candidates, z)
		}
	}
	for i := 0; i < p.SeedCount && len(candidates) > 0; i++ {
		seed := candidates[ctx.rng.Intn(len(candidates))]
		for _, z := range growZoneBlob(seed, p.Size, ctx.rng) {
			if z.isCenter() || z.resource != nil || z.building != nil {
				continue
			}
			z.space.setResource(&z.coordinate, createRisqResource(ctx.risq.nextResourceInternalId(), p.ResourceId))
		}
	}
	return nil
}

type resourceMinSpacingParams struct {
	Distance int `json:"distance"`
}

func stepResourceMinSpacing(ctx *mapScriptContext, raw json.RawMessage) error {
	var p resourceMinSpacingParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script resource_min_spacing: %v", err))
	}
	resource_zones := make([]*RisqZone, 0)
	for _, z := range ctx.allZones() {
		if z.resource != nil {
			resource_zones = append(resource_zones, z)
		}
	}
	kept := make([]*RisqZone, 0, len(resource_zones))
	for _, z := range util.ShuffleFrom(ctx.rng, resource_zones) {
		too_close := false
		for _, k := range kept {
			if int(game_utils.AxialDistance(z.space.coordinate, k.space.coordinate)) < p.Distance {
				too_close = true
				break
			}
		}
		if too_close {
			delete(z.space.resources, z.resource.internal_id)
			z.resource = nil
			continue
		}
		kept = append(kept, z)
	}
	return nil
}

type playerStartResourceJSON struct {
	ResourceId uint32 `json:"resource_id"`
	Count      int    `json:"count"`
}

type playerStartBuildingJSON struct {
	BuildingId uint32 `json:"building_id"`
}

type playerStartsParams struct {
	terrainPickJSON
	Pattern   string                     `json:"pattern"`
	AreaSize  int                        `json:"area_size"`
	Resources []playerStartResourceJSON  `json:"resources"`
	Buildings []playerStartBuildingJSON  `json:"buildings"`
}

var playerStartRingOffsets = map[int][]int{
	1: {0},
	2: {0, 3},
	3: {0, 2, 4},
	4: {0, 1, 3, 4},
	5: {0, 1, 2, 3, 4},
	6: {0, 1, 2, 3, 4, 5},
}

func stepPlayerStarts(ctx *mapScriptContext, raw json.RawMessage) error {
	var p playerStartsParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script player_starts: %v", err))
	}
	if p.Pattern != "ring" {
		return fmt.Errorf("unsupported player_starts pattern %q", p.Pattern)
	}
	offsets, ok := playerStartRingOffsets[ctx.num_players]
	if !ok {
		return fmt.Errorf("unsupported player count %d", ctx.num_players)
	}
	directions := game_utils.AxialDirectionVectors()
	starting_direction := util.RandomIntFrom(ctx.rng, 0, 5)
	ctx.player_starts = make([]playerStartInfo, len(ctx.risq.players))
	for i, offset := range offsets {
		direction := directions[(starting_direction+offset)%6]
		space := ctx.risq.getSpace(direction.Multiply(ctx.starting_distance))
		if space == nil {
			return fmt.Errorf("player start space is nil")
		}
		ctx.player_starts[i] = playerStartInfo{space: space, direction: direction}
		player := ctx.risq.players[i]
		footprint := growSpaceBlob(space, max(1, p.AreaSize), ctx.rng)
		for _, s := range footprint {
			terrain_id, err := p.resolve(ctx.rng)
			if err != nil {
				return err
			}
			s.terrain_id = terrain_id
		}
		footprint_zones := make([]*RisqZone, 0)
		for _, s := range footprint {
			footprint_zones = append(footprint_zones, s.getZonesAsRandomArray(false, ctx.rng)...)
		}
		footprint_zones = util.ShuffleFrom(ctx.rng, footprint_zones)
		zone_idx := 0
		nextFreeZone := func() *RisqZone {
			for zone_idx < len(footprint_zones) {
				z := footprint_zones[zone_idx]
				zone_idx++
				if z.resource == nil && z.building == nil {
					return z
				}
			}
			return nil
		}
		for bi, b := range p.Buildings {
			target := space.getCenterZone()
			if bi > 0 {
				target = nextFreeZone()
				if target == nil {
					break
				}
			}
			building := createRisqBuilding(ctx.risq.nextBuildingInternalId(), b.BuildingId, player.player.Player_id)
			target.space.setBuilding(&target.coordinate, building)
			player.buildings[building.internal_id] = building
			ctx.risq.buildings[building.internal_id] = building
		}
		for _, r := range p.Resources {
			for range r.Count {
				target := nextFreeZone()
				if target == nil {
					break
				}
				target.space.setResource(&target.coordinate, createRisqResource(ctx.risq.nextResourceInternalId(), r.ResourceId))
			}
		}
	}
	return nil
}
