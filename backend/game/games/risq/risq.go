package risq

import (
	"errors"
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/game_utils"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

/*
   ================
   >>>>> RISQ <<<<<
   ================

   Objective: Build your empire and conquer the world!
   Description: Strategy board game with simultaneous turn resolution, hexgonal
     map, complex deterministic mechanics (no randomness after map generation),
  	 resource gathering, empire-building, complex combat, and medieval themes.
*/

type GameRisq struct {
	game                      *game.GameBase
	players                   []*RisqPlayer
	board_size                uint16
	population_limit          uint16
	spaces                    [][]*RisqSpace
	next_resource_internal_id uint64
	next_building_internal_id uint64
	next_unit_internal_id     uint64
	next_order_internal_id    uint64
	turn_number               uint16
}

func CreateGame(g *game.GameBase) (*GameRisq, error) {
	risq := GameRisq{
		game:                      g,
		players:                   []*RisqPlayer{},
		population_limit:          100,
		next_resource_internal_id: 0,
		next_building_internal_id: 0,
		next_unit_internal_id:     0,
		next_order_internal_id:    0,
		turn_number:               0,
	}
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		color := ""
		switch player_id {
		case 0:
			color = "90, 90, 250"
		case 1:
			color = "250, 90, 90"
		case 2:
			color = "90, 250, 90"
		case 3:
			color = "190, 190, 50"
		case 4:
			color = "190, 50, 190"
		case 5:
			color = "50, 190, 190"
		default:
			fmt.Fprintln(os.Stderr, "Unknown player id for color", player_id)
		}
		risq.players = append(risq.players, createRisqPlayer(player, risq.population_limit, color))
		player_id++
	}
	if len(risq.players) < 2 {
		//return nil, errors.New("Need at least two players to play risq")
	} else if len(risq.players) > 6 {
		return nil, errors.New("can have max of six players playing risq")
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
	for range 3 {
		villager := createRisqUnit(r.nextUnitInternalId(), 1, p.player.Player_id)
		s.setUnit(&game_utils.Coordinate2D{X: 0, Y: 0}, villager)
		p.units[villager.internal_id] = villager
	}
	infantry := createRisqUnit(r.nextUnitInternalId(), 11, p.player.Player_id)
	s.setUnit(&game_utils.Coordinate2D{X: 0, Y: 0}, infantry)
	p.units[infantry.internal_id] = infantry
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

func (r *GameRisq) nextResourceInternalId() uint64 {
	r.next_resource_internal_id++
	return r.next_resource_internal_id
}

func (r *GameRisq) nextBuildingInternalId() uint64 {
	r.next_building_internal_id++
	return r.next_building_internal_id
}

func (r *GameRisq) nextUnitInternalId() uint64 {
	r.next_unit_internal_id++
	return r.next_unit_internal_id
}

func (r *GameRisq) nextOrderInternalId() uint64 {
	r.next_order_internal_id++
	return r.next_order_internal_id
}

func (r *GameRisq) coordinateToIndex(c *game_utils.Coordinate2D) *game_utils.Coordinate2D {
	return &game_utils.Coordinate2D{
		X: c.Y + int(r.board_size),
		Y: c.X - max(-int(r.board_size), -(int(r.board_size)+c.Y)),
	}
}

func (r *GameRisq) getSpace(c *game_utils.Coordinate2D) *RisqSpace {
	index := r.coordinateToIndex(c)
	if index.X < 0 || index.X >= len(r.spaces) {
		return nil
	}
	row := r.spaces[index.X]
	if index.Y < 0 || index.Y >= len(row) {
		return nil
	}
	return row[index.Y]
}

func (r *GameRisq) GetBase() *game.GameBase {
	return r.game
}

func (r *GameRisq) StartGame() {
	r.startNextTurn()
}

func (r *GameRisq) startNextTurn() {
	r.turn_number++
	for _, player := range r.players {
		player.player.AddUpdate(&game.UpdateMessage{Kind: "start-turn", Content: gin.H{
			"game": r.ToFrontend(player.player.GetClientId(), false),
		}})
	}
	r.game.AddViewerUpdate(&game.UpdateMessage{Kind: "start-turn", Content: gin.H{
		"game": r.ToFrontend(0, true),
	}})
}

func (r *GameRisq) Valid() bool {
	if r.game == nil {
		return false
	}
	for _, player := range r.players {
		if !player.valid() {
			return false
		}
	}
	return true
}

func (r *GameRisq) PlayerAction(action game.PlayerAction) {
	fmt.Println("player action:", action.Kind, action.Client_id, action.Action)
	player := r.game.Players[uint64(action.Client_id)]
	if player == nil {
		fmt.Fprintln(os.Stderr, "Invalid client id", action.Client_id)
		return
	}
	switch action.Kind {
	default:
		fmt.Fprintln(os.Stderr, "Unknown game update type", action.Kind)
	}
}

func (r *GameRisq) PlayerDisconnected(client_id uint64) {
}

func (r *GameRisq) PlayerReconnected(client_id uint64) {
}

func (r *GameRisq) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	game := gin.H{
		"board_size":       r.board_size,
		"population_limit": r.population_limit,
		"turn_number":      r.turn_number,
	}
	if r.game != nil {
		game["game_base"] = r.game.ToFrontend(client_id, is_viewer)
	}
	player_id := -1
	players := []gin.H{}
	for id, player := range r.players {
		if player != nil {
			if !is_viewer && client_id == player.player.GetClientId() {
				player_id = id
			}
			players = append(players, player.toFrontend(is_viewer || client_id == player.player.GetClientId()))
		}
	}
	game["players"] = players
	spaces := [][]gin.H{}
	for _, row := range r.spaces {
		spaces_row := []gin.H{}
		for _, space := range row {
			spaces_row = append(spaces_row, space.toFrontend(player_id, is_viewer))
		}
		spaces = append(spaces, spaces_row)
	}
	game["spaces"] = spaces
	return game
}
