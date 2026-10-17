// Package harness drives a real risq game through its exported API, the way a lobby room does.
package harness

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/util"
	"github.com/gin-gonic/gin"
)

// A started game of human players; the last submission of a turn resolves it synchronously on the caller's thread
type Game struct {
	T    *testing.T
	Risq *risq.GameRisq
	// Lite drops per-tick action histories and static config lists from Self and reuses a read until a harness call or Invalidate may have changed state
	Lite       bool
	cache      map[int]cachedSelf
	stateCache map[int]cachedState
	version    uint64
	Base       *game.GameBase
	clients    []uint64
}

var (
	testStdMu sync.Mutex
	testStd   *util.Logger
)

func SetTestLogger(t *testing.T, l *util.Logger) {
	t.Helper()
	testStdMu.Lock()
	testStd = l
	testStdMu.Unlock()
	t.Cleanup(func() {
		testStdMu.Lock()
		testStd = nil
		testStdMu.Unlock()
	})
}

func NewUnstartedGame(t *testing.T, mapName string, seed int64, humans int) *Game {
	return NewUnstartedGameWithSettings(t, mapName, seed, humans, nil)
}

func NewUnstartedGameWithSettings(t *testing.T, mapName string, seed int64, humans int, settings map[string]any) *Game {
	t.Helper()
	testStdMu.Lock()
	activeStd := testStd
	testStdMu.Unlock()
	base := game.CreateBaseGame(1, game.GameType_RISQ, map[string]any{"map": mapName, "seed": float64(seed)}, activeStd, nil)
	for key, value := range settings {
		base.GameSpecificSettings[key] = value
	}
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

func (g *Game) SetStd(std *util.Logger) {
	g.Base.Std = std
	if g.Risq != nil {
		g.Risq.SetStd(std)
	}
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
	g.Invalidate()
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
	g.Invalidate()
	g.Risq.PlayerAction(game.PlayerAction{Kind: kind, Client_id: int(g.clients[i]), Action: action})
	if failed := g.Failed(i); len(failed) > 0 {
		g.T.Fatalf("human %d's action %s was rejected: %v", i, kind, failed)
	}
	g.drain()
}

// Empties the update channels nobody reads, so the engine does not log dropped updates
func (g *Game) drain() {
	for _, client := range g.clients {
		for _, updates := range []chan *game.UpdateMessage{g.Base.Players[client].Updates, g.Base.Players[client].SettingUpdates} {
			for len(updates) > 0 {
				<-updates
			}
		}
	}
	for len(g.Base.ViewerUpdates) > 0 {
		<-g.Base.ViewerUpdates
	}
}

// Everyone who has not yet submitted submits nothing, which resolves the turn
func (g *Game) EndTurn() {
	g.T.Helper()
	g.Invalidate()
	turn := g.Risq.TurnNumber()
	for i, client := range g.clients {
		if g.Risq.TurnNumber() != turn {
			break
		}
		g.Risq.PlayerAction(game.PlayerAction{Kind: "submit-orders", Client_id: int(client), Action: gin.H{"orders": []defs.OrderFromFrontend{}}})
		for _, message := range g.Failed(i) {
			if message != "Orders already submitted" && message != "Eliminated players cannot submit orders" {
				g.T.Fatalf("human %d's empty submission was rejected: %s", i, message)
			}
		}
	}
	g.drain()
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
	if g.Lite {
		if cached, ok := g.stateCache[i]; ok && cached.turn == g.Risq.TurnNumber() && cached.version == g.version {
			return cached.state
		}
	}
	payload := g.Risq.ToFrontend(g.clients[i], false)
	if g.Lite {
		dropHeavyKeys(payload)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		g.T.Fatal(err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		g.T.Fatal(err)
	}
	if g.Lite {
		if g.stateCache == nil {
			g.stateCache = map[int]cachedState{}
		}
		g.stateCache[i] = cachedState{turn: g.Risq.TurnNumber(), version: g.version, state: state}
	}
	return state
}

type cachedState struct {
	turn    uint16
	version uint64
	state   State
}

// Per-tick histories and static config lists that Lite readers never look at
var heavyKeys = []string{"tick_actions", "builds", "move_path", "produces", "last_turn_replay", "space_links"}

func dropHeavyKeys(value any) {
	switch v := value.(type) {
	case gin.H:
		for _, key := range heavyKeys {
			delete(v, key)
		}
		for _, item := range v {
			dropHeavyKeys(item)
		}
	case []gin.H:
		for _, item := range v {
			dropHeavyKeys(item)
		}
	case [][]gin.H:
		for _, row := range v {
			dropHeavyKeys(row)
		}
	}
}

type cachedSelf struct {
	turn    uint16
	version uint64
	state   PlayerState
}

func (g *Game) Self(i int) PlayerState {
	g.T.Helper()
	if g.Lite {
		if cached, ok := g.cache[i]; ok && cached.turn == g.Risq.TurnNumber() && cached.version == g.version {
			return cached.state
		}
	}
	state := g.readSelf(i)
	if g.Lite {
		if g.cache == nil {
			g.cache = map[int]cachedSelf{}
		}
		g.cache[i] = cachedSelf{turn: g.Risq.TurnNumber(), version: g.version, state: state}
	}
	return state
}

func (g *Game) readSelf(i int) PlayerState {
	g.T.Helper()
	payload := g.Risq.PlayerToFrontend(g.clients[i], g.PlayerID(i))
	if g.Lite {
		dropHeavyKeys(payload)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		g.T.Fatal(err)
	}
	var self PlayerState
	if err := json.Unmarshal(data, &self); err != nil {
		g.T.Fatal(err)
	}
	return self
}

// Attaches the i-th human as a recipient, so the engine builds the updates they would read
func (g *Game) Connect(i int) {
	g.Base.PlayerConnected(g.clients[i])
}

// Attaches a viewer as a recipient of viewer updates
func (g *Game) AddViewer(client_id uint64) {
	g.Base.Viewers[client_id] = game.CreateViewer(client_id, fmt.Sprint("viewer", client_id))
}

func (g *Game) ConnectAll() {
	for i := range g.clients {
		g.Connect(i)
	}
}

// Marks cached reads stale; harness methods call it, and tests must after acting on the engine directly
func (g *Game) Invalidate() {
	g.version++
}

func (g *Game) playerStates(i int) []PlayerState {
	g.T.Helper()
	data, err := json.Marshal(g.Risq.PlayersToFrontend(g.clients[i]))
	if err != nil {
		g.T.Fatal(err)
	}
	var players []PlayerState
	if err := json.Unmarshal(data, &players); err != nil {
		g.T.Fatal(err)
	}
	return players
}

func (g *Game) SubmitRejected(i int, order defs.OrderFromFrontend, message string) {
	g.T.Helper()
	g.Invalidate()
	order.Player_id = g.PlayerID(i)
	before := g.Risq.StateHash()
	g.Risq.PlayerAction(game.PlayerAction{Kind: "submit-orders", Client_id: int(g.clients[i]), Action: gin.H{"orders": []defs.OrderFromFrontend{order}}})
	if failed := g.Failed(i); len(failed) != 1 || failed[0] != message {
		g.T.Fatalf("submission failures %v, want %q", failed, message)
	}
	if g.Risq.StateHash() != before {
		g.T.Fatal("rejected submission changed game state")
	}
	g.drain()
}

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
