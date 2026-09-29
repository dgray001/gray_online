package mapgen

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"

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
	Units                []playerStartUnitJSON     `json:"units,omitempty"`
	Resources            []playerStartResourceJSON `json:"resources"`
	Buildings            []playerStartBuildingJSON `json:"buildings"`
	ZoneTerrainOverrides []zoneTerrainOverrideJSON `json:"zone_terrain_overrides,omitempty"`
	// every player's starting stockpile; the game default when omitted
	StartingBank *StartingBank `json:"starting_bank,omitempty"`
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

// Places players on one perimeter ring. The six principal directions and the spaces between them
// are treated uniformly, so two players are opposite and twelve players form an evenly spaced
// approximation of a dodecagon while every start remains the same distance from the center.
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
		for ; placed < count; placed++ {
			slots = append(slots, startResourceSlot{resource_id: r.ResourceId, min_distance: lo, max_distance: hi, unplanned: true})
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
		if int(game_utils.AxialDistance(home, other.space.Coordinate())) < min_distance {
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
func prepareStartTerrain(ctx *mapScriptContext, pick terrainPickJSON, footprints [][]Space) error {
	prepared := make(map[uint]bool)
	for _, footprint := range footprints {
		for _, s := range footprint {
			if prepared[s.Key()] {
				continue
			}
			prepared[s.Key()] = true
			terrain_id, err := pick.resolve(ctx.rng)
			if err != nil {
				return err
			}
			s.ClearOccupants()
			s.SetTerrain(terrain_id)
		}
	}
	return nil
}

// The free zone nearest from whose space is within [lo, hi] spaces of it; the first of any tie in zones' order
func nearestFreeZone(zones []Zone, from game_utils.Coordinate2D, lo int, hi int) Zone {
	var nearest Zone
	nearest_distance := -1
	for _, z := range zones {
		d := int(game_utils.AxialDistance(from, z.Space().Coordinate()))
		if z.Occupied() || d < lo || d > hi || (nearest_distance != -1 && d >= nearest_distance) {
			continue
		}
		nearest, nearest_distance = z, d
	}
	return nearest
}

func edgeZones(zones []Zone) []Zone {
	edges := make([]Zone, 0, len(zones))
	for _, z := range zones {
		if !z.IsCenter() {
			edges = append(edges, z)
		}
	}
	return edges
}

func plannedResourceZone(ctx *mapScriptContext, home game_utils.Coordinate2D, rotation int, slot startResourceSlot, in_area map[uint]bool) Zone {
	offset := rotateAxial(slot.space_offset, rotation)
	s := ctx.board.Space(game_utils.Coordinate2D{X: home.X + offset.X, Y: home.Y + offset.Y})
	if slot.unplanned || s == nil || !in_area[s.Key()] {
		return nil
	}
	if z := s.Zone(rotateAxial(slot.zone, rotation)); z != nil && !z.Occupied() {
		return z
	}
	return nil
}

func startResourceZone(ctx *mapScriptContext, home game_utils.Coordinate2D, rotation int, slot startResourceSlot, in_area map[uint]bool, area_zones []Zone, board_zones []Zone) Zone {
	if z := plannedResourceZone(ctx, home, rotation, slot, in_area); z != nil {
		return z
	}
	if z := nearestFreeZone(area_zones, home, slot.min_distance, slot.max_distance); z != nil {
		return z
	}
	if z := nearestFreeZone(area_zones, home, 0, math.MaxInt); z != nil {
		return z
	}
	return nearestFreeZone(board_zones, home, 0, math.MaxInt)
}

// Places every start resource for every player, erroring only when the board has no free zone left
func placeStartResources(ctx *mapScriptContext, starts []playerStartInfo, footprints [][]Space, slots []startResourceSlot) error {
	board_zones := edgeZones(ctx.allZones())
	for i, start := range starts {
		in_area := make(map[uint]bool, len(footprints[i]))
		area_zones := make([]Zone, 0)
		for _, s := range footprints[i] {
			in_area[s.Key()] = true
			area_zones = append(area_zones, s.ShuffledEdgeZones(ctx.rng)...)
		}
		area_zones = util.ShuffleFrom(ctx.rng, area_zones)
		rotation := directionIndex(start.direction) - directionIndex(starts[0].direction)
		for _, slot := range slots {
			zone := startResourceZone(ctx, start.space.Coordinate(), rotation, slot, in_area, area_zones, board_zones)
			if zone == nil {
				return fmt.Errorf("no free zone left for start resource %d", slot.resource_id)
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
	if z := nearestFreeZone(area_zones, from, 0, math.MaxInt); z != nil {
		return z, nil
	}
	if z := nearestFreeZone(board_zones, from, 0, math.MaxInt); z != nil {
		return z, nil
	}
	return nil, fmt.Errorf("no free zone left for start building")
}

func placeStartUnits(ctx *mapScriptContext, p playerStartsParams, player_index int, home Space, footprint []Space) error {
	for _, u := range p.Units {
		count, err := u.Count.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		for range count {
			target, err := resolveZoneTarget(footprint, home, u.Target, ctx.rng)
			if err != nil {
				return err
			}
			ctx.board.PlaceUnit(target, u.UnitId, player_index)
		}
	}
	for _, o := range p.ZoneTerrainOverrides {
		target, err := resolveZoneTarget(footprint, home, o.Target, ctx.rng)
		if err != nil {
			return err
		}
		target.SetTerrainOverride(o.TerrainId)
	}
	return nil
}

func placeStartContents(ctx *mapScriptContext, p playerStartsParams, starts []playerStartInfo, footprints [][]Space) error {
	board_zones := ctx.allZones()
	for i, start := range starts {
		area_zones := make([]Zone, 0)
		for _, s := range footprints[i] {
			area_zones = append(area_zones, s.Zones()...)
		}
		area_zones = util.ShuffleFrom(ctx.rng, area_zones)
		for _, b := range p.Buildings {
			target, err := startBuildingZone(ctx, start.space, footprints[i], b.Target, area_zones, board_zones)
			if err != nil {
				return err
			}
			if !ctx.board.PlaceBuilding(target, b.BuildingId, i) {
				return fmt.Errorf("could not place start building %d", b.BuildingId)
			}
			if b.TerrainOverride != 0 {
				target.SetTerrainOverride(b.TerrainOverride)
			}
		}
		if err := placeStartUnits(ctx, p, i, start.space, footprints[i]); err != nil {
			return err
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
	if starts, err = assignStartSpaces(ctx, starts, area_size); err != nil {
		return err
	}
	ctx.player_starts = starts
	ctx.player_area_size = area_size
	footprints := startFootprints(starts, area_size)
	if err := prepareStartTerrain(ctx, p.terrainPickJSON, footprints); err != nil {
		return err
	}
	slots, err := planStartResources(ctx, p.Resources, area_size)
	if err != nil {
		return err
	}
	if err := placeStartResources(ctx, starts, footprints, slots); err != nil {
		return err
	}
	if err := placeStartContents(ctx, p, starts, footprints); err != nil {
		return err
	}
	if p.StartingBank != nil {
		for i := range starts {
			ctx.board.SetStartingBank(i, *p.StartingBank)
		}
	}
	return nil
}
