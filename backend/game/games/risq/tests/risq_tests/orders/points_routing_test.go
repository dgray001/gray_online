package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

func barracksAt(slot int) string {
	return fmt.Sprintf(`{"x":0,"y":1,"building":{"id":22,"player":%d}}`, slot)
}

func (w *world) zoneKey(slot, zx, zy int) int64 {
	home := w.home(slot)
	return harness.ZoneKey(home[0], home[1], zx, zy)
}

func (w *world) neighborZoneKey(slot int) int64 {
	n := w.neighbor(slot)
	return harness.ZoneKey(n[0], n[1], 0, 0)
}

func (w *world) neighborSpaceKey(slot int) int64 {
	n := w.neighbor(slot)
	return harness.SpaceKey(n[0], n[1])
}

// A space far enough from the slot's home that a villager cannot arrive within two turns, and nobody owns
func (w *world) farSpace(slot int) [2]int {
	home := w.home(slot)
	for q := -3; q <= 3; q++ {
		for r := -3; r <= 3; r++ {
			if abs(q+r) > 3 || abs(q-home[0])+abs(r-home[1])+abs(q+r-home[0]-home[1]) < 6 {
				continue
			}
			taken := false
			for other := 0; other < w.slots; other++ {
				taken = taken || homeSpaces[other] == [2]int{q, r}
			}
			if !taken {
				return [2]int{q, r}
			}
		}
	}
	w.t.Fatal("no far space")
	return [2]int{}
}

func (w *world) expectAt(u harness.Unit, slot, zx, zy int) {
	w.t.Helper()
	home := w.home(slot)
	if u.Space != (harness.Coord{X: home[0], Y: home[1]}) || u.Zone != (harness.Coord{X: zx, Y: zy}) || len(u.ActiveOrders) != 0 {
		w.t.Fatalf("slot %d unit should rest at zone (%d,%d) of its home with no orders: %+v", slot, zx, zy, u)
	}
}

type named struct {
	name  string
	steps []step
}

// The center holds one garrisoned unit and housing never finishes, so foundations stay foundations
func smallGarrison() {
	center, housing := defs.BuildingConfigs[1], defs.BuildingConfigs[2]
	center.Garrison_capacity, housing.Build_stamina = 1, 10000
	defs.BuildingConfigs[1], defs.BuildingConfigs[2] = center, housing
}

// Villagers finish production on the first tick, so a newborn and an entrant act on the same tick
func quickProduction() {
	villager := defs.UnitConfigs[1]
	villager.Production_stamina = 1
	defs.UnitConfigs[1] = villager
}

// New units follow their producer's gather point; every scenario owns a player slot and they all share five turns
func TestGatherPointRouting(t *testing.T) {
	w := newWorld(t, 10, map[int]string{1: barracksAt(1), 2: barracksAt(2)}, nil, smallGarrison)
	center := func(slot int) uint64 { return w.buildings(slot)[1] }
	barracks := func(slot int) uint64 { return w.buildings(slot)[22] }
	moves := func(slot int, point func() gin.H, orderType defs.OrderType) step {
		return step{
			pre:  func() { w.produce(slot, center(slot), 1, point()) },
			post: func() { w.expectOrder(w.fresh(slot, 1), orderType) },
		}
	}
	var wood float64
	var scenarios []named

	// routing: one slot, one point shape per turn
	scenarios = append(scenarios, named{"routing", []step{
		moves(0, func() gin.H { return gin.H{"location_kind": 1, "location_id": w.neighborSpaceKey(0)} }, defs.OrderType_UnitMoveSpace),
		moves(0, func() gin.H { return gin.H{"location_kind": 2, "location_id": w.neighborZoneKey(0)} }, defs.OrderType_UnitMoveZone),
		{
			pre: func() {
				wood = float64(w.g.Self(w.human(0)).Resources.Wood)
				w.produce(0, center(0), 1, gin.H{"location_kind": 2, "location_id": w.zoneKey(0, 1, -1), "object_type": 3})
			},
			post: func() {
				w.expectOrder(w.fresh(0, 1), defs.OrderType_UnitGather)
				if float64(w.g.Self(w.human(0)).Resources.Wood) <= wood {
					t.Fatal("gather-point unit did not gather")
				}
			},
		},
		moves(0, func() gin.H {
			return gin.H{"location_kind": 2, "location_id": w.neighborZoneKey(0), "object_type": 1, "object_id": 999999}
		}, defs.OrderType_UnitMoveZone),
		moves(0, func() gin.H {
			return gin.H{"location_kind": 2, "location_id": w.neighborZoneKey(0), "object_type": 2, "object_id": 999999}
		}, defs.OrderType_UnitMoveZone),
	}})

	arrived := func(slot int, kind uint32) func() {
		return func() {
			w.expectAt(w.fresh(slot, kind), slot, -1, 1)
			if len(w.g.Self(w.human(slot)).PlannedFoundations) != 0 {
				t.Fatal("missing foundation point planned a building")
			}
		}
	}
	scenarios = append(scenarios, named{"point shapes", []step{
		{
			pre: func() {
				w.produce(1, center(1), 1, gin.H{"location_kind": 1, "location_id": w.neighborSpaceKey(1), "object_type": 3})
			},
			post: func() { w.expectOrder(w.fresh(1, 1), defs.OrderType_UnitMoveSpace) },
		},
		{
			pre: func() {
				w.point(1, center(1), gin.H{"location_kind": 1, "location_id": w.neighborSpaceKey(1)})
				w.produce(1, center(1), 1, gin.H{"clear": true})
			},
			post: func() { w.expectAt(w.fresh(1, 1), 1, 0, 0) },
		},
		{
			pre: func() {
				w.produce(1, center(1), 1, gin.H{"location_kind": 2, "location_id": w.zoneKey(1, -1, 1), "object_type": 2, "object_id": 0})
			},
			post: arrived(1, 1),
		},
		{
			pre: func() {
				w.produce(1, barracks(1), 101, gin.H{"location_kind": 2, "location_id": w.zoneKey(1, -1, 1), "object_type": 2, "object_id": 0})
			},
			post: arrived(1, 101),
		},
	}})

	// soldiers cannot gather, so a resource point only moves them there
	var treeBefore float64
	var scout uint64
	scenarios = append(scenarios, named{"military resource point", []step{
		{
			pre: func() {
				treeBefore = w.g.State(w.human(2)).Space(w.home(2)[0], w.home(2)[1]).Resources[0].ResourcesLeft
				w.produce(2, barracks(2), 101, gin.H{"location_kind": 2, "location_id": w.zoneKey(2, 1, -1), "object_type": 3})
			},
			post: func() { scout = w.fresh(2, 101).InternalID },
		},
		{post: func() {
			w.expectAt(unitByID(w.g, w.human(2), scout), 2, 1, -1)
			if w.g.State(w.human(2)).Space(w.home(2)[0], w.home(2)[1]).Resources[0].ResourcesLeft != treeBefore {
				t.Fatal("military gather point gathered resources")
			}
		}},
	}})

	// foundation points: a planned or materialized site, reached by a site, object or zone point
	for i, c := range []struct {
		planned bool
		target  string
	}{{true, "site-point"}, {true, "object-point"}, {true, "zone-point"}, {false, "site-point"}, {false, "object-point"}, {false, "zone-point"}} {
		slot, planned, target := 3+i, c.planned, c.target
		explicit := target != "zone-point"
		site := func() int64 {
			if planned {
				far := w.farSpace(slot)
				return harness.ZoneKey(far[0], far[1], -1, 1)
			}
			return w.zoneKey(slot, -1, 1)
		}
		buildOrder := func() defs.OrderFromFrontend {
			if planned {
				far := w.farSpace(slot)
				return harness.OrderBuild(scan(w.g, w.human(slot)).villagers[:1], 2, far[0], far[1], -1, 1)
			}
			home := w.home(slot)
			return harness.OrderBuild(scan(w.g, w.human(slot)).villagers[:1], 2, home[0], home[1], -1, 1)
		}
		scenarios = append(scenarios, named{fmt.Sprintf("foundation %v/%s", planned, target), []step{
			{pre: func() { w.g.Submit(w.human(slot), buildOrder()) }},
			{pre: func() {
				orders := activeOrderIDs(w.g, w.human(slot), uint8(defs.OrderType_UnitBuild))
				w.g.Submit(w.human(slot), harness.Order(defs.OrderType_CancelOrder, nil, int64(orders[0]), false))
			}},
			{
				pre: func() {
					point := gin.H{"location_kind": 2, "location_id": site()}
					if explicit {
						point["object_type"], point["object_id"] = 2, 0
					}
					if target == "object-point" && !planned {
						point["object_id"] = buildingIDFor(w, slot, 2)
					}
					w.produce(slot, center(slot), 1, point)
				},
				post: func() {
					u := w.fresh(slot, 1)
					building := len(u.ActiveOrders) == 1 && u.ActiveOrders[0].OrderType == uint8(defs.OrderType_UnitBuild)
					if building != explicit {
						t.Fatalf("build order=%t, want %t: %+v", building, explicit, u.ActiveOrders)
					}
				},
			},
		}})
	}

	// one slot sees the point follow same-turn construction, then garrison a newborn, then run out of room
	var garrisonSite int64
	scenarios = append(scenarios, named{"garrison and construction", []step{
		{
			pre: func() {
				garrisonSite = w.zoneKey(9, -1, 1)
				w.point(9, center(9), gin.H{"location_kind": 2, "location_id": garrisonSite, "object_type": 2, "object_id": 0})
				home := w.home(9)
				w.g.Submit(w.human(9), harness.OrderBuild(scan(w.g, w.human(9)).villagers[:1], 2, home[0], home[1], -1, 1), harness.OrderProduce([]uint64{center(9)}, 1))
			},
			post: func() { w.expectOrder(w.fresh(9, 1), defs.OrderType_UnitBuild) },
		},
		{
			pre: func() {
				w.produce(9, center(9), 1, gin.H{"location_kind": 2, "location_id": w.zoneKey(9, 0, 0), "object_type": 2, "object_id": center(9)})
			},
			post: func() {
				u := w.fresh(9, 1)
				if u.GarrisonedIn == nil || *u.GarrisonedIn != center(9) {
					t.Fatalf("new unit was not garrisoned: %+v", u)
				}
				if u.CurrentStamina != u.TurnStamina+u.TurnStamina/2 || len(u.ActiveOrders) != 0 {
					t.Fatalf("direct garrison spent stamina or retained an order: %+v", u)
				}
			},
		},
		{
			pre: func() {
				w.produce(9, center(9), 1, gin.H{"location_kind": 2, "location_id": w.zoneKey(9, 0, 0), "object_type": 2, "object_id": center(9)})
			},
			post: func() {
				if u := w.fresh(9, 1); u.GarrisonedIn != nil || u.Zone != (harness.Coord{}) {
					t.Fatalf("full producer did not spawn outside: %+v", u)
				}
			},
		},
	}})

	runScenarios(w, 5, scenarios)
}

func buildingIDFor(w *world, slot int, kind uint32) uint64 {
	for _, b := range w.g.Self(w.human(slot)).Buildings {
		if b.BuildingID == kind {
			return b.InternalID
		}
	}
	w.t.Fatalf("slot %d has no building %d", slot, kind)
	return 0
}

func runScenarios(w *world, turns int, scenarios []named) {
	w.t.Helper()
	for turn := 0; turn < turns; turn++ {
		for _, s := range scenarios {
			w.scenario = fmt.Sprintf("%s turn %d", s.name, turn+1)
			if turn < len(s.steps) && s.steps[turn].pre != nil {
				s.steps[turn].pre()
			}
		}
		w.g.EndTurn()
		for _, s := range scenarios {
			w.scenario = fmt.Sprintf("%s turn %d", s.name, turn+1)
			if turn < len(s.steps) && s.steps[turn].post != nil {
				s.steps[turn].post()
			}
		}
	}
}
