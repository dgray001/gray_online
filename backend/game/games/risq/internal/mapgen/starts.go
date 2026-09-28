package mapgen

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

type playerStartResourceJSON struct {
	ResourceId uint32     `json:"resource_id"`
	Count      ScriptExpr `json:"count"`
	// optional [min, max] space distance from the start space: [0, 0] is the start space, [1, 1] the ring around it
	Distance []int `json:"distance,omitempty"`
}

// One start resource relative to a player's start, laid out once and rotated to each player so starts are symmetric
type startResourceSlot struct {
	resource_id  uint32
	min_distance int
	max_distance int
	space_offset game_utils.Coordinate2D
	zone         game_utils.Coordinate2D
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
func resolveZoneTarget(footprint []Space, start Space, target zoneTargetJSON, rng *rand.Rand) (Zone, error) {
	target_space := start
	if target.SpaceDistance > 0 {
		candidates := make([]Space, 0)
		for _, s := range footprint {
			if int(game_utils.AxialDistance(start.Coordinate(), s.Coordinate())) == target.SpaceDistance {
				candidates = append(candidates, s)
			}
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("no space at distance %d from player start", target.SpaceDistance)
		}
		target_space = candidates[rng.Intn(len(candidates))]
	}
	if target.Zone == "center" {
		return target_space.CenterZone(), nil
	}
	if target.Zone != "edge" {
		return nil, fmt.Errorf("zone target must be \"center\" or \"edge\", got %q", target.Zone)
	}
	directions := game_utils.AxialDirectionVectors()
	if target.EdgeIndex > 0 {
		if target.EdgeIndex > len(directions) {
			return nil, fmt.Errorf("edge_index %d out of range", target.EdgeIndex)
		}
		return target_space.Zone(directions[target.EdgeIndex-1]), nil
	}
	free := make([]Zone, 0, len(directions))
	for _, d := range directions {
		if z := target_space.Zone(d); z != nil && !z.Occupied() {
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

// Places up to 6 players evenly around a hex's 6 principal directions at one distance from center.
// Player counts above 6 place the remainder (also <= 6) on a second, closer ring along the same
// directions, since there are only 6 principal directions to place a single ring's players along.
func resolveRingPlayerStarts(ctx *mapScriptContext, starting_distance int) ([]playerStartInfo, error) {
	directions := game_utils.AxialDirectionVectors()
	starting_direction := util.RandomIntFrom(ctx.rng, 0, 5)
	starts := make([]playerStartInfo, ctx.num_players)
	place := func(count int, distance int, start_index int) error {
		offsets, ok := playerStartRingOffsets[count]
		if !ok {
			return fmt.Errorf("unsupported player count %d", count)
		}
		for i, offset := range offsets {
			direction := directions[(starting_direction+offset)%6]
			space := ctx.board.Space(*direction.Multiply(distance))
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

func rectangleSpaceBounds(spaces []Space) (row_min int, row_max int, col_min int, col_max int) {
	first := true
	for _, s := range spaces {
		row := s.Coordinate().Y
		col := s.Coordinate().X + floorDiv2(row)
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
func rowSpaceBounds(spaces []Space, row int) (col_min int, col_max int, ok bool) {
	for _, s := range spaces {
		if s.Coordinate().Y != row {
			continue
		}
		col := s.Coordinate().X + floorDiv2(row)
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
	n := ctx.num_players
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
			space := ctx.board.Space(game_utils.Coordinate2D{X: q, Y: row})
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

func planStartResources(ctx *mapScriptContext, resources []playerStartResourceJSON, area_size int) ([]startResourceSlot, error) {
	used := make(map[[4]int]bool)
	slots := make([]startResourceSlot, 0)
	for _, r := range resources {
		count, err := r.Count.resolveInt(ctx.vars)
		if err != nil {
			return nil, err
		}
		lo, hi := 0, area_size
		if len(r.Distance) == 2 {
			lo, hi = r.Distance[0], min(r.Distance[1], area_size)
		} else if len(r.Distance) != 0 {
			return nil, fmt.Errorf("resource %d distance must be [min, max]", r.ResourceId)
		}
		candidates := make([][4]int, 0)
		for dq := -hi; dq <= hi; dq++ {
			for dr := -hi; dr <= hi; dr++ {
				if d := int(game_utils.AxialDistance(game_utils.Coordinate2D{}, game_utils.Coordinate2D{X: dq, Y: dr})); d < lo || d > hi {
					continue
				}
				for _, z := range game_utils.AxialDirectionVectors() {
					candidates = append(candidates, [4]int{dq, dr, z.X, z.Y})
				}
			}
		}
		candidates = util.ShuffleFrom(ctx.rng, candidates)
		placed := 0
		for _, c := range candidates {
			if placed >= count {
				break
			}
			if used[c] {
				continue
			}
			used[c] = true
			placed++
			slots = append(slots, startResourceSlot{resource_id: r.ResourceId, min_distance: lo, max_distance: hi,
				space_offset: game_utils.Coordinate2D{X: c[0], Y: c[1]}, zone: game_utils.Coordinate2D{X: c[2], Y: c[3]}})
		}
	}
	return slots, nil
}

func directionIndex(d game_utils.Coordinate2D) int {
	for i, v := range game_utils.AxialDirectionVectors() {
		if v == d {
			return i
		}
	}
	return 0
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
		home_space_keys[start.space.Key()] = true
	}
	slots, err := planStartResources(ctx, p.Resources, area_size)
	if err != nil {
		return err
	}
	for i, start := range starts {
		space := start.space
		footprint := make([]Space, 0)
		for _, s := range hexRadiusSpaces(space, area_size) {
			if s.Key() != space.Key() && home_space_keys[s.Key()] {
				continue
			}
			footprint = append(footprint, s)
		}
		for _, s := range footprint {
			s.ClearOccupants()
			terrain_id, err := p.resolve(ctx.rng)
			if err != nil {
				return err
			}
			s.SetTerrain(terrain_id)
		}
		footprint_zones := make([]Zone, 0)
		for _, s := range footprint {
			footprint_zones = append(footprint_zones, s.ShuffledEdgeZones(ctx.rng)...)
		}
		footprint_zones = util.ShuffleFrom(ctx.rng, footprint_zones)
		freeZoneWithin := func(lo int, hi int) Zone {
			for _, z := range footprint_zones {
				d := int(game_utils.AxialDistance(space.Coordinate(), z.Space().Coordinate()))
				if d >= lo && d <= hi && !z.Occupied() {
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
			if !ctx.board.PlaceBuilding(target, b.BuildingId, i) {
				continue
			}
			if b.TerrainOverride != 0 {
				target.SetTerrainOverride(b.TerrainOverride)
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
				ctx.board.PlaceUnit(target, u.UnitId, i)
			}
		}
		for _, o := range p.ZoneTerrainOverrides {
			target, err := resolveZoneTarget(footprint, space, o.Target, ctx.rng)
			if err != nil {
				return err
			}
			target.SetTerrainOverride(o.TerrainId)
		}
		in_footprint := make(map[uint]bool, len(footprint))
		for _, s := range footprint {
			in_footprint[s.Key()] = true
		}
		rotation := directionIndex(start.direction) - directionIndex(starts[0].direction)
		for _, slot := range slots {
			offset := rotateAxial(slot.space_offset, rotation)
			local := rotateAxial(slot.zone, rotation)
			var target Zone
			if s := ctx.board.Space(game_utils.Coordinate2D{X: space.Coordinate().X + offset.X, Y: space.Coordinate().Y + offset.Y}); s != nil && in_footprint[s.Key()] {
				target = s.Zone(local)
			}
			if target == nil || target.Occupied() {
				target = freeZoneWithin(slot.min_distance, slot.max_distance)
			}
			if target != nil {
				ctx.board.PlaceResource(target, slot.resource_id)
			}
		}
	}
	return nil
}
