package risq

import (
	"errors"
	"strconv"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
)

// palette a player's color defaults to when the lobby doesn't request one; order = default assignment order
// first 8 match AoE's classic player order; last 4 are extra slots picked for contrast, no AoE precedent
var risqPlayerColors = []string{
	"90, 90, 250",   // blue
	"250, 60, 60",   // red
	"60, 220, 60",   // green
	"250, 230, 50",  // yellow
	"60, 230, 230",  // cyan
	"160, 70, 230",  // purple
	"250, 150, 30",  // orange
	"250, 120, 190", // pink
	"170, 170, 170", // gray
	"150, 90, 40",   // brown
	"50, 60, 160",   // navy
	"170, 250, 80",  // lime
}

// keyed by client_id string since that's how game_specific_settings arrives off the wire (raw JSON)
func requestedRisqPlayerColors(g *game.GameBase) map[uint64]string {
	requested := make(map[uint64]string)
	raw, ok := g.GameSpecificSettings["player_colors"].(map[string]interface{})
	if !ok {
		return requested
	}
	for client_id_str, color_raw := range raw {
		color, color_ok := color_raw.(string)
		if !color_ok {
			continue
		}
		client_id, err := strconv.ParseUint(client_id_str, 10, 64)
		if err != nil {
			continue
		}
		requested[client_id] = color
	}
	return requested
}

func nextAvailableRisqColor(used map[string]bool) string {
	for _, color := range risqPlayerColors {
		if !used[color] {
			return color
		}
	}
	return ""
}

func CreateGame(g *game.GameBase) (*GameRisq, error) {
	risq := GameRisq{
		game:                      g,
		players:                   []*RisqPlayer{},
		units:                     make(map[uint64]*RisqUnit),
		buildings:                 make(map[uint64]*RisqBuilding),
		population_limit:          100,
		next_resource_internal_id: 0,
		next_building_internal_id: 0,
		next_unit_internal_id:     0,
		next_order_internal_id:    0,
		turn_number:               0,
	}
	requested_colors := requestedRisqPlayerColors(g)
	used_colors := make(map[string]bool)
	for _, color := range requested_colors {
		used_colors[color] = true
	}
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		color, requested := requested_colors[player.GetClientId()]
		if !requested {
			color = nextAvailableRisqColor(used_colors)
			used_colors[color] = true
		}
		risq.players = append(risq.players, createRisqPlayer(player, risq.population_limit, color))
		player_id++
	}
	if len(risq.players) < 2 {
		//return nil, errors.New("Need at least two players to play risq")
	} else if len(risq.players) > 12 {
		return nil, errors.New("can have max of twelve players playing risq")
	}
	starting_distance := 0
	switch len(risq.players) {
	case 6:
		risq.board_size = 6
		starting_distance = util.RandomInt(4, 5)
	case 5:
		risq.board_size = 6
		starting_distance = util.RandomInt(4, 5)
	case 4:
		risq.board_size = 5
		starting_distance = util.RandomInt(3, 4)
	case 3:
		risq.board_size = 4
		starting_distance = util.RandomInt(3, 3)
	default:
		risq.board_size = 4
		starting_distance = util.RandomInt(4, 4)
	}
	risq.spaces = make([][]*RisqSpace, 2*int(risq.board_size)+1)
	for j := range risq.spaces {
		r := j - int(risq.board_size)
		l := 2*int(risq.board_size) + 1 - util.AbsInt(r)
		risq.spaces[j] = make([]*RisqSpace, l)
		for i := range risq.spaces[j] {
			q := max(-int(risq.board_size), -(int(risq.board_size)+r)) + i
			risq.spaces[j][i] = createRisqSpace(q, r, TerrainType(TerrainType_FLATLANDS))
		}
	}
	for _, row := range risq.spaces {
		for _, space := range row {
			for _, v := range game_utils.AxialDirectionVectors() {
				adjacent := risq.getSpace(space.coordinate.Add(&v))
				if adjacent != nil {
					space.setAdjacentSpace(adjacent, &v)
				}
			}
		}
	}
	starting_location := util.RandomInt(0, 5)
	axial_unit_vectors := game_utils.AxialDirectionVectors()
	starting_space0 := risq.getSpace(axial_unit_vectors[starting_location].Multiply(starting_distance))
	if starting_space0 == nil {
		return nil, errors.New("starting space 0 is nil")
	}
	risq.createPlayerStart(risq.players[0], starting_space0)
	switch len(risq.players) {
	case 1:
	case 2:
		starting_space1 := risq.getSpace(axial_unit_vectors[(starting_location+3)%6].Multiply(starting_distance))
		if starting_space1 == nil {
			return nil, errors.New("starting space 1 is nil")
		}
		risq.createPlayerStart(risq.players[1], starting_space1)
	case 3:
		starting_space1 := risq.getSpace(axial_unit_vectors[(starting_location+2)%6].Multiply(starting_distance))
		if starting_space1 == nil {
			return nil, errors.New("starting space 1 is nil")
		}
		risq.createPlayerStart(risq.players[1], starting_space1)
		starting_space2 := risq.getSpace(axial_unit_vectors[(starting_location+4)%6].Multiply(starting_distance))
		if starting_space2 == nil {
			return nil, errors.New("starting space 2 is nil")
		}
		risq.createPlayerStart(risq.players[2], starting_space2)
	case 4:
		starting_space1 := risq.getSpace(axial_unit_vectors[(starting_location+1)%6].Multiply(starting_distance))
		if starting_space1 == nil {
			return nil, errors.New("starting space 1 is nil")
		}
		risq.createPlayerStart(risq.players[1], starting_space1)
		starting_space2 := risq.getSpace(axial_unit_vectors[(starting_location+3)%6].Multiply(starting_distance))
		if starting_space2 == nil {
			return nil, errors.New("starting space 2 is nil")
		}
		risq.createPlayerStart(risq.players[2], starting_space2)
		starting_space3 := risq.getSpace(axial_unit_vectors[(starting_location+4)%6].Multiply(starting_distance))
		if starting_space3 == nil {
			return nil, errors.New("starting space 3 is nil")
		}
		risq.createPlayerStart(risq.players[3], starting_space3)
	case 5:
		starting_space1 := risq.getSpace(axial_unit_vectors[(starting_location+1)%6].Multiply(starting_distance))
		if starting_space1 == nil {
			return nil, errors.New("starting space 1 is nil")
		}
		risq.createPlayerStart(risq.players[1], starting_space1)
		starting_space2 := risq.getSpace(axial_unit_vectors[(starting_location+2)%6].Multiply(starting_distance))
		if starting_space2 == nil {
			return nil, errors.New("starting space 2 is nil")
		}
		risq.createPlayerStart(risq.players[2], starting_space2)
		starting_space3 := risq.getSpace(axial_unit_vectors[(starting_location+3)%6].Multiply(starting_distance))
		if starting_space3 == nil {
			return nil, errors.New("starting space 3 is nil")
		}
		risq.createPlayerStart(risq.players[3], starting_space3)
		starting_space4 := risq.getSpace(axial_unit_vectors[(starting_location+4)%6].Multiply(starting_distance))
		if starting_space4 == nil {
			return nil, errors.New("starting space 4 is nil")
		}
		risq.createPlayerStart(risq.players[4], starting_space4)
	case 6:
		starting_space1 := risq.getSpace(axial_unit_vectors[(starting_location+1)%6].Multiply(starting_distance))
		if starting_space1 == nil {
			return nil, errors.New("starting space 1 is nil")
		}
		risq.createPlayerStart(risq.players[1], starting_space1)
		starting_space2 := risq.getSpace(axial_unit_vectors[(starting_location+2)%6].Multiply(starting_distance))
		if starting_space2 == nil {
			return nil, errors.New("starting space 2 is nil")
		}
		risq.createPlayerStart(risq.players[2], starting_space2)
		starting_space3 := risq.getSpace(axial_unit_vectors[(starting_location+3)%6].Multiply(starting_distance))
		if starting_space3 == nil {
			return nil, errors.New("starting space 3 is nil")
		}
		risq.createPlayerStart(risq.players[3], starting_space3)
		starting_space4 := risq.getSpace(axial_unit_vectors[(starting_location+4)%6].Multiply(starting_distance))
		if starting_space4 == nil {
			return nil, errors.New("starting space 4 is nil")
		}
		risq.createPlayerStart(risq.players[4], starting_space4)
		starting_space5 := risq.getSpace(axial_unit_vectors[(starting_location+5)%6].Multiply(starting_distance))
		if starting_space5 == nil {
			return nil, errors.New("starting space 5 is nil")
		}
		risq.createPlayerStart(risq.players[5], starting_space4)
	default:
		return nil, errors.New("unknown number of players")
	}
	return &risq, nil
}

func (r *GameRisq) createPlayerStart(p *RisqPlayer, s *RisqSpace) {
	village_center := createRisqBuilding(r.nextBuildingInternalId(), 1, p.player.Player_id)
	s.setBuilding(&game_utils.Coordinate2D{X: 0, Y: 0}, village_center)
	p.buildings[village_center.internal_id] = village_center
	r.buildings[village_center.internal_id] = village_center
	for range 3 {
		villager := createRisqUnit(r.nextUnitInternalId(), 1, p)
		s.setUnit(&game_utils.Coordinate2D{X: 0, Y: 0}, villager)
		p.units[villager.internal_id] = villager
		r.units[villager.internal_id] = villager
	}
	infantry := createRisqUnit(r.nextUnitInternalId(), 11, p)
	s.setUnit(&game_utils.Coordinate2D{X: 0, Y: 0}, infantry)
	p.units[infantry.internal_id] = infantry
	r.units[infantry.internal_id] = infantry
	zones := s.getZonesAsRandomArray(false)
	forage := createRisqResource(r.nextResourceInternalId(), 1)
	s.setResource(&zones[0].coordinate, forage)
	deer := createRisqResource(r.nextResourceInternalId(), 2)
	s.setResource(&zones[1].coordinate, deer)
	tree1 := createRisqResource(r.nextResourceInternalId(), 11)
	s.setResource(&zones[2].coordinate, tree1)
	tree2 := createRisqResource(r.nextResourceInternalId(), 14)
	s.setResource(&zones[3].coordinate, tree2)
	stone := createRisqResource(r.nextResourceInternalId(), 21)
	s.setResource(&zones[4].coordinate, stone)
}
