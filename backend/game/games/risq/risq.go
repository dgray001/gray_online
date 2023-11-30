package risq

import (
	"fmt"

	"github.com/dgray001/gray_online/game"
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
	game       *game.GameBase
	players    []*RisqPlayer
	board_size uint16
	spaces     [][]*RisqSpace
}

func CreateGame(g *game.GameBase) (*GameRisq, error) {
	risq := GameRisq{
		game:       g,
		players:    []*RisqPlayer{},
		board_size: 4,
	}
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		risq.players = append(risq.players, createRisqPlayer(player))
		player_id++
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
	return &risq, nil
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
		fmt.Println("Invalid client id", action.Client_id)
		return
	}
	switch action.Kind {
	default:
		fmt.Println("Unknown game update type", action.Kind)
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
