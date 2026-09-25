package risq

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"

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

func growSpaceBlob(start *RisqSpace, size int, rng *rand.Rand) []*RisqSpace {
	return growBlob(start, size, rng,
		func(s *RisqSpace) uint { return s.coordinate_key },
		func(s *RisqSpace) []*RisqSpace { return s.sortedAdjacentSpaces() },
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

// Decodes a step's JSON params into T, returning an error rather than panicking -- unlike other
// config, which is validated at server startup, a malformed map script is only caught at launch
// time, so a decode failure here must not be able to crash a live server.
func decodeStepParams[T any](raw json.RawMessage, step_name string) (T, error) {
	var p T
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, fmt.Errorf("%s: %v", step_name, err)
	}
	return p, nil
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
	path := hexLine(from, to, ctx.risq)
	region := ctx.region(p.Region)
	seen := make(map[uint]bool)
	for _, cell := range path {
		strip := map[uint]*RisqSpace{cell.coordinate_key: cell}
		frontier := []*RisqSpace{cell}
		for depth := 1; depth < width; depth++ {
			next := make([]*RisqSpace, 0)
			for _, cur := range frontier {
				for _, adj := range cur.sortedAdjacentSpaces() {
					if _, in := strip[adj.coordinate_key]; !in {
						strip[adj.coordinate_key] = adj
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
			strip[key].terrain_id = terrain_id
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
		for _, adj := range cur.sortedAdjacentSpaces() {
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
		for _, adj := range cur.sortedAdjacentSpaces() {
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
	p, err := decodeStepParams[resourceScatterParams](raw, "resource_scatter")
	if err != nil {
		return err
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
	p, err := decodeStepParams[resourceClusterParams](raw, "resource_cluster")
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
	p, err := decodeStepParams[resourceMinSpacingParams](raw, "resource_min_spacing")
	if err != nil {
		return err
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

type playerStartUnitJSON struct {
	UnitId uint32         `json:"unit_id"`
	Count  ScriptExpr     `json:"count"`
	Target zoneTargetJSON `json:"target"`
}

type zoneTerrainOverrideJSON struct {
	TerrainId uint32         `json:"terrain_id"`
	Target    zoneTargetJSON `json:"target"`
}

type playerStartsParams struct {
	terrainPickJSON
	Pattern              string                    `json:"pattern"`
	AreaSize             ScriptExpr                `json:"area_size"`
	StartingDistance     ScriptExpr                `json:"starting_distance"`
	Units                []playerStartUnitJSON     `json:"units,omitempty"`
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

// Places up to 6 players evenly around a hex's 6 principal directions at one distance from center.
// Player counts above 6 place the remainder (also <= 6) on a second, closer ring along the same
// directions, since there are only 6 principal directions to place a single ring's players along.
func resolveRingPlayerStarts(ctx *mapScriptContext, starting_distance int) ([]playerStartInfo, error) {
	directions := game_utils.AxialDirectionVectors()
	starting_direction := util.RandomIntFrom(ctx.rng, 0, 5)
	starts := make([]playerStartInfo, len(ctx.risq.players))
	place := func(count int, distance int, start_index int) error {
		offsets, ok := playerStartRingOffsets[count]
		if !ok {
			return fmt.Errorf("unsupported player count %d", count)
		}
		for i, offset := range offsets {
			direction := directions[(starting_direction+offset)%6]
			space := ctx.risq.getSpace(direction.Multiply(distance))
			if space == nil {
				return fmt.Errorf("player start space is nil")
			}
			starts[start_index+i] = playerStartInfo{space: space, direction: direction}
		}
		return nil
	}
	outer := min(ctx.num_players, 6)
	if err := place(outer, starting_distance, 0); err != nil {
		return nil, err
	}
	if inner := ctx.num_players - outer; inner > 0 {
		if err := place(inner, max(1, starting_distance-2), outer); err != nil {
			return nil, err
		}
	}
	return starts, nil
}

func rectangleSpaceBounds(spaces []*RisqSpace) (row_min int, row_max int, col_min int, col_max int) {
	first := true
	for _, s := range spaces {
		row := s.coordinate.Y
		col := s.coordinate.X + floorDiv2(row)
		if first {
			row_min, row_max, col_min, col_max = row, row, col, col
			first = false
			continue
		}
		row_min, row_max = min(row_min, row), max(row_max, row)
		col_min, col_max = min(col_min, col), max(col_max, col)
	}
	return
}

func rowPlayerStartColumn(slot int, count int, col_min int, col_max int) int {
	if count <= 1 {
		return (col_min + col_max) / 2
	}
	return col_min + (slot*(col_max-col_min))/(count-1)
}

// A row's own column range among the spaces the current shape actually kept -- e.g. a triangle's
// rows narrow toward its tip, so this must be recomputed per row rather than reused from the whole
// shape's bounding box, or evenly-spread columns can land outside a narrower row's real footprint.
func rowSpaceBounds(spaces []*RisqSpace, row int) (col_min int, col_max int, ok bool) {
	for _, s := range spaces {
		if s.coordinate.Y != row {
			continue
		}
		col := s.coordinate.X + floorDiv2(row)
		if !ok {
			col_min, col_max, ok = col, col, true
			continue
		}
		col_min, col_max = min(col_min, col), max(col_max, col)
	}
	return
}

// Places players in two facing rows, spread evenly across columns; the natural pattern for a rectangle shape
func resolveRowsPlayerStarts(ctx *mapScriptContext, starting_distance int) ([]playerStartInfo, error) {
	spaces := ctx.allSpaces()
	row_min, row_max, _, _ := rectangleSpaceBounds(spaces)
	half := max((row_max-row_min)/2, 1)
	dist := util.Clamp(starting_distance, 1, half)
	row_near := util.Clamp(-dist, row_min, row_max)
	row_far := util.Clamp(dist, row_min, row_max)
	n := len(ctx.risq.players)
	group_near := (n + 1) / 2
	group_far := n - group_near
	starts := make([]playerStartInfo, n)
	place := func(count int, row int, direction game_utils.Coordinate2D, start_index int) error {
		col_min, col_max, ok := rowSpaceBounds(spaces, row)
		if !ok {
			return fmt.Errorf("no spaces at row %d for player starts", row)
		}
		for i := 0; i < count; i++ {
			col := rowPlayerStartColumn(i, count, col_min, col_max)
			q := col - floorDiv2(row)
			space := ctx.risq.getSpace(&game_utils.Coordinate2D{X: q, Y: row})
			if space == nil {
				return fmt.Errorf("player start space is nil")
			}
			starts[start_index+i] = playerStartInfo{space: space, direction: direction}
		}
		return nil
	}
	if err := place(group_near, row_near, game_utils.Coordinate2D{X: 0, Y: 1}, 0); err != nil {
		return nil, err
	}
	if err := place(group_far, row_far, game_utils.Coordinate2D{X: 0, Y: -1}, group_near); err != nil {
		return nil, err
	}
	return starts, nil
}

func stepPlayerStarts(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[playerStartsParams](raw, "player_starts")
	if err != nil {
		return err
	}
	starting_distance, err := p.StartingDistance.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	if starting_distance > int(ctx.board_size) {
		starting_distance = int(ctx.board_size)
	}
	var starts []playerStartInfo
	switch p.Pattern {
	case "ring":
		starts, err = resolveRingPlayerStarts(ctx, starting_distance)
	case "rows":
		starts, err = resolveRowsPlayerStarts(ctx, starting_distance)
	default:
		return fmt.Errorf("unsupported player_starts pattern %q", p.Pattern)
	}
	if err != nil {
		return err
	}
	area_size, err := p.AreaSize.resolveInt(ctx.vars)
	if err != nil {
		return err
	}
	ctx.player_starts = starts
	// a player's own home space is reserved; other players' footprints (their non-home spaces) may still overlap it
	home_space_keys := make(map[uint]bool, len(starts))
	for _, start := range starts {
		home_space_keys[start.space.coordinate_key] = true
	}
	for i, start := range starts {
		space := start.space
		player := ctx.risq.players[i]
		footprint := make([]*RisqSpace, 0)
		for _, s := range hexRadiusSpaces(space, area_size) {
			if s != space && home_space_keys[s.coordinate_key] {
				continue
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
			if target.building != building {
				continue
			}
			player.buildings[building.internal_id] = building
			ctx.risq.buildings[building.internal_id] = building
			if b.TerrainOverride != 0 {
				target.terrain_override = b.TerrainOverride
			}
		}
		for _, u := range p.Units {
			count, err := u.Count.resolveInt(ctx.vars)
			if err != nil {
				return err
			}
			for range count {
				target, err := resolveZoneTarget(footprint, space, u.Target, ctx.rng)
				if err != nil {
					return err
				}
				unit := createRisqUnit(ctx.risq.nextUnitInternalId(), u.UnitId, player)
				target.space.setUnit(&target.coordinate, unit)
				player.units[unit.internal_id] = unit
				ctx.risq.units[unit.internal_id] = unit
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

type shapeParams struct {
	Kind      string     `json:"kind"`
	Size      ScriptExpr `json:"size,omitempty"`
	InnerSize ScriptExpr `json:"inner_size,omitempty"`
	Thickness ScriptExpr `json:"thickness,omitempty"`
	Rows      ScriptExpr `json:"rows,omitempty"`
	Cols      ScriptExpr `json:"cols,omitempty"`
}

func stepShape(ctx *mapScriptContext, raw json.RawMessage) error {
	if ctx.shape != "" {
		return fmt.Errorf("shape: already declared as %q", ctx.shape)
	}
	var p shapeParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("shape: %v", err)
	}
	ctx.vars["recommended"] = float64(recommendedShapeSize(p.Kind, ctx.num_players))
	switch p.Kind {
	case "hexagon":
		size, err := p.Size.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		if size < 0 {
			return fmt.Errorf("shape: hexagon size must be >= 0")
		}
		ctx.risq.allocateBoard(uint16(size))
		ctx.shape = "hexagon"
	case "rectangle":
		rows, err := p.Rows.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		cols, err := p.Cols.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		if rows < 1 || cols < 1 {
			return fmt.Errorf("shape: rectangle rows and cols must be >= 1")
		}
		ctx.risq.allocateBoard(rectangleRequiredBoardSize(rows, cols))
		row_min := -((rows - 1) / 2)
		row_max := row_min + rows - 1
		col_min := -((cols - 1) / 2)
		col_max := col_min + cols - 1
		kept := 0
		for _, space := range ctx.allSpaces() {
			row := space.coordinate.Y
			col := space.coordinate.X + floorDiv2(row)
			if row < row_min || row > row_max || col < col_min || col > col_max {
				ctx.risq.removeSpace(space)
				continue
			}
			kept++
		}
		if kept < rows*cols {
			return fmt.Errorf("shape: board_size too small to fit a %dx%d rectangle (increase board_size)", cols, rows)
		}
		ctx.shape = "rectangle"
	case "ring":
		size, err := p.Size.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		var inner_size int
		if p.Thickness.provided() {
			thickness, err := p.Thickness.resolveInt(ctx.vars)
			if err != nil {
				return err
			}
			inner_size = size - thickness
		} else {
			inner_size, err = p.InnerSize.resolveInt(ctx.vars)
			if err != nil {
				return err
			}
		}
		if size < 0 || inner_size < 0 || inner_size >= size {
			return fmt.Errorf("shape: ring requires 0 <= inner_size < size")
		}
		ctx.risq.allocateBoard(uint16(size))
		for _, space := range ctx.allSpaces() {
			d := int(game_utils.AxialDistance(game_utils.Coordinate2D{}, space.coordinate))
			if d > size || d <= inner_size {
				ctx.risq.removeSpace(space)
			}
		}
		ctx.shape = "ring"
	case "triangle":
		size, err := p.Size.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		if size < 0 {
			return fmt.Errorf("shape: triangle size must be >= 0")
		}
		offset := triangleCenterOffset(size)
		ctx.risq.allocateBoard(triangleRequiredBoardSize(size))
		for _, space := range ctx.allSpaces() {
			q, r := space.coordinate.X, space.coordinate.Y
			if q < -offset || r < -offset || q+r > size-2*offset {
				ctx.risq.removeSpace(space)
			}
		}
		ctx.shape = "triangle"
	default:
		return fmt.Errorf("shape: unknown kind %q", p.Kind)
	}
	ctx.board_size = ctx.risq.board_size
	ctx.vars["board_size"] = float64(ctx.risq.board_size)
	return nil
}

func ringArea(outer int) int {
	inner := int(math.Round(float64(outer) / 2))
	return hexAreaForSize(outer) - hexAreaForSize(inner)
}

func recommendedShapeSize(kind string, num_players int) int {
	n := int(recommendedBoardSize(num_players))
	hex_area := hexAreaForSize(n)
	switch kind {
	case "rectangle":
		return int(math.Round(math.Sqrt(float64(hex_area))))
	case "triangle":
		return int(math.Round((-3 + math.Sqrt(1+8*float64(hex_area))) / 2))
	case "ring":
		outer := n
		for ringArea(outer) < hex_area {
			outer++
		}
		if outer > n && util.AbsInt(ringArea(outer-1)-hex_area) <= util.AbsInt(ringArea(outer)-hex_area) {
			outer--
		}
		return outer
	default:
		return n
	}
}

func triangleCenterOffset(size int) int {
	return int(math.Round(float64(size) / 3))
}

func triangleRequiredBoardSize(size int) uint16 {
	offset := triangleCenterOffset(size)
	required := max(2*offset, size-offset)
	if required < 0 {
		required = 0
	}
	return uint16(required)
}

// Computes the exact hex board radius needed to fit an R x C rectangle, checking the true axial
// distance of each row's two extreme columns rather than approximating with a closed-form formula
// (which broke for asymmetric row/col spans, e.g. even cols) -- must match stepShape's own carving.
func rectangleRequiredBoardSize(rows int, cols int) uint16 {
	row_min := -((rows - 1) / 2)
	row_max := row_min + rows - 1
	col_min := -((cols - 1) / 2)
	col_max := col_min + cols - 1
	required := 0
	for row := row_min; row <= row_max; row++ {
		shift := floorDiv2(row)
		for _, col := range [2]int{col_min, col_max} {
			x := col - shift
			d := int(game_utils.AxialDistance(game_utils.Coordinate2D{}, game_utils.Coordinate2D{X: x, Y: row}))
			required = max(required, d)
		}
	}
	return uint16(required)
}

type defineParams struct {
	Name  string     `json:"name"`
	Value ScriptExpr `json:"value"`
}

func stepDefine(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[defineParams](raw, "define")
	if err != nil {
		return err
	}
	v, err := p.Value.resolve(ctx.vars)
	if err != nil {
		return err
	}
	ctx.vars[p.Name] = v
	return nil
}

type mirrorParams struct {
	Step mapScriptStepJSON `json:"step"`
}

// Replays a wrapped step's board changes rotated to every other player start direction
func stepMirror(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[mirrorParams](raw, "mirror")
	if err != nil {
		return err
	}
	fn, ok := mapStepRegistry[p.Step.Step]
	if !ok {
		return fmt.Errorf("mirror: unknown step %q", p.Step.Step)
	}
	if len(ctx.player_starts) < 2 {
		return fn(ctx, p.Step.Params)
	}
	terrain_before := make(map[uint]uint32)
	for _, space := range ctx.allSpaces() {
		terrain_before[space.coordinate_key] = space.terrain_id
	}
	resource_zones_before := make(map[uint]bool)
	for _, zone := range ctx.allZones() {
		if zone.resource != nil {
			resource_zones_before[zone.coordinate_key] = true
		}
	}
	if err := fn(ctx, p.Step.Params); err != nil {
		return err
	}
	type terrainChange struct {
		coordinate game_utils.Coordinate2D
		terrain_id uint32
	}
	terrain_changes := make([]terrainChange, 0)
	for _, space := range ctx.allSpaces() {
		if before, ok := terrain_before[space.coordinate_key]; ok && before != space.terrain_id {
			terrain_changes = append(terrain_changes, terrainChange{coordinate: space.coordinate, terrain_id: space.terrain_id})
		}
	}
	type resourceChange struct {
		space_coordinate game_utils.Coordinate2D
		zone_coordinate  game_utils.Coordinate2D
		resource_id      uint32
	}
	resource_changes := make([]resourceChange, 0)
	for _, zone := range ctx.allZones() {
		if zone.resource != nil && !resource_zones_before[zone.coordinate_key] {
			resource_changes = append(resource_changes, resourceChange{
				space_coordinate: zone.space.coordinate,
				zone_coordinate:  zone.coordinate,
				resource_id:      zone.resource.resource_id,
			})
		}
	}
	base_dir := zoneDirection(ctx.player_starts[0].direction.X, ctx.player_starts[0].direction.Y)
	for i := 1; i < len(ctx.player_starts); i++ {
		dir := zoneDirection(ctx.player_starts[i].direction.X, ctx.player_starts[i].direction.Y)
		if base_dir < 0 || dir < 0 {
			continue
		}
		k := dir - base_dir
		for _, change := range terrain_changes {
			rotated := rotateAxial(change.coordinate, k)
			if space := ctx.risq.getSpace(&rotated); space != nil {
				space.terrain_id = change.terrain_id
			}
		}
		for _, change := range resource_changes {
			rotated_space_c := rotateAxial(change.space_coordinate, k)
			space := ctx.risq.getSpace(&rotated_space_c)
			if space == nil {
				continue
			}
			local := change.zone_coordinate
			if local.X != 0 || local.Y != 0 {
				local = rotateAxial(local, k)
			}
			zone := space.getZone(&local)
			if zone == nil || zone.resource != nil || zone.building != nil {
				continue
			}
			space.setResource(&local, createRisqResource(ctx.risq.nextResourceInternalId(), change.resource_id))
		}
	}
	return nil
}

type regionsSevenParams struct {
	Names []string `json:"names,omitempty"`
}

func stepRegionsSeven(ctx *mapScriptContext, raw json.RawMessage) error {
	if ctx.shape != "hexagon" {
		return fmt.Errorf("regions_seven: only valid on a hexagon shape, got %q", ctx.shape)
	}
	var p regionsSevenParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("regions_seven: %v", err)
	}
	names := append([]string{}, p.Names...)
	if len(names) < 7 {
		names = append(names, randomRegionNames(ctx.rng, 7-len(names))...)
	}
	if len(names) < 7 {
		return fmt.Errorf("regions_seven: not enough region names available")
	}
	n := int(ctx.board_size)
	total := len(ctx.allSpaces())
	center_radius := 0
	for 7*hexAreaForSize(center_radius) < total {
		center_radius++
	}
	center_keys := make(map[uint]bool)
	for _, space := range ctx.allSpaces() {
		if int(game_utils.AxialDistance(game_utils.Coordinate2D{}, space.coordinate)) <= center_radius {
			center_keys[space.coordinate_key] = true
		}
	}
	if err := ctx.risq.addRegion(names[0], 0, center_keys); err != nil {
		return err
	}
	sector_keys := [6]map[uint]bool{}
	for i := range sector_keys {
		sector_keys[i] = make(map[uint]bool)
	}
	for d := center_radius + 1; d <= n; d++ {
		ring := hexRingSectors(game_utils.Coordinate2D{}, d)
		for i, coords := range ring {
			for _, c := range coords {
				if space := ctx.risq.getSpace(&c); space != nil {
					sector_keys[i][space.coordinate_key] = true
				}
			}
		}
	}
	for i := 0; i < 6; i++ {
		if err := ctx.risq.addRegion(names[i+1], 0, sector_keys[i]); err != nil {
			return err
		}
	}
	return nil
}
