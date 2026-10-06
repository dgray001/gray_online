// Package harness drives a real risq game through its exported API, the way a lobby room does.
package harness

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/gin-gonic/gin"
)

// A started game of human players; the last submission of a turn resolves it synchronously on the caller's thread
type Game struct {
	T       *testing.T
	Risq    *risq.GameRisq
	Base    *game.GameBase
	clients []uint64
}

func NewUnstartedGame(t *testing.T, mapName string, seed int64, humans int) *Game {
	t.Helper()
	base := game.CreateBaseGame(1, game.GameType_RISQ, map[string]any{"map": mapName, "seed": float64(seed)})
	g := &Game{T: t, Base: base}
	for i := 1; i <= humans; i++ {
		game.CreatePlayer(uint64(i), fmt.Sprint("human", i), base)
		g.clients = append(g.clients, uint64(i))
	}
	var err error
	if g.Risq, err = risq.CreateGame(base, make(chan game.PlayerAction, 16)); err != nil {
		t.Fatal(err)
	}
	return g
}

// Engine player id of the i-th human; the engine assigns ids in map order, so they are read back, never assumed
func (g *Game) PlayerID(i int) int {
	return g.Base.Players[g.clients[i]].Player_id
}

// The human index playing engine player slot: a map's "player": N refers to a slot, and which human holds it varies
func (g *Game) Human(slot int) int {
	for i := range g.clients {
		if g.PlayerID(i) == slot {
			return i
		}
	}
	g.T.Fatalf("no human holds slot %d", slot)
	return -1
}

// Fails the test if the engine rejects the submission itself; orders it later refuses show up in Refusals instead
func (g *Game) Submit(i int, orders ...defs.OrderFromFrontend) {
	g.T.Helper()
	for k := range orders {
		orders[k].Player_id = g.PlayerID(i)
	}
	g.Risq.PlayerAction(game.PlayerAction{Kind: "submit-orders", Client_id: int(g.clients[i]), Action: gin.H{"orders": orders}})
	if failed := g.Failed(i); len(failed) > 0 {
		g.T.Fatalf("human %d's submission was rejected: %v", i, failed)
	}
	g.drain()
}

func (g *Game) Action(i int, kind string, action gin.H) {
	g.T.Helper()
	g.Risq.PlayerAction(game.PlayerAction{Kind: kind, Client_id: int(g.clients[i]), Action: action})
	if failed := g.Failed(i); len(failed) > 0 {
		g.T.Fatalf("human %d's action %s was rejected: %v", i, kind, failed)
	}
	g.drain()
}

// Empties the update channels nobody reads, so the engine does not log dropped updates
func (g *Game) drain() {
	for _, client := range g.clients {
		for updates := g.Base.Players[client].Updates; len(updates) > 0; {
			<-updates
		}
	}
	for len(g.Base.ViewerUpdates) > 0 {
		<-g.Base.ViewerUpdates
	}
}

// Everyone who has not yet submitted submits nothing, which resolves the turn; stops as soon as it has
func (g *Game) EndTurn() {
	g.T.Helper()
	state := State{Players: g.playerStates(0)}
	for i := range g.clients {
		if !state.Player(g.PlayerID(i)).OrdersSubmitted {
			g.Submit(i)
		}
	}
}

// Messages of the failed updates queued for the i-th human since the last call
func (g *Game) Failed(i int) []string {
	var messages []string
	for {
		select {
		case update := <-g.Base.Players[g.clients[i]].FailedUpdates:
			messages = append(messages, fmt.Sprint(update.Content["message"]))
		default:
			return messages
		}
	}
}

// What the i-th human's client would see right now
func (g *Game) State(i int) State {
	g.T.Helper()
	data, err := json.Marshal(g.Risq.ToFrontend(g.clients[i], false))
	if err != nil {
		g.T.Fatal(err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		g.T.Fatal(err)
	}
	return state
}

func (g *Game) Self(i int) PlayerState {
	g.T.Helper()
	state := State{Players: g.playerStates(i)}
	self := state.Player(g.PlayerID(i))
	if self == nil {
		g.T.Fatalf("player %d missing from its own state", i)
	}
	return *self
}
