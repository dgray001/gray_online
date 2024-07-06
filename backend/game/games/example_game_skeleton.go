package games

import (
	"errors"

	"github.com/dgray001/gray_online/game"
	"github.com/gin-gonic/gin"
)

/*
   ===================
   >>>>> EXAMPLE <<<<<
   ===================

   Description: a description of the game
*/

type GameExample struct {
	game *game.GameBase
}

func CreateGame(g *game.GameBase) (*GameExample, error) {
	return nil, errors.New("Example not implemented")
}

func (g *GameExample) GetBase() *game.GameBase {
	return g.game
}

func (g *GameExample) StartGame() {
}

func (g *GameExample) Valid() bool {
	return false
}

func (g *GameExample) PlayerAction(action game.PlayerAction) {
}

func (g *GameExample) PlayerDisconnected(client_id uint64) {
}

func (g *GameExample) PlayerReconnected(client_id uint64) {
}

func (g *GameExample) ToFrontend(client_id uint64, is_viewer bool) gin.H {
	return gin.H{}
}
