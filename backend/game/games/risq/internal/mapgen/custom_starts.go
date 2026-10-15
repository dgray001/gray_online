package mapgen

import (
	"fmt"
	"slices"

	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
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

func setCustomStartTerrain(board Board, space Space, terrain uint32) error {
	if terrain == 0 {
		terrain = defs.DefaultTerrainId
	}
	config := configOf(board)
	terrain_config, ok := config.Terrains[terrain]
	if !ok || config.MoveCost(terrain_config.Terrain_type).Impassable {
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

func placeCustomStartContents(ctx *mapScriptContext, home Space, start customPlayerStart, player int) error {
	for _, template := range start.Spaces {
		space := ctx.board.Space(customStartCoordinate(home.Coordinate(), template.X, template.Y))
		if template.Terrain != 0 {
			if err := setCustomStartTerrain(ctx.board, space, template.Terrain); err != nil {
				return err
			}
		}
		for _, zone := range template.Zones {
			if err := placeCustomZone(ctx, space, customStartZone(zone, player)); err != nil {
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
		if err := setCustomStartTerrain(board, space, start.Terrain); err != nil {
			return err
		}
		space.ClearOccupants()
	}
	return nil
}

func placeCustomPlayerStarts(ctx *mapScriptContext, m customMap, random_slots bool) error {
	if m.PlayerStart == nil {
		return nil
	}
	slots, err := prepareCustomStartSlots(ctx.board, m)
	if err != nil {
		return err
	}
	homes := slices.Clone(slots)
	if random_slots {
		util.ShuffleFrom(ctx.rng, homes)
	}
	homes = homes[:ctx.num_players]
	for player, home := range homes {
		if err := prepareCustomStartTerrain(ctx.board, home, *m.PlayerStart); err != nil {
			return err
		}
		if err := placeCustomStartContents(ctx, home, *m.PlayerStart, player); err != nil {
			return err
		}
	}
	if m.PlayerStart.specifiesAssets() {
		return nil
	}
	return placeDevelopmentStarts(ctx, homes, m.PlayerStart.Size)
}

// Whether the template names any unit or building, which then replaces the development setting's defaults
func (s customPlayerStart) specifiesAssets() bool {
	for _, space := range s.Spaces {
		for _, zone := range space.Zones {
			if zone.Building != nil || len(zone.Units) > 0 {
				return true
			}
		}
	}
	return false
}

// The development setting's default units and buildings at each home, placed the way a map script places them
func placeDevelopmentStarts(ctx *mapScriptContext, homes []Space, area_size int) error {
	var p playerStartsParams
	applyStartingDevelopmentDefaults(ctx, &p)
	config := ctx.config()
	p.Units = slices.DeleteFunc(p.Units, func(u playerStartUnitJSON) bool { _, exists := config.Units[u.UnitId]; return !exists })
	p.Buildings = slices.DeleteFunc(p.Buildings, func(b playerStartBuildingJSON) bool {
		_, exists := config.Buildings[b.BuildingId]
		return !exists
	})
	starts := make([]playerStartInfo, len(homes))
	for i, home := range homes {
		starts[i] = playerStartInfo{space: home}
	}
	ctx.player_starts, ctx.player_area_size = starts, area_size
	return placeStartContents(ctx, p, starts, startFootprints(starts, area_size), 0)
}
