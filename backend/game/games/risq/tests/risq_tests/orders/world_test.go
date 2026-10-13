package orders

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness/fakeboard"
	"github.com/gin-gonic/gin"
)

// Where each player slot's home space sits on the radius-three board; far enough apart that scenarios never meet
var homeSpaces = [][2]int{{0, 0}, {-2, 0}, {2, 0}, {0, 2}, {0, -2}, {2, -2}, {-2, 2}, {-2, -1}, {2, -3}, {-2, 1}, {2, -1}, {0, -1}}

// Independent scenarios share one game by each owning a player slot with its own home space
type world struct {
	t *testing.T
	g *harness.Game
	// extra zone json per slot, beyond the center village and the cedar tree
	zones    map[int]string
	slots    int
	known    map[int]map[uint64]bool
	starting map[int]map[uint32]uint64
	// the scenario whose step is running, for failure messages
	scenario string
}

type step struct {
	pre  func()
	post func()
}

func (w *world) doc(villagers map[int]int) string {
	home := map[[2]int]int{}
	for slot := 0; slot < w.slots; slot++ {
		home[homeSpaces[slot]] = slot
	}
	var spaces []string
	for q := -3; q <= 3; q++ {
		for r := -3; r <= 3; r++ {
			if abs(q+r) > 3 {
				continue
			}
			slot, isHome := home[[2]int{q, r}]
			if !isHome {
				spaces = append(spaces, fmt.Sprintf(`{"x":%d,"y":%d,"terrain":1}`, q, r))
				continue
			}
			count := villagers[slot]
			if count == 0 {
				count = 1
			}
			zones := []string{
				fmt.Sprintf(`{"x":0,"y":0,"building":{"id":1,"player":%d},"units":[{"id":1,"player":%d,"count":%d}]}`, slot, slot, count),
				`{"x":1,"y":-1,"resource":11}`,
			}
			if extra := w.zones[slot]; extra != "" {
				zones = append(zones, extra)
			}
			spaces = append(spaces, fmt.Sprintf(`{"x":%d,"y":%d,"terrain":1,"zones":[%s]}`, q, r, strings.Join(zones, ",")))
		}
	}
	return fmt.Sprintf(`{"board_size":3,"players":%d,"space_gold_income":0,"unlimited_population":true,"starting_bank":%s,"spaces":[%s]}`, w.slots, richBank, strings.Join(spaces, ","))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func newWorld(t *testing.T, slots int, zones map[int]string, villagers map[int]int, tweak func()) *world {
	t.Helper()
	// one extra player never submits during a turn, so EndTurn is what resolves it
	w := &world{t: t, slots: slots + 1, zones: zones, known: map[int]map[uint64]bool{}, starting: map[int]map[uint32]uint64{}}
	fakeboard.UseConfig(t, testConfig, nil, map[string]string{"world": w.doc(villagers)})
	if tweak != nil {
		tweak()
	}
	w.g = harness.NewGame(t, "custom:world", 1, w.slots)
	w.g.Lite = true
	for slot := 0; slot < w.slots; slot++ {
		w.known[slot] = map[uint64]bool{}
		for _, u := range w.g.Self(w.g.Human(slot)).Units {
			w.known[slot][u.InternalID] = true
		}
	}
	return w
}

func (w *world) human(slot int) int { return w.g.Human(slot) }

func (w *world) home(slot int) [2]int { return homeSpaces[slot] }

// The space one step east of the slot's home
func (w *world) neighbor(slot int) [2]int {
	home := w.home(slot)
	return [2]int{home[0] + 1, home[1]}
}

// Starting buildings never change in these scenarios, so they are looked up once
func (w *world) buildings(slot int) map[uint32]uint64 {
	if _, cached := w.starting[slot]; !cached {
		w.starting[slot] = scan(w.g, w.human(slot)).buildings
	}
	return w.starting[slot]
}

func (w *world) point(slot int, building uint64, point gin.H) {
	w.t.Helper()
	point["building_id"] = building
	w.g.Action(w.human(slot), "set-gather-point", point)
}

// Sets the building's gather point and queues one unit of the given kind from it
func (w *world) produce(slot int, producer uint64, kind uint32, point gin.H) {
	w.t.Helper()
	w.point(slot, producer, point)
	w.g.Submit(w.human(slot), harness.OrderProduce([]uint64{producer}, kind))
}

// The one unit of this kind the slot gained since it last looked
func (w *world) fresh(slot int, kind uint32) harness.Unit {
	w.t.Helper()
	var found []harness.Unit
	for _, u := range w.g.Self(w.human(slot)).Units {
		if u.UnitID == kind && !w.known[slot][u.InternalID] {
			found = append(found, u)
			w.known[slot][u.InternalID] = true
		}
	}
	if len(found) != 1 {
		w.t.Fatalf("slot %d gained %d units of kind %d, want one", slot, len(found), kind)
	}
	return found[0]
}

func (w *world) expectOrder(u harness.Unit, orderType defs.OrderType) {
	w.t.Helper()
	if len(u.ActiveOrders) != 1 || u.ActiveOrders[0].OrderType != uint8(orderType) || u.ActiveOrders[0].InternalID == 0 {
		w.t.Fatalf("%s: new unit has wrong synthetic order, want type %d: %+v", w.scenario, orderType, u)
	}
}
