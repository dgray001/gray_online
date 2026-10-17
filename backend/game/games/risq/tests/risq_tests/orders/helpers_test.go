package orders

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"slices"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game"
	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/config"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

var testConfig = testconfig.Dir()

func TestMain(m *testing.M) {
	debug.SetGCPercent(800)
	if err := defs.LoadConfig(testConfig); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

const richBank = `{"food":1000,"wood":1000,"stone":1000,"gold":1000}`

// Player 0 owns a village center, blacksmith, farm, barracks and redoubt around a cedar tree; player 1 sits three spaces west
type layout struct {
	bank      string
	extra     string
	spaces    string
	villagers int
	infantry  int
	humans    int
	connected bool
	unstarted bool
	viewers   []uint64
	tweak     func()
}

func (l layout) doc() string {
	home := fmt.Sprintf(`{"x":0,"y":0,"terrain":1,"zones":[`+
		`{"x":0,"y":0,"building":{"id":1,"player":0},"units":[{"id":1,"player":0,"count":%d},{"id":101,"player":0,"count":%d}]},`+
		`{"x":1,"y":0,"building":{"id":11,"player":0}},{"x":-1,"y":0,"building":{"id":3,"player":0}},`+
		`{"x":0,"y":1,"building":{"id":22,"player":0}},{"x":0,"y":-1,"building":{"id":23,"player":0}},{"x":1,"y":-1,"resource":11}]}`, l.villagers, l.infantry)
	route := `,{"x":1,"y":0,"terrain":1},{"x":2,"y":0,"terrain":1},{"x":3,"y":0,"terrain":1}`
	enemy := `,{"x":-3,"y":0,"terrain":1,"zones":[{"x":0,"y":0,"building":{"id":1,"player":1},"units":[{"id":1,"player":1,"count":1}]}]}`
	return fmt.Sprintf(`{"board_size":3,"players":%d,"space_gold_income":0,"unlimited_population":true,"starting_bank":%s,"spaces":[%s%s%s]%s}`,
		l.humans, l.bank, home, route, enemy+l.spaces, l.extra)
}

func newGame(t *testing.T, l layout) *harness.Game {
	t.Helper()
	if l.bank == "" {
		l.bank = richBank
	}
	if l.villagers == 0 {
		l.villagers = 2
	}
	if l.infantry == 0 {
		l.infantry = 1
	}
	if l.humans == 0 {
		l.humans = 2
	}
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"orders": l.doc()})
	if l.tweak != nil {
		l.tweak()
	}
	var g *harness.Game
	switch {
	case l.unstarted:
		g = harness.NewUnstartedGame(t, "custom:orders", 1, l.humans)
	case !l.connected:
		g = harness.NewGame(t, "custom:orders", 1, l.humans)
	default:
		g = harness.NewConnectedGame(t, "custom:orders", 1, l.humans, l.viewers...)
	}
	g.Lite = true
	return g
}

// The ids of what each human starts with, so tests do not re-serialize state to find them
type roster struct {
	villagers []uint64
	infantry  []uint64
	buildings map[uint32]uint64
}

func scan(g *harness.Game, p int) roster {
	g.T.Helper()
	r := roster{buildings: map[uint32]uint64{}}
	state := g.Self(p)
	for _, u := range state.Units {
		if u.UnitID == 1 {
			r.villagers = append(r.villagers, u.InternalID)
		} else {
			r.infantry = append(r.infantry, u.InternalID)
		}
	}
	for _, b := range state.Buildings {
		if _, seen := r.buildings[b.BuildingID]; !seen || b.InternalID < r.buildings[b.BuildingID] {
			r.buildings[b.BuildingID] = b.InternalID
		}
	}
	slices.Sort(r.villagers)
	slices.Sort(r.infantry)
	return r
}

func unitIDs(g *harness.Game, p int, kind uint32) []uint64 {
	var ids []uint64
	for _, u := range g.Self(p).Units {
		if u.UnitID == kind {
			ids = append(ids, u.InternalID)
		}
	}
	slices.SortFunc(ids, cmp.Compare[uint64])
	return ids
}

func buildingID(g *harness.Game, p int, kind uint32) uint64 {
	g.T.Helper()
	for _, b := range g.Self(p).Buildings {
		if b.BuildingID == kind {
			return b.InternalID
		}
	}
	g.T.Fatalf("building %d missing", kind)
	return 0
}

func unitByID(g *harness.Game, p int, id uint64) harness.Unit {
	g.T.Helper()
	for _, u := range g.Self(p).Units {
		if u.InternalID == id {
			return u
		}
	}
	g.T.Fatalf("unit %d missing", id)
	return harness.Unit{}
}

func hasUnit(g *harness.Game, p int, id uint64) bool {
	for _, u := range g.Self(p).Units {
		if u.InternalID == id {
			return true
		}
	}
	return false
}

// One of the player's own units or buildings as the engine serializes it
func entity(g *harness.Game, p int, collection string, id uint64) gin.H {
	g.T.Helper()
	for _, item := range g.Risq.PlayerToFrontend(uint64(p+1), g.PlayerID(p))[collection].([]gin.H) {
		if item["internal_id"] == id {
			return item
		}
	}
	g.T.Fatalf("%s %d missing", collection, id)
	return nil
}

func activeOrderIDs(g *harness.Game, p int, orderType uint8) []uint64 {
	var ids []uint64
	for _, o := range g.Self(p).ActiveOrders {
		if o.OrderType == orderType {
			ids = append(ids, o.InternalID)
		}
	}
	return ids
}

func rawAction(g *harness.Game, p int, kind string, action gin.H) {
	g.Invalidate()
	g.Risq.PlayerAction(game.PlayerAction{Client_id: p + 1, Kind: kind, Action: action})
}

// Fails unless the engine rejects the action with exactly one failure containing message
func rejectAction(g *harness.Game, p int, kind string, action gin.H, message string) string {
	g.T.Helper()
	queued, before := queuedUpdates(g), g.Risq.StateHash()
	rawAction(g, p, kind, action)
	failures := g.Failed(p)
	if len(failures) != 1 || !strings.Contains(failures[0], message) {
		g.T.Fatalf("%s failures %v, want %q", kind, failures, message)
	}
	if queuedUpdates(g) != queued {
		g.T.Fatalf("rejected %s emitted a success notification", kind)
	}
	if g.Risq.StateHash() != before {
		g.T.Fatalf("rejected %s changed game state", kind)
	}
	return failures[0]
}

// Fails if running fn changes any engine state; for batches of rejections and no-ops
func unchanged(g *harness.Game, what string, fn func()) {
	g.T.Helper()
	before := g.Risq.StateHash()
	fn()
	if g.Risq.StateHash() != before {
		g.T.Fatalf("%s changed game state", what)
	}
}

func queuedUpdates(g *harness.Game) string {
	counts := []int{len(g.Base.ViewerUpdates)}
	var clients []uint64
	for id := range g.Base.Players {
		clients = append(clients, id)
	}
	slices.Sort(clients)
	for _, id := range clients {
		counts = append(counts, len(g.Base.Players[id].Updates), len(g.Base.Players[id].SettingUpdates))
	}
	return fmt.Sprint(counts)
}

func nextUpdate(t *testing.T, updates <-chan *game.UpdateMessage) *game.UpdateMessage {
	t.Helper()
	select {
	case u := <-updates:
		return u
	default:
		t.Fatal("missing update")
		return nil
	}
}

func latestUpdate(t *testing.T, updates <-chan *game.UpdateMessage) *game.UpdateMessage {
	t.Helper()
	u := nextUpdate(t, updates)
	for len(updates) > 0 {
		u = nextUpdate(t, updates)
	}
	return u
}

// Serializes a payload with every unordered collection sorted, so equal states compare equal
func snapshotJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(normalizeSnapshot("", decoded))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func normalizeSnapshot(key string, value any) any {
	switch data := value.(type) {
	case map[string]any:
		for field, item := range data {
			data[field] = normalizeSnapshot(field, item)
		}
	case []any:
		for i, item := range data {
			data[i] = normalizeSnapshot("", item)
		}
		if slices.Contains([]string{"players", "units", "buildings", "resources", "subjects", "garrisoned_units", "planned_foundations", "available_mercenaries"}, key) {
			slices.SortFunc(data, func(a, b any) int { return strings.Compare(fmt.Sprint(a), fmt.Sprint(b)) })
		}
	}
	return value
}
