package egyptian_rat_slap

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

/*
   ===============================
   >>>>> EGYPTIAN RAT SLAP <<<<<
   ===============================
*/

type GameEgyptianRatSlap struct {
	game    *game.GameBase
	players []*EgyptianRatSlapPlayer
}

func CreateGame(g *game.GameBase) (*GameEgyptianRatSlap, error) {
	egyptian_rat_slap := GameEgyptianRatSlap{
		game:    g,
		players: []*EgyptianRatSlapPlayer{},
	}
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		egyptian_rat_slap.players = append(egyptian_rat_slap.players, &EgyptianRatSlapPlayer{
			player: player,
		})
		player_id++
	}
	return &egyptian_rat_slap, nil
}

func (g *GameEgyptianRatSlap) GetBase() *game.GameBase {
	return g.game
}

func (g *GameEgyptianRatSlap) StartGame() {
}

func (g *GameEgyptianRatSlap) Valid() bool {
	if g.game == nil {
		return false
	}
	return true
}

func (g *GameEgyptianRatSlap) PlayerAction(action game.PlayerAction) {
	fmt.Println("player action:", action.Kind, action.Client_id, action.Action)
	switch action.Kind {
	default:
		fmt.Fprintln(os.Stderr, "Unknown game update type", action.Kind)
	}
}

func (g *GameEgyptianRatSlap) PlayerDisconnected(client_id uint64) {
}

func (g *GameEgyptianRatSlap) PlayerReconnected(client_id uint64) {
}

func (g *GameEgyptianRatSlap) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	game := gin.H{}
	if g.game != nil {
		game["game_base"] = g.game.ToFrontend(client_id, is_viewer)
	}
	players := []gin.H{}
	for _, player := range g.players {
		if player != nil {
			players = append(players, player.toFrontend(is_viewer || client_id == player.player.GetClientId()))
		}
	}
	game["players"] = players
	return game
}
