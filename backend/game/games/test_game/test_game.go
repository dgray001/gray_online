package test_game

import (
	"fmt"
	"os"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

/*
   =====================
   >>>>> Test Game <<<<<
   =====================

   Description: game only shown in dev, for debugging purposes
*/

type TestGame struct {
	game    *game.GameBase
	players []*TestGamePlayer
}

func CreateGame(g *game.GameBase) (*TestGame, error) {
	test_game := TestGame{
		game:    g,
		players: []*TestGamePlayer{},
	}
	var player_id = 0
	for _, player := range g.Players {
		player.Player_id = player_id
		test_game.players = append(test_game.players, &TestGamePlayer{
			player: player,
		})
		player_id++
	}
	return &test_game, nil
}

func (g *TestGame) GetBase() *game.GameBase {
	return g.game
}

func (g *TestGame) StartGame() {
}

func (g *TestGame) Valid() bool {
	return true
}

func (g *TestGame) PlayerAction(action game.PlayerAction) {
	fmt.Println("player action:", action.Kind, action.Client_id, action.Action)
	switch action.Kind {
	case "show_info":
		game.Game_BroadcastUpdate(g, &game.UpdateMessage{
			Kind:    "show_info",
			Content: g.ToFrontend(0, false),
		})
	case "end_game":
		g.game.EndGame("Game ended")
	default:
		fmt.Fprintln(os.Stderr, "Unknown game update type", action.Kind)
	}
}

func (g *TestGame) PlayerDisconnected(client_id uint64) {
}

func (g *TestGame) PlayerReconnected(client_id uint64) {
}

func (g *TestGame) ToFrontend(client_id uint64, is_viewer bool) gin.H {
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
