package risq

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"path"

	"github.com/dgray001/gray_online/game/game_utils"
)

//go:embed config/maps/scripted/*
var mapScripts embed.FS

//go:embed config/maps/scripted/default.json
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
	risq                *GameRisq
	rng                 *rand.Rand
	num_players         int
	board_size          uint16
	recommended_size    *uint16 // overrides the num_players-derived default when set
	player_starts       []playerStartInfo
	regions             map[string]map[uint]bool
	vars                map[string]float64
	shape               string
}

func recommendedBoardSize(num_players int) uint16 {
	switch {
	case num_players <= 3:
		return 4
	case num_players == 4:
		return 5
	case num_players <= 6:
		return 6
	default:
		return uint16(6 + (num_players-5)/2)
	}
}

func newMapScriptVars(num_players int, board_size uint16, total_spaces int) map[string]float64 {
	return map[string]float64{
		"num_players":  float64(num_players),
		"board_size":   float64(board_size),
		"total_spaces": float64(total_spaces),
		"total_zones":  float64(total_spaces * 7),
	}
}

func loadMapScript(name string) ([]mapScriptStepJSON, error) {
	data, err := mapScripts.ReadFile(path.Join("config/maps/scripted", name+".json"))
	if err != nil {
		data = defaultMapScript
	}
	var steps []mapScriptStepJSON
	if err := json.Unmarshal(data, &steps); err != nil {
		return nil, fmt.Errorf("failed to parse map script %q: %v", name, err)
	}
	return steps, nil
}

func runMapScript(ctx *mapScriptContext, steps []mapScriptStepJSON) error {
	if len(steps) == 0 || steps[0].Step != "shape" {
		return fmt.Errorf("map script must start with a shape step")
	}
	for i, s := range steps {
		if i > 0 && s.Step == "shape" {
			return fmt.Errorf("shape step must be the first step in the script")
		}
		fn, ok := mapStepRegistry[s.Step]
		if !ok {
			return fmt.Errorf("unknown map script step %q", s.Step)
		}
		if err := fn(ctx, s.Params); err != nil {
			return fmt.Errorf("map script step %q: %w", s.Step, err)
		}
		if s.Step == "shape" {
			total_spaces := len(ctx.allSpaces())
			ctx.vars["total_spaces"] = float64(total_spaces)
			ctx.vars["total_zones"] = float64(total_spaces * 7)
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
		for _, space := range row {
			if space != nil {
				spaces = append(spaces, space)
			}
		}
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
		"regions_seven":        stepRegionsSeven,
		"shape":                stepShape,
	}
}
