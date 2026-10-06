package aisim

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq"
)

type Game struct {
	Base    *game.GameBase
	Risq    *risq.GameRisq
	Actions chan game.PlayerAction
	Trace   []game.PlayerAction
}

func New(t *testing.T, mapName string, documents ...string) *Game {
	t.Helper()
	players := make([]any, len(documents))
	for slot, document := range documents {
		name := fmt.Sprintf("test-slot-%d", slot)
		WriteModel(t, name, document)
		players[slot] = map[string]any{"nickname": name, "config": name}
	}
	base := game.CreateBaseGame(1, game.GameType_RISQ, map[string]any{"map": mapName, "seed": float64(1), "ai_players": players})
	g := &Game{Base: base, Actions: make(chan game.PlayerAction, 16)}
	var err error
	if g.Risq, err = risq.CreateGame(base, g.Actions); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.Stop(t) })
	game.Game_StartGame(g.Risq)
	return g
}
