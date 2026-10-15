package aisim

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/ai"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
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

func WriteModel(t *testing.T, name, document string) {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal([]byte(document), &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := ai.ParseModel(raw, rand.New(rand.NewSource(1))).(*ai.RulesModel); !ok {
		t.Fatalf("invalid model %s", name)
	}
	if err := defs.WriteConfigFile([]byte(document), "ai", name+".json"); err != nil {
		t.Fatal(err)
	}
}

func (g *Game) Stop(t *testing.T) {
	t.Helper()
	done := make(chan struct{})
	go func() { g.Risq.StopAi(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("StopAi did not finish")
	}
}

func (g *Game) Run(t *testing.T, turns uint16) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for completed, _ := g.Risq.Standings(); completed < turns; completed, _ = g.Risq.Standings() {
		select {
		case action := <-g.Actions:
			g.Trace = append(g.Trace, action)
			g.Risq.PlayerAction(action)
		case <-g.Base.GameEndedChannel:
			t.Fatalf("game ended before turn %d: %+v", turns, g.Risq.Results())
		case <-deadline.C:
			t.Fatalf("waiting for turn %d: state=%+v trace=%+v", turns, g.State(t), g.Trace)
		}
		for len(g.Base.ViewerUpdates) > 0 {
			<-g.Base.ViewerUpdates
		}
		if finished, _ := g.Risq.Standings(); finished > completed {
			g.Check(t)
			for slot := range len(g.Base.AiPlayers) {
				player := g.Own(t, slot)
				if failures := player.Refusals(); len(failures) != 0 {
					t.Fatalf("player %d refused orders: %v; trace=%+v", player.Player.PlayerID, failures, g.Trace)
				}
			}
		}
	}
}
