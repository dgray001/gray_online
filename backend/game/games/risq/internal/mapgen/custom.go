package mapgen

import (
	"bytes"
	"encoding/json"
	"fmt"

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
	Count  int    `json:"count"`
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
	Zones      []customZone `json:"zones,omitempty"`
	PlayerSlot *int         `json:"player_slot,omitempty"`
}

type customRegion struct {
	Name      string   `json:"name"`
	GoldBonus float64  `json:"gold_bonus,omitempty"`
	Spaces    [][2]int `json:"spaces"`
}

type customConnection struct {
	From [2]int `json:"from"`
	To   [2]int `json:"to"`
	// 0 to 5 joins From's edge in that direction to To's opposite edge
	Direction int `json:"direction"`
}

type customBackgroundImage struct {
	Name     string `json:"name"`
	TopLeft  [2]int `json:"top_left"`
	TopRight [2]int `json:"top_right"`
}

type customMap struct {
	BackgroundImage *customBackgroundImage `json:"background_image,omitempty"`

	BoardSize    uint16        `json:"board_size"`
	Players      int           `json:"players"`
	StartingBank *StartingBank `json:"starting_bank,omitempty"`
	mapRules
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

func placeCustomZone(board Board, space Space, cz customZone, num_players int) error {
	zone := space.Zone(game_utils.Coordinate2D{X: cz.X, Y: cz.Y})
	if zone == nil {
		return fmt.Errorf("(%d,%d) has no zone (%d,%d)", space.Coordinate().X, space.Coordinate().Y, cz.X, cz.Y)
	}
	if _, known := defs.TerrainConfigs[cz.TerrainOverride]; cz.TerrainOverride != 0 && !known {
		logMapProblem(fmt.Errorf("space (%d,%d) zone (%d,%d): unknown terrain override %d ignored", space.Coordinate().X, space.Coordinate().Y, cz.X, cz.Y, cz.TerrainOverride))
	} else if cz.TerrainOverride != 0 {
		zone.SetTerrainOverride(cz.TerrainOverride)
	}
	if cz.ResourceId != 0 {
		if _, ok := defs.ResourceConfigs[cz.ResourceId]; !ok || zone.Occupied() {
			return fmt.Errorf("resource %d is unknown or its zone is occupied", cz.ResourceId)
		}
		board.PlaceResource(zone, cz.ResourceId)
	}
	return placeCustomOccupants(board, zone, cz, num_players)
}

func placeCustomOccupants(board Board, zone Zone, cz customZone, num_players int) error {
	if b := cz.Building; b != nil && b.Player >= 0 && b.Player < num_players {
		resources := []float64{}
		if b.ResourcesLeft != nil {
			config := defs.BuildingConfigs[b.BuildingId]
			if !config.IsGatherable() || *b.ResourcesLeft < 0 || *b.ResourcesLeft > config.Gather.Starting_resources {
				return fmt.Errorf("invalid resources_left for building %d", b.BuildingId)
			}
			resources = append(resources, *b.ResourcesLeft)
		}
		if _, ok := defs.BuildingConfigs[b.BuildingId]; !ok || zone.Occupied() || !board.PlaceBuilding(zone, b.BuildingId, b.Player, resources...) {
			return fmt.Errorf("could not place building %d", b.BuildingId)
		}
	}
	for _, u := range cz.Units {
		if _, ok := defs.UnitConfigs[u.UnitId]; !ok || u.Player < 0 {
			return fmt.Errorf("unit %d is unknown or has a negative player", u.UnitId)
		}
		if u.Count < 0 {
			logMapProblem(fmt.Errorf("unit %d has a negative count %d, placing none", u.UnitId, u.Count))
			continue
		}
		for range u.Count {
			if u.Player < num_players {
				board.PlaceUnit(zone, u.UnitId, u.Player)
			}
		}
	}
	return nil
}

func generateCustom(board Board, num_players int, name string) error {
	m, err := loadCustomMap(name)
	if err != nil {
		return err
	}
	if err := buildCustomMap(board, num_players, m); err != nil {
		return fmt.Errorf("custom map %q: %v", name, err)
	}
	return nil
}

func GenerateCustomDocument(board Board, num_players int, data []byte) error {
	if err := checkPlayerCount(num_players); err != nil {
		return err
	}
	m, err := parseCustomMap(data)
	if err != nil {
		return err
	}
	return buildCustomMap(board, num_players, m)
}

func buildCustomMap(board Board, num_players int, m customMap) error {
	if num_players > m.Players {
		return fmt.Errorf("map has %d player slots, got %d players", m.Players, num_players)
	}
	if err := carveCustomSpaces(board, m); err != nil {
		return err
	}
	if err := fillCustomSpaces(board, m, num_players); err != nil {
		return err
	}
	if err := placeCustomPlayerStarts(board, m, num_players); err != nil {
		return err
	}
	if err := applyMapRules(board, num_players, m.mapRules); err != nil {
		return err
	}
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

func fillCustomSpaces(board Board, m customMap, num_players int) error {
	for _, cs := range m.Spaces {
		space := board.Space(game_utils.Coordinate2D{X: cs.X, Y: cs.Y})
		if _, ok := defs.TerrainConfigs[cs.Terrain]; cs.Terrain != 0 && !ok {
			return fmt.Errorf("space (%d,%d): unknown terrain %d", cs.X, cs.Y, cs.Terrain)
		} else if cs.Terrain != 0 {
			space.SetTerrain(cs.Terrain)
		}
		for _, cz := range cs.Zones {
			if err := placeCustomZone(board, space, cz, num_players); err != nil {
				return fmt.Errorf("space (%d,%d): %v", cs.X, cs.Y, err)
			}
		}
	}
	return fillCustomRegionsAndBank(board, m, num_players)
}

func fillCustomRegionsAndBank(board Board, m customMap, num_players int) error {
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
	for i := 0; m.StartingBank != nil && i < num_players; i++ {
		board.SetStartingBank(i, *m.StartingBank)
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
		if err := board.ConnectSpaces(a, b, connection.Direction); err != nil {
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
