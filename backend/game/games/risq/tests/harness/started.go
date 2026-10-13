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

// Like NewGame, but every human and the given viewers are attached before the game starts, as in a real lobby
func NewConnectedGame(t *testing.T, mapName string, seed int64, humans int, viewers ...uint64) *Game {
	t.Helper()
	g := NewUnstartedGame(t, mapName, seed, humans)
	g.ConnectAll()
	for _, viewer := range viewers {
		g.AddViewer(viewer)
	}
	game.Game_StartGame(g.Risq)
	return g
}
