package mapgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type customBuilding struct {
	BuildingId    uint32   `json:"id"`
	Player        int      `json:"player"`
	ResourcesLeft *float64 `json:"resources_left,omitempty"`
}

type customUnit struct {
	UnitId uint32 `json:"id"`
	Player int    `json:"player"`
	// a number or an expression over the setup variables
	Count ScriptExpr `json:"count"`
}

// A starting stockpile whose amounts may be expressions over the setup variables
type customBank struct {
	Food  ScriptExpr `json:"food"`
	Wood  ScriptExpr `json:"wood"`
	Stone ScriptExpr `json:"stone"`
	Gold  ScriptExpr `json:"gold"`
}

func (b customBank) resolve(vars map[string]float64) (StartingBank, error) {
	var bank StartingBank
	amounts := []struct {
		expr   ScriptExpr
		target *float64
	}{{b.Food, &bank.Food}, {b.Wood, &bank.Wood}, {b.Stone, &bank.Stone}, {b.Gold, &bank.Gold}}
	for _, amount := range amounts {
		value, err := amount.expr.resolve(vars)
		if err != nil {
			return bank, err
		}
		*amount.target = value
	}
	return bank, nil
}

type customZone struct {
	X               int             `json:"x"`
	Y               int             `json:"y"`
	ResourceId      uint32          `json:"resource,omitempty"`
	TerrainOverride uint32          `json:"terrain_override,omitempty"`
	Building        *customBuilding `json:"building,omitempty"`
	Units           []customUnit    `json:"units,omitempty"`
}

type customSpace struct {
	X          int          `json:"x"`
	Y          int          `json:"y"`
	Terrain    uint32       `json:"terrain,omitempty"`
	Name       string       `json:"name,omitempty"`
	GoldIncome *float64     `json:"gold_income,omitempty"`
	Zones      []customZone `json:"zones,omitempty"`
	PlayerSlot *int         `json:"player_slot,omitempty"`
}

const maxCustomSpaceNameLength = 40

type customRegion struct {
	Name      string   `json:"name"`
	GoldBonus float64  `json:"gold_bonus,omitempty"`
	Spaces    [][2]int `json:"spaces"`
}

type customConnection struct {
	From [2]int `json:"from"`
	To   [2]int `json:"to"`
	// required, 0 to 5: the edge of From the link leaves through
	Direction *int `json:"direction"`
	// optional, 0 to 5: the edge of To it arrives at; defaults to the edge opposite Direction
	To_direction *int `json:"to_direction,omitempty"`
}

type customBackgroundImage struct {
	Name     string     `json:"name"`
	TopLeft  [2]float64 `json:"top_left"`
	TopRight [2]float64 `json:"top_right"`
}

type customMap struct {
	BackgroundImage *customBackgroundImage `json:"background_image,omitempty"`

	BoardSize uint16 `json:"board_size"`
	Players   int    `json:"players"`
	// degrees the default view turns the hex grid; the background image counter-rotates by its corners
	Rotation float64 `json:"rotation,omitempty"`
	// folder under the config dir whose files replace the default stat tables for this map
	ConfigDir    string      `json:"config_dir,omitempty"`
	StartingBank *customBank `json:"starting_bank,omitempty"`
	rulesParams
	Spaces      []customSpace      `json:"spaces"`
	Regions     []customRegion     `json:"regions,omitempty"`
	Connections []customConnection `json:"connections,omitempty"`
	PlayerStart *customPlayerStart `json:"player_start,omitempty"`
}

// Settings both map formats share; a nil or empty field leaves the game default
type mapRules struct {
	// researched for every player before turn 1
	StartingTechs       []uint32 `json:"starting_techs,omitempty"`
	UnlimitedPopulation bool     `json:"unlimited_population,omitempty"`
	// gold per turn for owning a space
	SpaceGoldIncome *float64 `json:"space_gold_income,omitempty"`
	// false lets players hire mercenaries into any space they own, not just regions they hold entirely
	MercenariesNeedRegion *bool `json:"mercenaries_need_region,omitempty"`
}

func parseCustomMap(data []byte) (customMap, error) {
	var m customMap
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		return m, err
	}
	if m.Players < 1 || m.Players > 12 {
		return m, fmt.Errorf("players must be 1 to 12, got %d", m.Players)
	}
	if m.PlayerStart == nil {
		for _, space := range m.Spaces {
			if space.PlayerSlot != nil {
				return m, fmt.Errorf("player slots require a player_start template")
			}
		}
	}
	return m, nil
}

func loadCustomMap(name string) (customMap, error) {
	data, err := defs.ReadConfigFile("maps", "custom", name+".json")
	if err != nil {
		return customMap{}, fmt.Errorf("custom map %q: %v", name, err)
	}
	m, err := parseCustomMap(data)
	if err != nil {
		return m, fmt.Errorf("custom map %q: %v", name, err)
	}
	return m, nil
}

// The config folder a "custom:<name>" map asks for, or "" for the default; any other map kind uses the default
func MapConfigDir(map_name string) (string, error) {
	kind, name, _ := strings.Cut(map_name, ":")
	if kind != "custom" {
		return "", nil
	}
	m, err := loadCustomMap(name)
	return m.ConfigDir, err
}

func CustomDocumentConfigDir(data []byte) (string, error) {
	m, err := parseCustomMap(data)
	return m.ConfigDir, err
}

func placeCustomZone(ctx *mapScriptContext, space Space, cz customZone) error {
	board := ctx.board
	zone := space.Zone(game_utils.Coordinate2D{X: cz.X, Y: cz.Y})
	if zone == nil {
		return fmt.Errorf("(%d,%d) has no zone (%d,%d)", space.Coordinate().X, space.Coordinate().Y, cz.X, cz.Y)
	}
	config := ctx.config()
	if _, known := config.Terrains[cz.TerrainOverride]; cz.TerrainOverride != 0 && !known {
		logMapProblem(fmt.Errorf("space (%d,%d) zone (%d,%d): unknown terrain override %d ignored", space.Coordinate().X, space.Coordinate().Y, cz.X, cz.Y, cz.TerrainOverride))
	} else if cz.TerrainOverride != 0 {
		zone.SetTerrainOverride(cz.TerrainOverride)
	}
	if cz.ResourceId != 0 {
		if _, ok := config.Resources[cz.ResourceId]; !ok || zone.Occupied() {
			return fmt.Errorf("resource %d is unknown or its zone is occupied", cz.ResourceId)
		}
		board.PlaceResource(zone, cz.ResourceId)
	}
	return placeCustomOccupants(ctx, zone, cz)
}

func placeCustomOccupants(ctx *mapScriptContext, zone Zone, cz customZone) error {
	board, config := ctx.board, ctx.config()
	if b := cz.Building; b != nil && b.Player >= 0 && b.Player < ctx.num_players {
		resources := []float64{}
		if b.ResourcesLeft != nil {
			building := config.Buildings[b.BuildingId]
			if !building.IsGatherable() || *b.ResourcesLeft < 0 || *b.ResourcesLeft > building.Gather.Starting_resources {
				return fmt.Errorf("invalid resources_left for building %d", b.BuildingId)
			}
			resources = append(resources, *b.ResourcesLeft)
		}
		if _, ok := config.Buildings[b.BuildingId]; !ok || zone.Occupied() || !board.PlaceBuilding(zone, b.BuildingId, b.Player, resources...) {
			return fmt.Errorf("could not place building %d", b.BuildingId)
		}
	}
	for _, u := range cz.Units {
		if _, ok := config.Units[u.UnitId]; !ok || u.Player < 0 {
			return fmt.Errorf("unit %d is unknown or has a negative player", u.UnitId)
		}
		count, err := u.Count.resolveInt(ctx.vars)
		if err != nil {
			return err
		}
		if count < 0 {
			logMapProblem(fmt.Errorf("unit %d has a negative count %d, placing none", u.UnitId, count))
			continue
		}
		for range count {
			if u.Player < ctx.num_players {
				board.PlaceUnit(zone, u.UnitId, u.Player)
			}
		}
	}
	return nil
}

func generateCustom(board Board, rng *rand.Rand, settings Settings, num_players int, name string) error {
	m, err := loadCustomMap(name)
	if err != nil {
		return err
	}
	if err := buildCustomMap(newCustomContext(board, rng, settings, num_players, m), m, true); err != nil {
		return fmt.Errorf("custom map %q: %v", name, err)
	}
	return nil
}

// For previews and tests: the default settings apply and start slots fill in order instead of by seeded draw
func GenerateCustomDocument(board Board, num_players int, data []byte) error {
	return generateCustomDocument(board, rand.New(rand.NewSource(0)), ResolveSettings(nil, num_players), num_players, data, false)
}

// Like a launched game: the settings fill the variables and the rng picks which start slots play
func GenerateCustomDocumentSeeded(board Board, rng *rand.Rand, settings Settings, num_players int, data []byte) error {
	return generateCustomDocument(board, rng, settings, num_players, data, true)
}

func generateCustomDocument(board Board, rng *rand.Rand, settings Settings, num_players int, data []byte, random_slots bool) error {
	if err := checkPlayerCount(num_players); err != nil {
		return err
	}
	m, err := parseCustomMap(data)
	if err != nil {
		return err
	}
	return buildCustomMap(newCustomContext(board, rng, settings, num_players, m), m, random_slots)
}

// The same variables a map script gets, so a custom map's expressions can use or ignore every setting
func newCustomContext(board Board, rng *rand.Rand, settings Settings, num_players int, m customMap) *mapScriptContext {
	return &mapScriptContext{
		board: board, rng: rng, num_players: num_players, board_size: m.BoardSize,
		vars: newMapScriptVars(num_players, settings, m.BoardSize, len(m.Spaces)),
	}
}

func buildCustomMap(ctx *mapScriptContext, m customMap, random_slots bool) error {
	board, num_players := ctx.board, ctx.num_players
	if num_players > m.Players {
		return fmt.Errorf("map has %d player slots, got %d players", m.Players, num_players)
	}
	if err := carveCustomSpaces(board, m); err != nil {
		return err
	}
	if err := fillCustomSpaces(ctx, m); err != nil {
		return err
	}
	if err := placeCustomPlayerStarts(ctx, m, random_slots); err != nil {
		return err
	}
	rules, err := m.rulesParams.resolve(ctx.vars)
	if err != nil {
		return err
	}
	if err := applyMapRules(board, num_players, rules); err != nil {
		return err
	}
	for _, space := range m.Spaces {
		if space.GoldIncome != nil {
			board.Space(game_utils.Coordinate2D{X: space.X, Y: space.Y}).SetGoldIncome(*space.GoldIncome)
		}
	}
	board.SetRotation(m.Rotation)
	if m.BackgroundImage != nil {
		board.SetBackgroundImage(m.BackgroundImage.Name, m.BackgroundImage.TopLeft, m.BackgroundImage.TopRight)
	}
	return connectCustomSpaces(board, m.Connections)
}

func applyMapRules(board Board, num_players int, rules mapRules) error {
	for player := 0; player < num_players; player++ {
		for _, tech_id := range rules.StartingTechs {
			if err := board.GrantStartingTech(player, tech_id); err != nil {
				return err
			}
		}
	}
	if rules.UnlimitedPopulation {
		board.SetUnlimitedPopulation()
	}
	if rules.SpaceGoldIncome != nil {
		board.SetSpaceGoldIncome(*rules.SpaceGoldIncome)
	}
	if rules.MercenariesNeedRegion != nil {
		board.SetMercenariesNeedRegion(*rules.MercenariesNeedRegion)
	}
	return nil
}

func fillCustomSpaces(ctx *mapScriptContext, m customMap) error {
	board := ctx.board
	for _, cs := range m.Spaces {
		space := board.Space(game_utils.Coordinate2D{X: cs.X, Y: cs.Y})
		if _, ok := ctx.config().Terrains[cs.Terrain]; cs.Terrain != 0 && !ok {
			return fmt.Errorf("space (%d,%d): unknown terrain %d", cs.X, cs.Y, cs.Terrain)
		} else if cs.Terrain != 0 {
			space.SetTerrain(cs.Terrain)
		}
		name := strings.TrimSpace(cs.Name)
		if len([]rune(name)) > maxCustomSpaceNameLength {
			return fmt.Errorf("space (%d,%d): name is longer than %d characters", cs.X, cs.Y, maxCustomSpaceNameLength)
		}
		space.SetName(name)
		for _, cz := range cs.Zones {
			if err := placeCustomZone(ctx, space, cz); err != nil {
				return fmt.Errorf("space (%d,%d): %v", cs.X, cs.Y, err)
			}
		}
	}
	return fillCustomRegionsAndBank(ctx, m)
}

func fillCustomRegionsAndBank(ctx *mapScriptContext, m customMap) error {
	board := ctx.board
	for _, region := range m.Regions {
		keys := make(map[uint]bool, len(region.Spaces))
		for _, c := range region.Spaces {
			space := board.Space(game_utils.Coordinate2D{X: c[0], Y: c[1]})
			if space == nil {
				return fmt.Errorf("region %q: space (%d,%d) is not on the map", region.Name, c[0], c[1])
			}
			keys[space.Key()] = true
		}
		if err := board.AddRegion(region.Name, region.GoldBonus, keys); err != nil {
			return err
		}
	}
	if m.StartingBank == nil {
		return nil
	}
	bank, err := m.StartingBank.resolve(ctx.vars)
	if err != nil {
		return err
	}
	for i := 0; i < ctx.num_players; i++ {
		board.SetStartingBank(i, bank)
	}
	return nil
}

func connectCustomSpaces(board Board, connections []customConnection) error {
	for _, connection := range connections {
		a := board.Space(game_utils.Coordinate2D{X: connection.From[0], Y: connection.From[1]})
		b := board.Space(game_utils.Coordinate2D{X: connection.To[0], Y: connection.To[1]})
		if a == nil || b == nil || a.Key() == b.Key() || a.Impassable() || b.Impassable() {
			return fmt.Errorf("invalid connection %v -> %v", connection.From, connection.To)
		}
		if connection.Direction == nil {
			return fmt.Errorf("connection %v -> %v needs a direction from 0 to 5", connection.From, connection.To)
		}
		to_direction := (*connection.Direction + 3) % 6
		if connection.To_direction != nil {
			to_direction = *connection.To_direction
		}
		if err := board.ConnectSpaces(a, b, *connection.Direction, to_direction); err != nil {
			return err
		}
	}
	return nil
}

func carveCustomSpaces(board Board, m customMap) error {
	board.Allocate(m.BoardSize)
	listed := make(map[uint]bool, len(m.Spaces))
	for _, cs := range m.Spaces {
		space := board.Space(game_utils.Coordinate2D{X: cs.X, Y: cs.Y})
		if space == nil || listed[space.Key()] {
			return fmt.Errorf("space (%d,%d) is off the board or listed twice", cs.X, cs.Y)
		}
		listed[space.Key()] = true
	}
	for _, space := range board.Spaces() {
		if !listed[space.Key()] {
			board.RemoveSpace(space)
		}
	}
	return nil
}
