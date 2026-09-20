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

// Returns every space within hex distance radius of start (radius 0 -> just start, radius 1 -> start + 6 neighbors, etc.)
func hexRadiusSpaces(start *RisqSpace, radius int) []*RisqSpace {
	seen := map[uint]bool{start.coordinate_key: true}
	result := []*RisqSpace{start}
	frontier := []*RisqSpace{start}
	for len(frontier) > 0 {
		cur := frontier[0]
		frontier = frontier[1:]
		for _, n := range cur.adjacent_spaces {
			if seen[n.coordinate_key] || int(game_utils.AxialDistance(start.coordinate, n.coordinate)) > radius {
				continue
			}
			seen[n.coordinate_key] = true
			result = append(result, n)
			frontier = append(frontier, n)
		}
	}
	return result
}

func growSpaceBlob(start *RisqSpace, size int, rng *rand.Rand) []*RisqSpace {
	return growBlob(start, size, rng,
		func(s *RisqSpace) uint { return s.coordinate_key },
		func(s *RisqSpace) []*RisqSpace {
			neighbors := make([]*RisqSpace, 0, len(s.adjacent_spaces))
			for _, adj := range s.adjacent_spaces {
				neighbors = append(neighbors, adj)
			}
			return neighbors
		},
		func(*RisqSpace) bool { return false },
	)
}

func growZoneBlob(start *RisqZone, size int, rng *rand.Rand) []*RisqZone {
	return growBlob(start, size, rng,
		func(z *RisqZone) uint { return z.coordinate_key },
		func(z *RisqZone) []*RisqZone { return z.adjacent_zones },
		func(z *RisqZone) bool { return z.isCenter() },
	)
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
	SeedCount  ScriptExpr `json:"seed_count"`
	Size       ScriptExpr `json:"size"`
	MinSpacing ScriptExpr `json:"min_spacing"`
	Region     string     `json:"region,omitempty"`
}

func stepTerrainBlob(ctx *mapScriptContext, raw json.RawMessage) error {
	var p terrainBlobParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script terrain_blob: %v", err))
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
	all := ctx.allSpaces()
	if len(all) == 0 {
		return nil
	}
	seeds := make([]*RisqSpace, 0, seed_count)
	for attempt := 0; attempt < seed_count*20 && len(seeds) < seed_count; attempt++ {
		candidate := all[ctx.rng.Intn(len(all))]
		ok := true
		for _, s := range seeds {
			if int(game_utils.AxialDistance(candidate.coordinate, s.coordinate)) < min_spacing {
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
			space.terrain_id = terrain_id
			if region != nil {
				region[space.coordinate_key] = true
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
	var p terrainLineParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script terrain_line: %v", err))
	}
	width, err := p.Width.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	from := game_utils.Coordinate2D{X: p.From.X, Y: p.From.Y}
	to := game_utils.Coordinate2D{X: p.To.X, Y: p.To.Y}
	path := hexLine(from, to, ctx.risq)
	region := ctx.region(p.Region)
	seen := make(map[uint]bool)
	for _, cell := range path {
		strip := map[uint]*RisqSpace{cell.coordinate_key: cell}
		frontier := []*RisqSpace{cell}
		for depth := 1; depth < width; depth++ {
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
	Width  ScriptExpr `json:"width"`
	Region string     `json:"region,omitempty"`
}

func stepTerrainBorder(ctx *mapScriptContext, raw json.RawMessage) error {
	var p terrainBorderParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script terrain_border: %v", err))
	}
	width, err := p.Width.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	region := ctx.region(p.Region)
	center := game_utils.Coordinate2D{X: 0, Y: 0}
	threshold := int(ctx.board_size) - width
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
	Chance          ScriptExpr         `json:"chance"`
	CategoryWeights map[string]float64 `json:"category_weights"`
}

func stepResourceScatter(ctx *mapScriptContext, raw json.RawMessage) error {
	var p resourceScatterParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script resource_scatter: %v", err))
	}
	chance, err := p.Chance.resolve(ctx.vars)
	if err != nil {
		return err
	}
	for _, space := range ctx.allSpaces() {
		for _, zone := range space.getZonesAsRandomArray(false, ctx.rng) {
			if zone.resource != nil || zone.building != nil {
				continue
			}
			if ctx.rng.Float64() >= chance {
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
	ResourceId uint32     `json:"resource_id"`
	SeedCount  ScriptExpr `json:"seed_count"`
	Size       ScriptExpr `json:"size"`
}

func stepResourceCluster(ctx *mapScriptContext, raw json.RawMessage) error {
	var p resourceClusterParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script resource_cluster: %v", err))
	}
	seed_count, err := p.SeedCount.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	size, err := p.Size.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	candidates := make([]*RisqZone, 0)
	for _, z := range ctx.allZones() {
		if !z.isCenter() && z.resource == nil && z.building == nil {
			candidates = append(candidates, z)
		}
	}
	for i := 0; i < seed_count && len(candidates) > 0; i++ {
		seed := candidates[ctx.rng.Intn(len(candidates))]
		for _, z := range growZoneBlob(seed, size, ctx.rng) {
			if z.isCenter() || z.resource != nil || z.building != nil {
				continue
			}
			z.space.setResource(&z.coordinate, createRisqResource(ctx.risq.nextResourceInternalId(), p.ResourceId))
		}
	}
	return nil
}

type resourceMinSpacingParams struct {
	Distance ScriptExpr `json:"distance"`
}

func stepResourceMinSpacing(ctx *mapScriptContext, raw json.RawMessage) error {
	var p resourceMinSpacingParams
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script resource_min_spacing: %v", err))
	}
	distance, err := p.Distance.resolveInt(ctx.vars)
	if err != nil {
		return err
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
			if int(game_utils.AxialDistance(z.space.coordinate, k.space.coordinate)) < distance {
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
	ResourceId uint32     `json:"resource_id"`
	Count      ScriptExpr `json:"count"`
}

// Targets one zone relative to a player's start space, for deterministic map-script placement
type zoneTargetJSON struct {
	SpaceDistance int    `json:"space_distance,omitempty"` // hex distance from the start space; 0 = the start space itself
	Zone          string `json:"zone"`                     // "center" or "edge"
	EdgeIndex     int    `json:"edge_index,omitempty"`     // 1-6; 0 = a random free edge zone
}

type playerStartBuildingJSON struct {
	BuildingId      uint32         `json:"building_id"`
	TerrainOverride uint32         `json:"terrain_override,omitempty"`
	Target          zoneTargetJSON `json:"target"`
}

type zoneTerrainOverrideJSON struct {
	TerrainId uint32         `json:"terrain_id"`
	Target    zoneTargetJSON `json:"target"`
}

type playerStartsParams struct {
	terrainPickJSON
	Pattern              string                    `json:"pattern"`
	AreaSize             ScriptExpr                `json:"area_size"`
	Resources            []playerStartResourceJSON `json:"resources"`
	Buildings            []playerStartBuildingJSON `json:"buildings"`
	ZoneTerrainOverrides []zoneTerrainOverrideJSON `json:"zone_terrain_overrides,omitempty"`
}

// Resolves a zoneTargetJSON to a concrete zone within the given footprint, relative to start
func resolveZoneTarget(footprint []*RisqSpace, start *RisqSpace, target zoneTargetJSON, rng *rand.Rand) (*RisqZone, error) {
	target_space := start
	if target.SpaceDistance > 0 {
		candidates := make([]*RisqSpace, 0)
		for _, s := range footprint {
			if int(game_utils.AxialDistance(start.coordinate, s.coordinate)) == target.SpaceDistance {
				candidates = append(candidates, s)
			}
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("no space at distance %d from player start", target.SpaceDistance)
		}
		target_space = candidates[rng.Intn(len(candidates))]
	}
	if target.Zone == "center" {
		return target_space.getCenterZone(), nil
	}
	if target.Zone != "edge" {
		return nil, fmt.Errorf("zone target must be \"center\" or \"edge\", got %q", target.Zone)
	}
	directions := game_utils.AxialDirectionVectors()
	if target.EdgeIndex > 0 {
		if target.EdgeIndex > len(directions) {
			return nil, fmt.Errorf("edge_index %d out of range", target.EdgeIndex)
		}
		return target_space.getZone(&directions[target.EdgeIndex-1]), nil
	}
	free := make([]*RisqZone, 0, len(directions))
	for _, d := range directions {
		if z := target_space.getZone(&d); z != nil && z.resource == nil && z.building == nil {
			free = append(free, z)
		}
	}
	if len(free) == 0 {
		return nil, fmt.Errorf("no free edge zone available")
	}
	return free[rng.Intn(len(free))], nil
}

var playerStartRingOffsets = map[int][]int{
	1: {0},
	2: {0, 3},
	3: {0, 2, 4},
	4: {0, 1, 3, 4},
	5: {0, 1, 2, 3, 4},
	6: {0, 1, 2, 3, 4, 5},
}

func clearSpaceOccupants(r *GameRisq, s *RisqSpace) {
	for _, row := range s.zones {
		for _, zone := range row {
			zone.terrain_override = 0
			if zone.resource != nil {
				s.removeResource(zone.resource)
			}
			if zone.building != nil {
				building := zone.building
				s.removeBuilding(building)
				delete(r.buildings, building.internal_id)
				delete(r.players[building.player_id].buildings, building.internal_id)
			}
		}
	}
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
	area_size, err := p.AreaSize.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	directions := game_utils.AxialDirectionVectors()
	starting_direction := util.RandomIntFrom(ctx.rng, 0, 5)
	ctx.player_starts = make([]playerStartInfo, len(ctx.risq.players))
	// a player's own home space is reserved; other players' footprints (their non-home ring spaces) may still overlap it
	home_space_keys := make(map[uint]bool, len(offsets))
	for _, offset := range offsets {
		direction := directions[(starting_direction+offset)%6]
		home_space := ctx.risq.getSpace(direction.Multiply(ctx.starting_distance))
		if home_space == nil {
			return fmt.Errorf("player start space is nil")
		}
		home_space_keys[home_space.coordinate_key] = true
	}
	for i, offset := range offsets {
		direction := directions[(starting_direction+offset)%6]
		space := ctx.risq.getSpace(direction.Multiply(ctx.starting_distance))
		if space == nil {
			return fmt.Errorf("player start space is nil")
		}
		ctx.player_starts[i] = playerStartInfo{space: space, direction: direction}
		player := ctx.risq.players[i]
		footprint := make([]*RisqSpace, 0)
		for _, s := range hexRadiusSpaces(space, area_size) {
			if s != space && home_space_keys[s.coordinate_key] {
				continue // never claim another player's home space
			}
			footprint = append(footprint, s)
		}
		for _, s := range footprint {
			clearSpaceOccupants(ctx.risq, s)
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
		for _, b := range p.Buildings {
			target, err := resolveZoneTarget(footprint, space, b.Target, ctx.rng)
			if err != nil {
				return err
			}
			building := createRisqBuilding(ctx.risq.nextBuildingInternalId(), b.BuildingId, player.player.Player_id)
			target.space.setBuilding(&target.coordinate, building)
			player.buildings[building.internal_id] = building
			ctx.risq.buildings[building.internal_id] = building
			if b.TerrainOverride != 0 {
				target.terrain_override = b.TerrainOverride
			}
		}
		for _, o := range p.ZoneTerrainOverrides {
			target, err := resolveZoneTarget(footprint, space, o.Target, ctx.rng)
			if err != nil {
				return err
			}
			target.terrain_override = o.TerrainId
		}
		for _, r := range p.Resources {
			count, err := r.Count.resolveInt(ctx.vars)
			if err != nil {
				return err
			}
			for range count {
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
