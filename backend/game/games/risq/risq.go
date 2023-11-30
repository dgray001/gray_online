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
	spaces                    [][]*RisqSpace
	next_building_internal_id uint64
	next_unit_internal_id     uint64
}

func CreateGame(g *game.GameBase) (*GameRisq, error) {
	risq := GameRisq{
		game:                      g,
		players:                   []*RisqPlayer{},
		next_building_internal_id: 0,
		next_unit_internal_id:     0,
	}
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		risq.players = append(risq.players, createRisqPlayer(player))
		player_id++
	}
	if len(risq.players) < 2 {
		//return nil, errors.New("Need at least two players to play risq")
	} else if len(risq.players) > 6 {
		return nil, errors.New("Can have of six players playing risq")
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
		starting_distance = util.RandomInt(2, 3)
	}
	risq.spaces = make([][]*RisqSpace, 2*int(risq.board_size)+1)
	for j := range risq.spaces {
		r := j - int(risq.board_size)
		l := 2*int(risq.board_size) + 1 - util.AbsInt(r)
		risq.spaces[j] = make([]*RisqSpace, l)
		for i := range risq.spaces[j] {
			q := max(-int(risq.board_size), -(int(risq.board_size)+r)) + i
			risq.spaces[j][i] = createRisqSpace(q, r)
		}
	}
	starting_location := util.RandomInt(0, 5)
	axial_unit_vectors := game_utils.AxialDirectionVectors()
	starting_space0 := risq.getSpace(axial_unit_vectors[starting_location].Multiply(starting_distance))
	if starting_space0 == nil {
		return nil, errors.New("Starting space 0 is nil")
	}
	risq.createPlayerStart(risq.players[0], starting_space0)
	switch len(risq.players) {
	case 1:
	case 2:
		starting_space1 := risq.getSpace(axial_unit_vectors[(starting_location+3)%6].Multiply(starting_distance))
		if starting_space1 == nil {
			return nil, errors.New("Starting space 1 is nil")
		}
		risq.createPlayerStart(risq.players[1], starting_space1)
	case 3:
		starting_space1 := risq.getSpace(axial_unit_vectors[(starting_location+2)%6].Multiply(starting_distance))
		if starting_space1 == nil {
			return nil, errors.New("Starting space 1 is nil")
		}
		risq.createPlayerStart(risq.players[1], starting_space1)
		starting_space2 := risq.getSpace(axial_unit_vectors[(starting_location+4)%6].Multiply(starting_distance))
		if starting_space2 == nil {
			return nil, errors.New("Starting space 2 is nil")
		}
		risq.createPlayerStart(risq.players[2], starting_space2)
	case 4:
		starting_space1 := risq.getSpace(axial_unit_vectors[(starting_location+1)%6].Multiply(starting_distance))
		if starting_space1 == nil {
			return nil, errors.New("Starting space 1 is nil")
		}
		risq.createPlayerStart(risq.players[1], starting_space1)
		starting_space2 := risq.getSpace(axial_unit_vectors[(starting_location+3)%6].Multiply(starting_distance))
		if starting_space2 == nil {
			return nil, errors.New("Starting space 2 is nil")
		}
		risq.createPlayerStart(risq.players[2], starting_space2)
		starting_space3 := risq.getSpace(axial_unit_vectors[(starting_location+4)%6].Multiply(starting_distance))
		if starting_space3 == nil {
			return nil, errors.New("Starting space 3 is nil")
		}
		risq.createPlayerStart(risq.players[3], starting_space3)
	case 5:
		starting_space1 := risq.getSpace(axial_unit_vectors[(starting_location+1)%6].Multiply(starting_distance))
		if starting_space1 == nil {
			return nil, errors.New("Starting space 1 is nil")
		}
		risq.createPlayerStart(risq.players[1], starting_space1)
		starting_space2 := risq.getSpace(axial_unit_vectors[(starting_location+2)%6].Multiply(starting_distance))
		if starting_space2 == nil {
			return nil, errors.New("Starting space 2 is nil")
		}
		risq.createPlayerStart(risq.players[2], starting_space2)
		starting_space3 := risq.getSpace(axial_unit_vectors[(starting_location+3)%6].Multiply(starting_distance))
		if starting_space3 == nil {
			return nil, errors.New("Starting space 3 is nil")
		}
		risq.createPlayerStart(risq.players[3], starting_space3)
		starting_space4 := risq.getSpace(axial_unit_vectors[(starting_location+4)%6].Multiply(starting_distance))
		if starting_space4 == nil {
			return nil, errors.New("Starting space 4 is nil")
		}
		risq.createPlayerStart(risq.players[4], starting_space4)
	case 6:
		starting_space1 := risq.getSpace(axial_unit_vectors[(starting_location+1)%6].Multiply(starting_distance))
		if starting_space1 == nil {
			return nil, errors.New("Starting space 1 is nil")
		}
		risq.createPlayerStart(risq.players[1], starting_space1)
		starting_space2 := risq.getSpace(axial_unit_vectors[(starting_location+2)%6].Multiply(starting_distance))
		if starting_space2 == nil {
			return nil, errors.New("Starting space 2 is nil")
		}
		risq.createPlayerStart(risq.players[2], starting_space2)
		starting_space3 := risq.getSpace(axial_unit_vectors[(starting_location+3)%6].Multiply(starting_distance))
		if starting_space3 == nil {
			return nil, errors.New("Starting space 3 is nil")
		}
		risq.createPlayerStart(risq.players[3], starting_space3)
		starting_space4 := risq.getSpace(axial_unit_vectors[(starting_location+4)%6].Multiply(starting_distance))
		if starting_space4 == nil {
			return nil, errors.New("Starting space 4 is nil")
		}
		risq.createPlayerStart(risq.players[4], starting_space4)
		starting_space5 := risq.getSpace(axial_unit_vectors[(starting_location+5)%6].Multiply(starting_distance))
		if starting_space5 == nil {
			return nil, errors.New("Starting space 5 is nil")
		}
		risq.createPlayerStart(risq.players[5], starting_space4)
	default:
		return nil, errors.New("Unknown number of players")
	}
	return &risq, nil
}

func (r *GameRisq) createPlayerStart(p *RisqPlayer, s *RisqSpace) {
	village_center := createRisqBuilding(r.nextBuildingInternalId(), 1, p.player.Player_id)
	p.buildings[village_center.internal_id] = village_center
	s.setBuilding(&game_utils.Coordinate2D{X: 0, Y: 0}, village_center)
	for i := 0; i < 3; i++ {
		villager := createRisqUnit(r.nextUnitInternalId(), 1, p.player.Player_id)
		p.units[villager.internal_id] = villager
		s.setUnit(&game_utils.Coordinate2D{X: 0, Y: 0}, villager)
	}
}

func (r *GameRisq) nextBuildingInternalId() uint64 {
	r.next_building_internal_id++
	return r.next_building_internal_id
}

func (r *GameRisq) nextUnitInternalId() uint64 {
	r.next_unit_internal_id++
	return r.next_unit_internal_id
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
}

func (r *GameRisq) Valid() bool {
	if r.game == nil {
		return false
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
		"board_size": r.board_size,
	}
	if r.game != nil {
		game["game_base"] = r.game.ToFrontend(client_id, is_viewer)
	}
	player_id := -1
	players := []gin.H{}
	for id, player := range r.players {
		if player != nil {
			if client_id == player.player.GetClientId() {
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
