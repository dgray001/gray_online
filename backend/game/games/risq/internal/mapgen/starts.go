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

func (r playerStartResourceJSON) resourceIds(config *defs.Config) ([]uint32, error) {
	ids := r.ResourceIds
	if ids == nil {
		ids = []uint32{r.ResourceId}
	} else if r.ResourceId != 0 || len(ids) == 0 {
		return nil, fmt.Errorf("start resource requires resource_id or a nonempty resource_ids list")
	}
	for _, id := range ids {
		if _, ok := config.Resources[id]; !ok {
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
	On terrainFilterJSON `json:"on,omitempty"`
	// overrides the terrain of each start space itself; the rest of the area uses the pick above
	HomeTerrainId ScriptExpr `json:"home_terrain_id"`
	// limits the default starting trees, groves (+10) and forests (+20) to these species
	TreeIds              []ScriptExpr              `json:"tree_ids"`
	Pattern              string                    `json:"pattern"`
	StartBalance         *ScriptExpr               `json:"start_balance,omitempty"`
	Nomad                *bool                     `json:"nomad,omitempty"`
	AreaSize             ScriptExpr                `json:"area_size"`
	StartingDistance     ScriptExpr                `json:"starting_distance"`
	InnerDistance        *ScriptExpr               `json:"inner_distance,omitempty"`
	MaxShift             *ScriptExpr               `json:"max_shift,omitempty"`
	RowInset             *ScriptExpr               `json:"row_inset,omitempty"`
	Inset                ScriptExpr                `json:"inset,omitempty"`
	ShortEdgeStarts      ScriptExpr                `json:"short_edge_starts,omitempty"`
	ShortEdgeBias        ScriptExpr                `json:"short_edge_bias,omitempty"`
	Assets               *playerStartAssets        `json:"assets,omitempty"`
	Units                []playerStartUnitJSON     `json:"-"`
	Resources            []playerStartResourceJSON `json:"resources"`
	Buildings            []playerStartBuildingJSON `json:"-"`
	ZoneTerrainOverrides []zoneTerrainOverrideJSON `json:"zone_terrain_overrides,omitempty"`
	// every player's starting stockpile; the game default when omitted
	StartingBank *StartingBank `json:"starting_bank,omitempty"`
}

type playerStartAssets struct {
	Units     []playerStartUnitJSON     `json:"units,omitempty"`
	Buildings []playerStartBuildingJSON `json:"buildings,omitempty"`
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
			if !ctx.startAllowed(ctx.board.Space(game_utils.Coordinate2D{X: home.X + dq, Y: home.Y + dr})) {
				return true
			}
		}
	}
	return false
}

func startFits(ctx *mapScriptContext, rule startAreaRule, home game_utils.Coordinate2D, assigned []playerStartInfo, area_size int, max_shift int, orig game_utils.Coordinate2D) bool {
	if rule <= startRuleNotInsideOthers {
		if startAreaClipped(ctx, home, area_size) {
			return false
		}
		if max_shift >= 0 && int(game_utils.AxialDistance(orig, home)) > max_shift {
			return false
		}
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
func nearestFittingSpaces(ctx *mapScriptContext, rule startAreaRule, original Space, assigned []playerStartInfo, area_size int, max_shift int) []Space {
	var nearest []Space
	nearest_distance := -1
	for _, s := range ctx.allSpaces() {
		if !ctx.startAllowed(s) || !startFits(ctx, rule, s.Coordinate(), assigned, area_size, max_shift, original.Coordinate()) {
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

func nearestStartSpace(ctx *mapScriptContext, original Space, assigned []playerStartInfo, area_size int, max_shift int) (Space, error) {
	for rule := startRuleClearOfOthers; rule <= startRuleDistinct; rule++ {
		nearest := nearestFittingSpaces(ctx, rule, original, assigned, area_size, max_shift)
		if len(nearest) == 1 {
			return nearest[0], nil
		}
		if len(nearest) > 1 {
			return nearest[ctx.rng.Intn(len(nearest))], nil
		}
	}
	used := make(map[uint]bool, len(assigned))
	for _, a := range assigned {
		used[a.space.Key()] = true
	}
	findBest := func(filterAllowed bool) Space {
		var best Space
		bestDist := -1
		for _, s := range ctx.allSpaces() {
			if used[s.Key()] || (filterAllowed && !ctx.startAllowed(s)) {
				continue
			}
			d := int(game_utils.AxialDistance(original.Coordinate(), s.Coordinate()))
			if bestDist == -1 || d < bestDist {
				best, bestDist = s, d
			}
		}
		return best
	}
	if s := findBest(true); s != nil {
		return s, nil
	}
	if s := findBest(false); s != nil {
		return s, nil
	}
	return nil, fmt.Errorf("no unused space left for player start near %v", original.Coordinate())
}

// Moves each pattern-chosen start to the nearest space passing the strictest rule any space passes
func assignStartSpaces(ctx *mapScriptContext, originals []playerStartInfo, area_size int, max_shift int) ([]playerStartInfo, error) {
	if len(ctx.allSpaces()) < len(originals) {
		return nil, fmt.Errorf("board has %d spaces, cannot place %d player starts", len(ctx.allSpaces()), len(originals))
	}
	assigned := make([]playerStartInfo, 0, len(originals))
	for _, original := range originals {
		space, err := nearestStartSpace(ctx, original.space, assigned, area_size, max_shift)
		if err != nil {
			return nil, err
		}
		assigned = append(assigned, playerStartInfo{space: space, direction: original.direction})
	}
	return assigned, nil
}

// Each player's start area, excluding other players' home spaces
func startFootprints(ctx *mapScriptContext, starts []playerStartInfo, area_size int) [][]Space {
	home_keys := make(map[uint]bool, len(starts))
	for _, start := range starts {
		home_keys[start.space.Key()] = true
	}
	footprints := make([][]Space, len(starts))
	for i, start := range starts {
		for _, s := range hexRadiusSpaces(start.space, area_size) {
			if (s.Key() == start.space.Key() || !home_keys[s.Key()]) && ctx.startAllowed(s) {
				footprints[i] = append(footprints[i], s)
			}
		}
	}
	return footprints
}

// Clears and re-terrains every start area space exactly once, before anything is placed
func prepareStartTerrain(ctx *mapScriptContext, pick terrainPickJSON, home ScriptExpr, starts []playerStartInfo, footprints [][]Space) (map[game_utils.Coordinate2D]uint32, error) {
	terrain := make(map[game_utils.Coordinate2D]uint32)
	for _, s := range footprints[0] {
		offset := game_utils.Coordinate2D{X: s.Coordinate().X - starts[0].space.Coordinate().X, Y: s.Coordinate().Y - starts[0].space.Coordinate().Y}
		terrain_id, err := pick.resolve(ctx)
		if home.provided() && offset == (game_utils.Coordinate2D{}) {
			terrain_id, err = resolveTerrainId(home, ctx)
		}
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
	for i, start := range starts {
		rotation := directionIndex(start.direction) - directionIndex(starts[0].direction)
		for _, s := range footprints[i] {
			rel := game_utils.Coordinate2D{X: s.Coordinate().X - start.space.Coordinate().X, Y: s.Coordinate().Y - start.space.Coordinate().Y}
			offset := rotateAxial(rel, -rotation)
			terrain_id, ok := terrain[offset]
			if !ok {
				var err error
				if terrain_id, err = pick.resolve(ctx); err != nil {
					return nil, err
				}
				terrain[offset] = terrain_id
			}
			s.SetTerrain(terrain_id)
		}
	}
	return terrain, nil
}

func startBuildingZone(ctx *mapScriptContext, home Space, footprint []Space, target zoneTargetJSON, area_zones []Zone, board_zones []Zone) (Zone, error) {
	planned, err := resolveZoneTarget(footprint, home, target, ctx.rng)
	if err != nil && !errors.Is(err, errTargetUnavailable) {
		return nil, err
	}
	from := home.Coordinate()
	if err == nil && ctx.startAllowed(planned.Space()) {
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
	if z := nearestFreeZone(ctx.allZones(), from, 0, math.MaxInt, ""); z != nil {
		return z, nil
	}
	return nil, fmt.Errorf("no free zone left for start building")
}

func startPlacementZone(ctx *mapScriptContext, start playerStartInfo, placement startPlacement) Zone {
	rotation := directionIndex(start.direction) - directionIndex(ctx.player_starts[0].direction)
	offset := rotateAxial(placement.offset, rotation)
	space := ctx.board.Space(game_utils.Coordinate2D{X: start.space.Coordinate().X + offset.X, Y: start.space.Coordinate().Y + offset.Y})
	if !ctx.startAllowed(space) {
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
	if z := nearestFreeZone(board_zones, from, 0, math.MaxInt, ""); z != nil {
		return z
	}
	return nearestFreeZone(ctx.allZones(), from, 0, math.MaxInt, "")
}

func placeStartContents(ctx *mapScriptContext, p playerStartsParams, starts []playerStartInfo, footprints [][]Space, first_player int) error {
	board_zones := ctx.startAllowedZones()
	layout := startLayout{}
	area_zones := make([]Zone, 0)
	for _, s := range footprints[0] {
		area_zones = append(area_zones, s.Zones()...)
	}
	area_zones = util.ShuffleFrom(ctx.rng, area_zones)
	for _, b := range p.Buildings {
		target, err := startBuildingZone(ctx, starts[0].space, footprints[0], b.Target, area_zones, board_zones)
		if err != nil || !ctx.board.PlaceBuilding(target, b.BuildingId, first_player) {
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
			ctx.board.PlaceUnit(target, u.UnitId, first_player)
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
			if target == nil || !ctx.board.PlaceBuilding(target, placement.building_id, first_player+i) {
				return fmt.Errorf("could not mirror start building %d", placement.building_id)
			}
			if placement.terrain_id != 0 {
				target.SetTerrainOverride(placement.terrain_id)
			}
		}
		for _, placement := range layout.units {
			z := startPlacementZone(ctx, starts[i], placement)
			if z == nil {
				z = nearestFreeZone(board_zones, starts[i].space.Coordinate(), 0, math.MaxInt, "")
			}
			if z == nil {
				z = nearestFreeZone(ctx.allZones(), starts[i].space.Coordinate(), 0, math.MaxInt, "")
			}
			if z != nil {
				ctx.board.PlaceUnit(z, placement.unit_id, first_player+i)
			}
		}
		for _, placement := range layout.overrides {
			if z := startPlacementZone(ctx, starts[i], placement); z != nil {
				z.SetTerrainOverride(placement.terrain_id)
			}
		}
	}
	return nil
}

func stepPlayerStarts(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[playerStartsParams](raw, "player_starts")
	if err != nil {
		return err
	}
	applyStartingDevelopmentDefaults(ctx, &p)
	if !p.On.empty() {
		matches, err := p.On.matcher(ctx)
		if err != nil {
			return err
		}
		ctx.start_eligible = make(map[uint]bool)
		for _, s := range ctx.allSpaces() {
			if matches(s) {
				ctx.start_eligible[s.Key()] = true
			}
		}
	}
	if p.Resources == nil {
		p.Resources = defaultStartingResources(defs.MapResourceLevel(ctx.vars["map_resources"]))
		if len(p.TreeIds) > 0 {
			if p.Resources, err = withTreeSpecies(ctx, p.Resources, p.TreeIds); err != nil {
				return err
			}
		}
	}
	if (p.Nomad == nil && ctx.vars["starting_development"] == float64(defs.StartingDevelopment_NOMAD)) || (p.Nomad != nil && *p.Nomad) {
		return placeNomadStarts(ctx, p)
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
	case "triangle":
		inset, insetErr := p.Inset.resolveInt(ctx.vars)
		if insetErr != nil {
			return insetErr
		}
		starts, err = resolveTrianglePlayerStarts(ctx, inset)
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
	case "asterisk":
		inner_distance := 0
		if p.InnerDistance != nil {
			inner_distance, err = p.InnerDistance.resolveInt(ctx.vars)
			if err != nil {
				return err
			}
		}
		starts, err = resolveAsteriskPlayerStarts(ctx, starting_distance, inner_distance)
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
	max_shift := -1
	if p.MaxShift != nil {
		if max_shift, err = p.MaxShift.resolveInt(ctx.vars); err != nil {
			return err
		}
	}
	if p.Pattern == "ring" || (p.Pattern == "rows" && (ctx.shape != "rectangle" || !p.On.empty())) {
		if starts, err = assignStartSpaces(ctx, starts, area_size, max_shift); err != nil {
			return err
		}
	}
	starts = util.ShuffleFrom(ctx.rng, starts)
	for i := range starts {
		if ctx.startAllowed(starts[i].space) {
			starts[0], starts[i] = starts[i], starts[0]
			break
		}
	}
	ctx.player_starts = starts
	ctx.player_area_size = area_size
	footprints := startFootprints(ctx, starts, area_size)
	if _, err := prepareStartTerrain(ctx, p.terrainPickJSON, p.HomeTerrainId, starts, footprints); err != nil {
		return err
	}
	if err := placeBalancedStarts(ctx, p, starts, footprints, area_size); err != nil {
		return err
	}
	if p.StartingBank != nil {
		for i := range starts {
			ctx.board.SetStartingBank(i, *p.StartingBank)
		}
	}
	return nil
}
