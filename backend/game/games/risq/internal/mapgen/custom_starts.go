package mapgen

import (
	"fmt"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
)

type customPlayerStart struct {
	Size    int           `json:"size"`
	Terrain uint32        `json:"terrain,omitempty"`
	Spaces  []customSpace `json:"spaces"`
}

func validateCustomStartTemplate(start customPlayerStart, boardSize uint16) error {
	if start.Size < 0 || start.Size > int(boardSize) {
		return fmt.Errorf("invalid player start size %d", start.Size)
	}
	used := make(map[game_utils.Coordinate2D]bool)
	for _, space := range start.Spaces {
		coordinate := game_utils.Coordinate2D{X: space.X, Y: space.Y}
		if space.PlayerSlot != nil || used[coordinate] || int(game_utils.AxialDistance(game_utils.Coordinate2D{}, coordinate)) > start.Size {
			return fmt.Errorf("invalid player start template space %v", coordinate)
		}
		used[coordinate] = true
	}
	return nil
}

func customStartCoordinate(home game_utils.Coordinate2D, x, y int) game_utils.Coordinate2D {
	return game_utils.Coordinate2D{X: home.X + x, Y: home.Y + y}
}

func setCustomStartTerrain(space Space, terrain uint32) error {
	if terrain == 0 {
		terrain = defs.DefaultTerrainId
	}
	config, ok := defs.TerrainConfigs[terrain]
	if !ok || config.Terrain_type.MoveCost().Impassable {
		return fmt.Errorf("invalid player start terrain %d", terrain)
	}
	space.SetTerrain(terrain)
	return nil
}

func customStartZone(zone customZone, player int) customZone {
	if zone.Building != nil {
		building := *zone.Building
		building.Player = player
		zone.Building = &building
	}
	zone.Units = append([]customUnit(nil), zone.Units...)
	for i := range zone.Units {
		zone.Units[i].Player = player
	}
	return zone
}

func customStartSlots(board Board, m customMap) ([]Space, error) {
	starts := make([]Space, m.Players)
	for _, space := range m.Spaces {
		if space.PlayerSlot == nil {
			continue
		}
		slot := *space.PlayerSlot
		if slot < 0 || slot >= len(starts) || starts[slot] != nil {
			return nil, fmt.Errorf("invalid or duplicate player slot %d", slot)
		}
		starts[slot] = board.Space(game_utils.Coordinate2D{X: space.X, Y: space.Y})
	}
	return starts, nil
}

func placeCustomStartContents(board Board, home Space, start customPlayerStart, player, numPlayers int) error {
	for _, template := range start.Spaces {
		space := board.Space(customStartCoordinate(home.Coordinate(), template.X, template.Y))
		if template.Terrain != 0 {
			if err := setCustomStartTerrain(space, template.Terrain); err != nil {
				return err
			}
		}
		for _, zone := range template.Zones {
			if err := placeCustomZone(board, space, customStartZone(zone, player), numPlayers); err != nil {
				return err
			}
		}
	}
	return nil
}

func customStartFootprint(board Board, home Space, size int) ([]Space, error) {
	spaces := make([]Space, 0)
	for x := -size; x <= size; x++ {
		for y := max(-size, -x-size); y <= min(size, -x+size); y++ {
			space := board.Space(customStartCoordinate(home.Coordinate(), x, y))
			if space == nil || space.Impassable() {
				return nil, fmt.Errorf("player start at %v has a clipped or impassable footprint", home.Coordinate())
			}
			spaces = append(spaces, space)
		}
	}
	return spaces, nil
}

func validateCustomStartSlots(board Board, starts []Space, size int) error {
	for i, home := range starts {
		if home == nil {
			return fmt.Errorf("missing player slot %d", i)
		}
		if _, err := customStartFootprint(board, home, size); err != nil {
			return err
		}
		for _, other := range starts[:i] {
			if board.Distance(home, other) <= 2*size {
				return fmt.Errorf("player slot %d overlaps another start area", i)
			}
		}
	}
	return nil
}

func prepareCustomStartSlots(board Board, m customMap) ([]Space, error) {
	if err := validateCustomStartTemplate(*m.PlayerStart, board.BoardSize()); err != nil {
		return nil, err
	}
	starts, err := customStartSlots(board, m)
	if err != nil {
		return nil, err
	}
	if err := validateCustomStartSlots(board, starts, m.PlayerStart.Size); err != nil {
		return nil, err
	}
	return starts, nil
}

func prepareCustomStartTerrain(board Board, home Space, start customPlayerStart) error {
	footprint, err := customStartFootprint(board, home, start.Size)
	if err != nil {
		return err
	}
	for _, space := range footprint {
		if err := setCustomStartTerrain(space, start.Terrain); err != nil {
			return err
		}
		space.ClearOccupants()
	}
	return nil
}

func placeCustomPlayerStarts(board Board, m customMap, numPlayers int) error {
	if m.PlayerStart == nil {
		return nil
	}
	starts, err := prepareCustomStartSlots(board, m)
	if err != nil {
		return err
	}
	for player, home := range starts[:numPlayers] {
		if err := prepareCustomStartTerrain(board, home, *m.PlayerStart); err != nil {
			return err
		}
		if err := placeCustomStartContents(board, home, *m.PlayerStart, player, numPlayers); err != nil {
			return err
		}
	}
	return nil
}
