package mapgen

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
)

type playerStartResourceJSON struct {
	ResourceId  uint32     `json:"resource_id"`
	ResourceIds []uint32   `json:"resource_ids,omitempty"`
	Count       ScriptExpr `json:"count"`
	Zone        string     `json:"zone,omitempty"`
	// optional [min, max] space distance from the start space: [0, 0] is the start space, [1, 1] the ring around it
	Distance []int `json:"distance,omitempty"`
}

func (r playerStartResourceJSON) resourceIds() ([]uint32, error) {
	ids := r.ResourceIds
	if ids == nil {
		ids = []uint32{r.ResourceId}
	} else if r.ResourceId != 0 || len(ids) == 0 {
		return nil, fmt.Errorf("start resource requires resource_id or a nonempty resource_ids list")
	}
	for _, id := range ids {
		if _, ok := defs.ResourceConfigs[id]; !ok {
			return nil, fmt.Errorf("unknown start resource id %d", id)
		}
	}
	return ids, nil
}

// One start resource relative to a player's start, laid out once and rotated to each player so starts are symmetric
type startResourceSlot struct {
	resource_id  uint32
	min_distance int
	max_distance int
	zone_kind    string
	space_offset game_utils.Coordinate2D
	zone         game_utils.Coordinate2D
	unplanned    bool // no distinct planned zone was left, so placement falls back to the nearest free zone
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
	RowInset             *ScriptExpr               `json:"row_inset,omitempty"`
	ShortEdgeStarts      ScriptExpr                `json:"short_edge_starts,omitempty"`
	ShortEdgeBias        ScriptExpr                `json:"short_edge_bias,omitempty"`
	Units                []playerStartUnitJSON     `json:"units,omitempty"`
	Resources            []playerStartResourceJSON `json:"resources"`
	Buildings            []playerStartBuildingJSON `json:"buildings"`
	ZoneTerrainOverrides []zoneTerrainOverrideJSON `json:"zone_terrain_overrides,omitempty"`
	// every player's starting stockpile; the game default when omitted
	StartingBank *StartingBank `json:"starting_bank,omitempty"`
}

type startPlacement struct {
	unit_id, building_id, terrain_id uint32
	offset, zone                     game_utils.Coordinate2D
}

type startLayout struct {
	units, buildings, overrides []startPlacement
}

var errTargetUnavailable = errors.New("start target unavailable")

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
			return nil, fmt.Errorf("%w: no space at distance %d from player start", errTargetUnavailable, target.SpaceDistance)
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
		return nil, fmt.Errorf("%w: no free edge zone available", errTargetUnavailable)
	}
	return free[rng.Intn(len(free))], nil
}

// Places players on one perimeter ring
func ringStartDirection(c game_utils.Coordinate2D) game_utils.Coordinate2D {
	directions := game_utils.AxialDirectionVectors()
	best := directions[0]
	bestDot := c.X*best.X + c.Y*best.Y
	for _, direction := range directions[1:] {
		dot := c.X*direction.X + c.Y*direction.Y
		if dot > bestDot {
			best, bestDot = direction, dot
		}
	}
	return best
}

func resolveRingPlayerStarts(ctx *mapScriptContext, starting_distance int) ([]playerStartInfo, error) {
	if starting_distance < 1 {
		return nil, fmt.Errorf("ring player starts require a positive starting distance")
	}
	candidates := make([]Space, 0)
	origin := game_utils.Coordinate2D{}
	for _, space := range ctx.allSpaces() {
		if int(game_utils.AxialDistance(origin, space.Coordinate())) == starting_distance {
			candidates = append(candidates, space)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("ring has no spaces at distance %d", starting_distance)
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i].Coordinate(), candidates[j].Coordinate()
		angleA := math.Atan2(float64(a.Y), float64(a.X))
		angleB := math.Atan2(float64(b.Y), float64(b.X))
		if angleA != angleB {
			return angleA < angleB
		}
		if a.X != b.X {
			return a.X < b.X
		}
		return a.Y < b.Y
	})
	// Rotate the evenly spaced pattern by the seeded RNG so maps retain orientation variation.
	offset := ctx.rng.Intn(len(candidates))
	starts := make([]playerStartInfo, ctx.num_players)
	for i := 0; i < ctx.num_players; i++ {
		index := (offset + (i*len(candidates))/ctx.num_players) % len(candidates)
		space := candidates[index]
		starts[i] = playerStartInfo{space: space, direction: ringStartDirection(space.Coordinate())}
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

func rowPlayerStartColumn(index int, n int, col_min int, col_max int) int {
	span := col_max - col_min
	return col_min + int(math.Round(float64(index*span)/float64(max(1, n-1))))
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

// Places an evenly spaced zigzag with a random starting row and horizontal direction.
func resolveRowsPlayerStarts(ctx *mapScriptContext, starting_distance, inset, short_edge_starts int, short_edge_bias float64) ([]playerStartInfo, error) {
	spaces := ctx.allSpaces()
	row_min, row_max, _, _ := rectangleSpaceBounds(spaces)
	half := max((row_max-row_min)/2, 1)
	dist := util.Clamp(starting_distance, 1, half)
	row_near := util.Clamp(-dist, row_min+inset, row_max-inset)
	row_far := util.Clamp(dist, row_min+inset, row_max-inset)
	n := ctx.num_players
	long_count := n - min(short_edge_starts, n)
	first_row := ctx.rng.Intn(2)
	reverse := ctx.rng.Intn(2) == 1
	starts := make([]playerStartInfo, n)
	for i := range starts {
		row, direction := row_near, game_utils.Coordinate2D{X: 0, Y: 1}
		if (i+first_row)%2 == 1 {
			row, direction = row_far, game_utils.Coordinate2D{X: 0, Y: -1}
		}
		if i >= long_count {
			row = int(math.Round(float64(row_near+row_far)/2 + short_edge_bias*float64((long_count%2)*(1-2*first_row))))
		}
		col_min, col_max, ok := rowSpaceBounds(spaces, row)
		if !ok {
			return nil, fmt.Errorf("no spaces at row %d for player starts", row)
		}
		col_min, col_max = col_min+inset, col_max-inset
		col := rowPlayerStartColumn(i, long_count, col_min, col_max)
		if i >= long_count {
			col = rowPlayerStartColumn(i-long_count, n-long_count, col_min, col_max)
		}
		if reverse {
			col = col_max - (col - col_min)
		}
		if i >= long_count {
			direction = game_utils.Coordinate2D{X: 1}
			if col == col_max {
				direction.X = -1
			}
		}
		q := col - floorDiv2(row)
		space := ctx.board.Space(game_utils.Coordinate2D{X: q, Y: row})
		if space == nil {
			return nil, fmt.Errorf("player start space is nil")
		}
		starts[i] = playerStartInfo{space: space, direction: direction}
	}
	return starts, nil
}

func planStartResources(ctx *mapScriptContext, resources []playerStartResourceJSON, area_size int) ([]startResourceSlot, error) {
	used := make(map[[4]int]bool)
	slots := make([]startResourceSlot, 0)
	for _, r := range resources {
		ids, err := r.resourceIds()
		if err != nil {
			return nil, err
		}
		first_slot := len(slots)
		count, err := r.Count.resolveInt(ctx.vars)
		if err != nil {
			return nil, err
		}
		if !validateResourceZone(r.Zone) {
			return nil, fmt.Errorf("resource %d: invalid zone %q", r.ResourceId, r.Zone)
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
				if resourceLocalMatches(game_utils.Coordinate2D{}, r.Zone) {
					candidates = append(candidates, [4]int{dq, dr, 0, 0})
				}
				for _, z := range game_utils.AxialDirectionVectors() {
					if resourceLocalMatches(z, r.Zone) {
						candidates = append(candidates, [4]int{dq, dr, z.X, z.Y})
					}
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
			slots = append(slots, startResourceSlot{resource_id: r.ResourceId, min_distance: lo, max_distance: hi, zone_kind: r.Zone,
				space_offset: game_utils.Coordinate2D{X: c[0], Y: c[1]}, zone: game_utils.Coordinate2D{X: c[2], Y: c[3]}})
		}
		for ; placed < count; placed++ {
			slots = append(slots, startResourceSlot{resource_id: r.ResourceId, min_distance: lo, max_distance: hi, zone_kind: r.Zone, unplanned: true})
		}
		for i := first_slot; i < len(slots); i++ {
			slots[i].resource_id = ids[0]
			if len(ids) > 1 {
				slots[i].resource_id = ids[ctx.rng.Intn(len(ids))]
			}
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

type startAreaRule int

// Start placement rules from strictest to loosest; each is tried only after the previous finds no space
const (
	startRuleClearOfOthers   startAreaRule = iota // area fully on the board and overlapping no other start area
	startRuleNotInsideOthers                      // area fully on the board, home not inside another start area
	startRuleAllowClipped                         // home not inside another start area
	startRuleDistinct                             // home space differs from every other home
)

func startAreaClipped(ctx *mapScriptContext, home game_utils.Coordinate2D, area_size int) bool {
	for dq := -area_size; dq <= area_size; dq++ {
		for dr := max(-area_size, -dq-area_size); dr <= min(area_size, -dq+area_size); dr++ {
			if ctx.board.Space(game_utils.Coordinate2D{X: home.X + dq, Y: home.Y + dr}) == nil {
				return true
			}
		}
	}
	return false
}

func startFits(ctx *mapScriptContext, rule startAreaRule, home game_utils.Coordinate2D, assigned []playerStartInfo, area_size int) bool {
	if rule <= startRuleNotInsideOthers && startAreaClipped(ctx, home, area_size) {
		return false
	}
	min_distance := 1
	switch rule {
	case startRuleClearOfOthers:
		min_distance = 2*area_size + 1
	case startRuleNotInsideOthers, startRuleAllowClipped:
		min_distance = area_size + 1
	}
	for _, other := range assigned {
		if ctx.board.Distance(ctx.board.Space(home), other.space) < min_distance {
			return false
		}
	}
	return true
}

// Every space tied for nearest to original among those satisfying rule, in key order
func nearestFittingSpaces(ctx *mapScriptContext, rule startAreaRule, original Space, assigned []playerStartInfo, area_size int) []Space {
	var nearest []Space
	nearest_distance := -1
	for _, s := range ctx.allSpaces() {
		if !startFits(ctx, rule, s.Coordinate(), assigned, area_size) {
			continue
		}
		d := int(game_utils.AxialDistance(original.Coordinate(), s.Coordinate()))
		if nearest_distance == -1 || d < nearest_distance {
			nearest, nearest_distance = nil, d
		}
		if d == nearest_distance {
			nearest = append(nearest, s)
		}
	}
	sort.Slice(nearest, func(i, j int) bool { return nearest[i].Key() < nearest[j].Key() })
	return nearest
}

func nearestStartSpace(ctx *mapScriptContext, original Space, assigned []playerStartInfo, area_size int) (Space, error) {
	for rule := startRuleClearOfOthers; rule <= startRuleDistinct; rule++ {
		nearest := nearestFittingSpaces(ctx, rule, original, assigned, area_size)
		if len(nearest) == 1 {
			return nearest[0], nil
		}
		if len(nearest) > 1 {
			return nearest[ctx.rng.Intn(len(nearest))], nil
		}
	}
	return nil, fmt.Errorf("no unused space left for player start near %v", original.Coordinate())
}

// Moves each pattern-chosen start to the nearest space passing the strictest rule any space passes
func assignStartSpaces(ctx *mapScriptContext, originals []playerStartInfo, area_size int) ([]playerStartInfo, error) {
	if len(ctx.allSpaces()) < len(originals) {
		return nil, fmt.Errorf("board has %d spaces, cannot place %d player starts", len(ctx.allSpaces()), len(originals))
	}
	assigned := make([]playerStartInfo, 0, len(originals))
	for _, original := range originals {
		space, err := nearestStartSpace(ctx, original.space, assigned, area_size)
		if err != nil {
			return nil, err
		}
		assigned = append(assigned, playerStartInfo{space: space, direction: original.direction})
	}
	return assigned, nil
}

// Each player's start area, excluding other players' home spaces
func startFootprints(starts []playerStartInfo, area_size int) [][]Space {
	home_keys := make(map[uint]bool, len(starts))
	for _, start := range starts {
		home_keys[start.space.Key()] = true
	}
	footprints := make([][]Space, len(starts))
	for i, start := range starts {
		for _, s := range hexRadiusSpaces(start.space, area_size) {
			if s.Key() == start.space.Key() || !home_keys[s.Key()] {
				footprints[i] = append(footprints[i], s)
			}
		}
	}
	return footprints
}

// Clears and re-terrains every start area space exactly once, before anything is placed
func prepareStartTerrain(ctx *mapScriptContext, pick terrainPickJSON, starts []playerStartInfo, footprints [][]Space) (map[game_utils.Coordinate2D]uint32, error) {
	terrain := make(map[game_utils.Coordinate2D]uint32)
	for _, s := range footprints[0] {
		offset := game_utils.Coordinate2D{X: s.Coordinate().X - starts[0].space.Coordinate().X, Y: s.Coordinate().Y - starts[0].space.Coordinate().Y}
		terrain_id, err := pick.resolve(ctx.rng)
		if err != nil {
			return nil, err
		}
		terrain[offset] = terrain_id
	}
	for _, footprint := range footprints {
		for _, s := range footprint {
			s.ClearOccupants()
		}
	}
	for _, start := range starts {
		rotation := directionIndex(start.direction) - directionIndex(starts[0].direction)
		for offset, terrain_id := range terrain {
			rotated := rotateAxial(offset, rotation)
			if s := ctx.board.Space(game_utils.Coordinate2D{X: start.space.Coordinate().X + rotated.X, Y: start.space.Coordinate().Y + rotated.Y}); s != nil {
				s.SetTerrain(terrain_id)
			}
		}
	}
	return terrain, nil
}

// The free zone nearest from whose space is within [lo, hi] spaces of it; the first of any tie in zones' order
func nearestFreeZone(zones []Zone, from game_utils.Coordinate2D, lo int, hi int, selector string) Zone {
	var nearest Zone
	nearest_distance := -1
	for _, z := range zones {
		d := int(game_utils.AxialDistance(from, z.Space().Coordinate()))
		if z.Occupied() || !resourceZoneMatches(z, selector) || d < lo || d > hi || (nearest_distance != -1 && d >= nearest_distance) {
			continue
		}
		nearest, nearest_distance = z, d
	}
	return nearest
}

func plannedResourceZone(ctx *mapScriptContext, home game_utils.Coordinate2D, rotation int, slot startResourceSlot, in_area map[uint]bool) Zone {
	offset := rotateAxial(slot.space_offset, rotation)
	s := ctx.board.Space(game_utils.Coordinate2D{X: home.X + offset.X, Y: home.Y + offset.Y})
	if slot.unplanned || s == nil || !in_area[s.Key()] {
		return nil
	}
	if z := s.Zone(rotateAxial(slot.zone, rotation)); z != nil && !z.Occupied() && resourceZoneMatches(z, slot.zone_kind) {
		return z
	}
	return nil
}

func startResourceZone(ctx *mapScriptContext, home game_utils.Coordinate2D, rotation int, slot startResourceSlot, in_area map[uint]bool, area_zones []Zone, board_zones []Zone) Zone {
	if z := plannedResourceZone(ctx, home, rotation, slot, in_area); z != nil {
		return z
	}
	if z := nearestFreeZone(area_zones, home, slot.min_distance, slot.max_distance, slot.zone_kind); z != nil {
		return z
	}
	if z := nearestFreeZone(area_zones, home, 0, math.MaxInt, slot.zone_kind); z != nil {
		return z
	}
	return nearestFreeZone(board_zones, home, 0, math.MaxInt, slot.zone_kind)
}

// The mirrored zone when it is free, else the free zone nearest it (in the player's area first)
func mirroredResourceZone(ctx *mapScriptContext, mirrored game_utils.Coordinate2D, rotation int, slot startResourceSlot, area_zones []Zone, board_zones []Zone) Zone {
	if s := ctx.board.Space(mirrored); s != nil {
		if z := s.Zone(rotateAxial(slot.zone, rotation)); z != nil && !z.Occupied() {
			return z
		}
	}
	if z := nearestFreeZone(area_zones, mirrored, 0, math.MaxInt, slot.zone_kind); z != nil {
		return z
	}
	return nearestFreeZone(board_zones, mirrored, 0, math.MaxInt, slot.zone_kind)
}

// Places every start resource for every player, erroring only when the board has no free zone left
func placeStartResources(ctx *mapScriptContext, starts []playerStartInfo, footprints [][]Space, slots []startResourceSlot) error {
	board_zones := ctx.allZones()
	for i, start := range starts {
		in_area := make(map[uint]bool, len(footprints[i]))
		area_zones := make([]Zone, 0)
		for _, s := range footprints[i] {
			in_area[s.Key()] = true
			area_zones = append(area_zones, s.Zones()...)
		}
		area_zones = util.ShuffleFrom(ctx.rng, area_zones)
		rotation := directionIndex(start.direction) - directionIndex(starts[0].direction)
		for slot_index := range slots {
			slot := &slots[slot_index]
			zone := startResourceZone(ctx, start.space.Coordinate(), rotation, *slot, in_area, area_zones, board_zones)
			if zone == nil {
				return fmt.Errorf("no free zone left for start resource %d", slot.resource_id)
			}
			if i == 0 {
				slot.space_offset = game_utils.Coordinate2D{X: zone.Space().Coordinate().X - start.space.Coordinate().X, Y: zone.Space().Coordinate().Y - start.space.Coordinate().Y}
				slot.zone = zone.Local()
				slot.unplanned = false
			} else {
				rotated := rotateAxial(slot.space_offset, rotation)
				mirrored := game_utils.Coordinate2D{X: start.space.Coordinate().X + rotated.X, Y: start.space.Coordinate().Y + rotated.Y}
				if zone = mirroredResourceZone(ctx, mirrored, rotation, *slot, area_zones, board_zones); zone == nil {
					return fmt.Errorf("no free zone left for start resource %d", slot.resource_id)
				}
			}
			ctx.board.PlaceResource(zone, slot.resource_id)
		}
	}
	return nil
}

func startBuildingZone(ctx *mapScriptContext, home Space, footprint []Space, target zoneTargetJSON, area_zones []Zone, board_zones []Zone) (Zone, error) {
	planned, err := resolveZoneTarget(footprint, home, target, ctx.rng)
	if err != nil && !errors.Is(err, errTargetUnavailable) {
		return nil, err
	}
	from := home.Coordinate()
	if err == nil {
		if !planned.Occupied() {
			return planned, nil
		}
		from = planned.Space().Coordinate()
	}
	if z := nearestFreeZone(area_zones, from, 0, math.MaxInt, ""); z != nil {
		return z, nil
	}
	if z := nearestFreeZone(board_zones, from, 0, math.MaxInt, ""); z != nil {
		return z, nil
	}
	return nil, fmt.Errorf("no free zone left for start building")
}

func startPlacementZone(ctx *mapScriptContext, start playerStartInfo, placement startPlacement) Zone {
	rotation := directionIndex(start.direction) - directionIndex(ctx.player_starts[0].direction)
	offset := rotateAxial(placement.offset, rotation)
	space := ctx.board.Space(game_utils.Coordinate2D{X: start.space.Coordinate().X + offset.X, Y: start.space.Coordinate().Y + offset.Y})
	if space == nil {
		return nil
	}
	return space.Zone(rotateAxial(placement.zone, rotation))
}

// The mirrored zone when it is free, else the free zone nearest its space (in the player's area first)
func mirroredBuildingZone(ctx *mapScriptContext, start playerStartInfo, placement startPlacement, footprint []Space, board_zones []Zone) Zone {
	if z := startPlacementZone(ctx, start, placement); z != nil && !z.Occupied() {
		return z
	}
	rotation := directionIndex(start.direction) - directionIndex(ctx.player_starts[0].direction)
	offset := rotateAxial(placement.offset, rotation)
	from := game_utils.Coordinate2D{X: start.space.Coordinate().X + offset.X, Y: start.space.Coordinate().Y + offset.Y}
	area_zones := make([]Zone, 0)
	for _, s := range footprint {
		area_zones = append(area_zones, s.Zones()...)
	}
	if z := nearestFreeZone(area_zones, from, 0, math.MaxInt, ""); z != nil {
		return z
	}
	return nearestFreeZone(board_zones, from, 0, math.MaxInt, "")
}

func placeStartContents(ctx *mapScriptContext, p playerStartsParams, starts []playerStartInfo, footprints [][]Space) error {
	board_zones := ctx.allZones()
	layout := startLayout{}
	area_zones := make([]Zone, 0)
	for _, s := range footprints[0] {
		area_zones = append(area_zones, s.Zones()...)
	}
	area_zones = util.ShuffleFrom(ctx.rng, area_zones)
	for _, b := range p.Buildings {
		target, err := startBuildingZone(ctx, starts[0].space, footprints[0], b.Target, area_zones, board_zones)
		if err != nil || !ctx.board.PlaceBuilding(target, b.BuildingId, 0) {
			return fmt.Errorf("could not place start building %d", b.BuildingId)
		}
		placement := startPlacement{building_id: b.BuildingId, offset: game_utils.Coordinate2D{X: target.Space().Coordinate().X - starts[0].space.Coordinate().X, Y: target.Space().Coordinate().Y - starts[0].space.Coordinate().Y}, zone: target.Local(), terrain_id: b.TerrainOverride}
		layout.buildings = append(layout.buildings, placement)
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
			target, err := resolveZoneTarget(footprints[0], starts[0].space, u.Target, ctx.rng)
			if err != nil {
				return err
			}
			layout.units = append(layout.units, startPlacement{unit_id: u.UnitId, offset: game_utils.Coordinate2D{X: target.Space().Coordinate().X - starts[0].space.Coordinate().X, Y: target.Space().Coordinate().Y - starts[0].space.Coordinate().Y}, zone: target.Local()})
			ctx.board.PlaceUnit(target, u.UnitId, 0)
		}
	}
	for _, o := range p.ZoneTerrainOverrides {
		target, err := resolveZoneTarget(footprints[0], starts[0].space, o.Target, ctx.rng)
		if err != nil {
			return err
		}
		layout.overrides = append(layout.overrides, startPlacement{terrain_id: o.TerrainId, offset: game_utils.Coordinate2D{X: target.Space().Coordinate().X - starts[0].space.Coordinate().X, Y: target.Space().Coordinate().Y - starts[0].space.Coordinate().Y}, zone: target.Local()})
		target.SetTerrainOverride(o.TerrainId)
	}
	for i := 1; i < len(starts); i++ {
		for _, placement := range layout.buildings {
			target := mirroredBuildingZone(ctx, starts[i], placement, footprints[i], board_zones)
			if target == nil || !ctx.board.PlaceBuilding(target, placement.building_id, i) {
				return fmt.Errorf("could not mirror start building %d", placement.building_id)
			}
			if placement.terrain_id != 0 {
				target.SetTerrainOverride(placement.terrain_id)
			}
		}
		for _, placement := range layout.units {
			ctx.board.PlaceUnit(startPlacementZone(ctx, starts[i], placement), placement.unit_id, i)
		}
		for _, placement := range layout.overrides {
			startPlacementZone(ctx, starts[i], placement).SetTerrainOverride(placement.terrain_id)
		}
	}
	return nil
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
		inset := 1
		if p.RowInset != nil {
			inset, err = p.RowInset.resolveInt(ctx.vars)
			if err != nil {
				return err
			}
		}
		short_edge_starts := 0
		short_edge_starts, err = p.ShortEdgeStarts.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		short_edge_bias := 0.0
		short_edge_bias, err = p.ShortEdgeBias.resolve(ctx.vars)
		if err != nil {
			return err
		}
		starts, err = resolveRowsPlayerStarts(ctx, starting_distance, inset, short_edge_starts, short_edge_bias)
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
	if ctx.shape != "rectangle" || p.Pattern != "rows" {
		if starts, err = assignStartSpaces(ctx, starts, area_size); err != nil {
			return err
		}
	}
	starts = util.ShuffleFrom(ctx.rng, starts)
	ctx.player_starts = starts
	ctx.player_area_size = area_size
	footprints := startFootprints(starts, area_size)
	if _, err := prepareStartTerrain(ctx, p.terrainPickJSON, starts, footprints); err != nil {
		return err
	}
	slots, err := planStartResources(ctx, p.Resources, area_size)
	if err != nil {
		return err
	}
	if err := placeStartContents(ctx, p, starts, footprints); err != nil {
		return err
	}
	if err := placeStartResources(ctx, starts, footprints, slots); err != nil {
		return err
	}
	if p.StartingBank != nil {
		for i := range starts {
			ctx.board.SetStartingBank(i, *p.StartingBank)
		}
	}
	return nil
}
