package risq

import (
	"fmt"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

/*
   ================
	 >>>>> RISQ <<<<<
	 ================

	 Objective: ???
	 Description: ???
*/

type GameRisq struct {
	game *game.GameBase
}

func CreateGame(g *game.GameBase) (*GameRisq, error) {
	risq := GameRisq{
		game: g,
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
	game := gin.H{}
	if r.game != nil {
		game["game_base"] = r.game.ToFrontend(client_id, is_viewer)
	}
	return game
}
