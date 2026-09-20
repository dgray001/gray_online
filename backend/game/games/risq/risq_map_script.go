package risq

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"path"

	"github.com/dgray001/gray_online/game/game_utils"
)

//go:embed config/maps/*
var mapScripts embed.FS

//go:embed config/maps/default.json
var defaultMapScript []byte

type mapScriptStepJSON struct {
	Step   string          `json:"step"`
	Params json.RawMessage `json:"params"`
}

type mapStepFunc func(ctx *mapScriptContext, raw json.RawMessage) error

type playerStartInfo struct {
	space     *RisqSpace
	direction game_utils.Coordinate2D
}

type mapScriptContext struct {
	risq              *GameRisq
	rng               *rand.Rand
	num_players       int
	board_size        uint16
	starting_distance int
	player_starts     []playerStartInfo
	regions           map[string]map[uint]bool
	vars              map[string]float64
}

func newMapScriptVars(num_players int, board_size uint16, starting_distance int, total_spaces int) map[string]float64 {
	return map[string]float64{
		"num_players":       float64(num_players),
		"board_size":        float64(board_size),
		"starting_distance": float64(starting_distance),
		"total_spaces":      float64(total_spaces),
		"total_zones":       float64(total_spaces * 7),
	}
}

// TODO: step param decode panics happen at game-launch time (unlike other config, which panics
// at server init before any player connects); a malformed non-default map crashes the live server
func loadMapScript(name string) []mapScriptStepJSON {
	data, err := mapScripts.ReadFile(path.Join("config/maps", name+".json"))
	if err != nil {
		data = defaultMapScript
	}
	var steps []mapScriptStepJSON
	if err := json.Unmarshal(data, &steps); err != nil {
		panic(fmt.Sprintf("failed to parse map script %q: %v", name, err))
	}
	return steps
}

func runMapScript(ctx *mapScriptContext, steps []mapScriptStepJSON) error {
	for _, s := range steps {
		fn, ok := mapStepRegistry[s.Step]
		if !ok {
			return fmt.Errorf("unknown map script step %q", s.Step)
		}
		if err := fn(ctx, s.Params); err != nil {
			return fmt.Errorf("map script step %q: %w", s.Step, err)
		}
	}
	return nil
}

// Lazily creates the named region for shaping steps to write membership into
func (c *mapScriptContext) region(name string) map[uint]bool {
	if name == "" {
		return nil
	}
	r, ok := c.regions[name]
	if !ok {
		r = make(map[uint]bool)
		c.regions[name] = r
	}
	return r
}

func (c *mapScriptContext) allSpaces() []*RisqSpace {
	spaces := make([]*RisqSpace, 0)
	for _, row := range c.risq.spaces {
		spaces = append(spaces, row...)
	}
	return spaces
}

func (c *mapScriptContext) allZones() []*RisqZone {
	zones := make([]*RisqZone, 0)
	for _, space := range c.allSpaces() {
		for _, row := range space.zones {
			zones = append(zones, row...)
		}
	}
	return zones
}

// Rotates an axial coordinate by steps increments of sixty degrees
func rotateAxial(c game_utils.Coordinate2D, steps int) game_utils.Coordinate2D {
	x, z := c.X, c.Y
	y := -x - z
	n := ((steps % 6) + 6) % 6
	for range n {
		x, y, z = -y, -z, -x
	}
	return game_utils.Coordinate2D{X: x, Y: z}
}

// Replays a wrapped step's board changes rotated to every other player start direction
func stepMirror(ctx *mapScriptContext, raw json.RawMessage) error {
	var p struct {
		Step mapScriptStepJSON `json:"step"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(fmt.Sprintf("map script mirror: %v", err))
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

var mapStepRegistry map[string]mapStepFunc

func init() {
	mapStepRegistry = map[string]mapStepFunc{
		"terrain_fill":         stepTerrainFill,
		"terrain_blob":         stepTerrainBlob,
		"terrain_line":         stepTerrainLine,
		"terrain_border":       stepTerrainBorder,
		"ensure_connectivity":  stepEnsureConnectivity,
		"resource_scatter":     stepResourceScatter,
		"resource_cluster":     stepResourceCluster,
		"resource_min_spacing": stepResourceMinSpacing,
		"mirror":               stepMirror,
		"player_starts":        stepPlayerStarts,
		"define":               stepDefine,
	}
}
