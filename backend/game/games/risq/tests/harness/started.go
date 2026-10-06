package harness

import (
	"testing"

	"github.com/dgray001/gray_online/game"
)

func NewGame(t *testing.T, mapName string, seed int64, humans int) *Game {
	t.Helper()
	g := NewUnstartedGame(t, mapName, seed, humans)
	game.Game_StartGame(g.Risq)
	return g
}
