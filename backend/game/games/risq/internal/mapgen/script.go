package mapgen

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type mapScriptStepJSON struct {
	Step   string          `json:"step"`
	Params json.RawMessage `json:"params"`
}

type mapStepFunc func(ctx *mapScriptContext, raw json.RawMessage) error

type playerStartInfo struct {
	space     Space
	direction game_utils.Coordinate2D
}

type mapScriptContext struct {
	board         Board
	rng           *rand.Rand
	num_players   int
	board_size    uint16
	player_starts []playerStartInfo
	// radius (in spaces) of each player start area, set by the player_starts step
	player_area_size int
	regions          map[string]map[uint]bool
	vars             map[string]float64
	shape            string
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
	data, err := defs.ReadConfigFile("maps", "scripted", name+".json")
	if err != nil {
		return nil, fmt.Errorf("map script %q: %v", name, err)
	}
	var steps []mapScriptStepJSON
	if err := json.Unmarshal(data, &steps); err != nil {
		return nil, fmt.Errorf("failed to parse map script %q: %v", name, err)
	}
	return steps, nil
}

// Prints a map problem to stderr so it is visible even when generation carries on or the caller drops the error
func logMapProblem(err error) {
	fmt.Fprintln(os.Stderr, "map generation:", err)
}

// Rejects a player count no generator can place, instead of panicking inside one
func checkPlayerCount(num_players int) error {
	if num_players >= 1 {
		return nil
	}
	err := fmt.Errorf("need at least 1 player, got %d", num_players)
	logMapProblem(err)
	return err
}

// Builds the board from "custom:<name>" (a fixed map file) or "script:<name>" (a generator script)
func Generate(board Board, rng *rand.Rand, num_players int, map_name string) error {
	if err := checkPlayerCount(num_players); err != nil {
		return err
	}
	kind, name, _ := strings.Cut(map_name, ":")
	switch kind {
	case "custom":
		return generateCustom(board, num_players, name)
	case "script":
		return generateScripted(board, rng, num_players, name)
	}
	return fmt.Errorf("map %q must start with \"custom:\" or \"script:\"", map_name)
}

func generateScripted(board Board, rng *rand.Rand, num_players int, name string) error {
	steps, err := loadMapScript(name)
	if err != nil {
		return err
	}
	ctx := &mapScriptContext{
		board:       board,
		rng:         rng,
		num_players: num_players,
		regions:     make(map[string]map[uint]bool),
		vars:        newMapScriptVars(num_players, 0, 0),
	}
	if err := runMapScript(ctx, steps); err != nil {
		return err
	}
	if len(ctx.player_starts) != num_players {
		return errors.New("map script did not place all player starts")
	}
	return nil
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

func (c *mapScriptContext) allSpaces() []Space {
	return c.board.Spaces()
}

func (c *mapScriptContext) allZones() []Zone {
	zones := make([]Zone, 0)
	for _, space := range c.allSpaces() {
		zones = append(zones, space.Zones()...)
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
		"resource_place":       stepResourcePlace,
		"player_starts":        stepPlayerStarts,
		"define":               stepDefine,
		"rules":                stepRules,
		"regions_seven":        stepRegionsSeven,
		"shape":                stepShape,
	}
}

type rulesParams struct {
	StartingTechs         []uint32    `json:"starting_techs,omitempty"`
	UnlimitedPopulation   bool        `json:"unlimited_population,omitempty"`
	SpaceGoldIncome       *ScriptExpr `json:"space_gold_income,omitempty"`
	MercenariesNeedRegion *bool       `json:"mercenaries_need_region,omitempty"`
}

func stepRules(ctx *mapScriptContext, raw json.RawMessage) error {
	p, err := decodeStepParams[rulesParams](raw, "rules")
	if err != nil {
		return err
	}
	rules := mapRules{StartingTechs: p.StartingTechs, UnlimitedPopulation: p.UnlimitedPopulation, MercenariesNeedRegion: p.MercenariesNeedRegion}
	if p.SpaceGoldIncome != nil {
		gold, err := p.SpaceGoldIncome.resolve(ctx.vars)
		if err != nil {
			return err
		}
		rules.SpaceGoldIncome = &gold
	}
	return applyMapRules(ctx.board, ctx.num_players, rules)
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

// Decodes a step's JSON params into T, returning an error rather than panicking -- unlike other
// config, which is validated at server startup, a malformed map script is only caught at launch
// time, so a decode failure here must not be able to crash a live server.
func decodeStepParams[T any](raw json.RawMessage, step_name string) (T, error) {
	var p T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&p)
	if err != nil && strings.Contains(err.Error(), "unknown field") {
		logMapProblem(fmt.Errorf("%s: %v, ignored", step_name, err))
		var lenient T
		err = json.Unmarshal(raw, &lenient)
		p = lenient
	}
	if err != nil {
		return p, fmt.Errorf("%s: %v", step_name, err)
	}
	return p, nil
}
