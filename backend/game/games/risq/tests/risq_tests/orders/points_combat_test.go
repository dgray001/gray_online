package orders

import (
	"fmt"
	"testing"

	"github.com/dgray001/gray_online/game/games/risq/internal/defs"
	"github.com/dgray001/gray_online/game/games/risq/tests/harness"
	"github.com/gin-gonic/gin"
)

const enemySlot = 6

func (w *world) health(p int, collection string, id uint64) float64 {
	return entity(w.g, p, collection, id)["combat_stats"].(gin.H)["health"].(float64)
}

// The enemy's unit or building standing in the slot's home space
func (w *world) enemyIn(slot int, buildingKind, unitKind uint32) (building, unit uint64) {
	e, home := w.human(enemySlot), w.home(slot)
	state := w.g.Self(e)
	for _, b := range state.Buildings {
		if b.BuildingID == buildingKind && b.Space == (harness.Coord{X: home[0], Y: home[1]}) {
			building = b.InternalID
		}
	}
	for _, u := range state.Units {
		if u.UnitID == unitKind && u.Space == (harness.Coord{X: home[0], Y: home[1]}) {
			unit = u.InternalID
		}
	}
	return building, unit
}

func (w *world) enemyUnitHealth(id uint64) float64 {
	if !hasUnit(w.g, w.human(enemySlot), id) {
		w.t.Fatalf("%s: enemy unit %d is gone (killed by the new soldier?)", w.scenario, id)
	}
	return unitByID(w.g, w.human(enemySlot), id).CombatStats.Health
}

// Gather points that attack, repair or refuse to attack friends, and two units racing for one garrison slot
func TestGatherPointCombat(t *testing.T) {
	enemyAt := func(slot int, content string) string {
		return fmt.Sprintf(`{"x":1,"y":0,%s}`, content)
	}
	housing := func(owner int) string { return fmt.Sprintf(`"building":{"id":2,"player":%d}`, owner) }
	heavy := fmt.Sprintf(`{"id":103,"player":%d,"count":1}`, enemySlot)
	zones := map[int]string{
		0: barracksAt(0) + "," + enemyAt(0, `"units":[`+heavy+`]`),
		1: barracksAt(1) + "," + enemyAt(1, housing(enemySlot)+`,"units":[`+heavy+`]`),
		2: enemyAt(2, housing(2)+`,"units":[`+heavy+`]`),
		3: enemyAt(3, housing(3)+fmt.Sprintf(`,"units":[{"id":1,"player":%d,"count":1}]`, enemySlot)),
	}
	w := newWorld(t, 7, zones, nil, func() { smallGarrison(); quickProduction() })
	e := w.human(enemySlot)
	var passive []uint64
	for _, u := range w.g.Self(e).Units {
		if u.UnitID == 103 {
			passive = append(passive, u.InternalID)
		}
	}
	w.g.Action(e, "set-unit-behavior", gin.H{"internal_ids": passive, "stance": 1, "attack_back": false})
	center := func(slot int) uint64 { return w.buildings(slot)[1] }
	var scenarios []named

	// a soldier produced at an attack point starts hitting a visible enemy unit or building at once
	for slot, kind := range []string{"unit", "building"} {
		slot, kind := slot, kind
		var target uint64
		var before float64
		scenarios = append(scenarios, named{"attack " + kind, []step{{
			pre: func() {
				building, unit := w.enemyIn(slot, 2, 103)
				object := uint8(1)
				target, before = unit, w.enemyUnitHealth(unit)
				if kind == "building" {
					object, target = 2, building
					before = w.health(e, "buildings", building)
				}
				w.produce(slot, w.buildings(slot)[22], 101, gin.H{"location_kind": 2, "location_id": w.zoneKey(slot, 1, 0), "object_type": object, "object_id": target})
			},
			post: func() {
				orderType := defs.OrderType_UnitAttackUnit
				var after float64
				if kind == "building" {
					orderType, after = defs.OrderType_UnitAttackBuilding, w.health(e, "buildings", target)
				} else {
					after = w.enemyUnitHealth(target)
				}
				w.expectOrder(w.fresh(slot, 101), orderType)
				if after >= before {
					t.Fatalf("gather-point attack caused no damage: before %v, after %v", before, after)
				}
			},
		}}})
	}

	// a healthy friendly building cannot be garrisoned or attacked, so the unit just walks to it
	scenarios = append(scenarios, named{"friendly building", []step{{
		pre: func() {
			own := buildingIDFor(w, 2, 2)
			w.produce(2, center(2), 1, gin.H{"location_kind": 2, "location_id": w.zoneKey(2, 1, 0), "object_type": 2, "object_id": own})
		},
		post: func() {
			u := w.fresh(2, 1)
			w.expectAt(u, 2, 1, 0)
			if u.GarrisonedIn != nil || w.health(w.human(2), "buildings", buildingIDFor(w, 2, 2)) != 120 {
				t.Fatalf("friendly building fallback failed: %+v", u)
			}
		},
	}}})

	// a damaged friendly building that cannot be garrisoned gets repaired
	var housingHealth float64
	scenarios = append(scenarios, named{"repair", []step{
		{pre: func() {
			var attacker uint64
			for _, u := range w.g.Self(e).Units {
				if u.UnitID == 1 && u.Space == (harness.Coord{X: w.home(3)[0], Y: w.home(3)[1]}) {
					attacker = u.InternalID
				}
			}
			w.g.Submit(e, harness.Order(defs.OrderType_UnitAttackBuilding, []uint64{attacker}, int64(buildingIDFor(w, 3, 2)), false))
		}},
		{pre: func() {
			order := w.g.Self(e).ActiveOrders[0].InternalID
			w.g.Submit(e, harness.Order(defs.OrderType_CancelOrder, nil, int64(order), false))
		}},
		{
			pre: func() {
				id := buildingIDFor(w, 3, 2)
				housingHealth = w.health(w.human(3), "buildings", id)
				w.produce(3, center(3), 1, gin.H{"location_kind": 2, "location_id": w.zoneKey(3, 1, 0), "object_type": 2, "object_id": id})
			},
			post: func() {
				after := w.health(w.human(3), "buildings", buildingIDFor(w, 3, 2))
				if housingHealth >= 120 || after <= housingHealth || w.fresh(3, 1).GarrisonedIn != nil || w.g.Self(w.human(3)).Resources.Wood >= 1000 {
					t.Fatalf("repair fallback failed: before %v, after %v", housingHealth, after)
				}
			},
		},
	}})

	// an existing villager and a newborn race for the producer's one garrison slot, in either batch order
	for i, reverse := range []bool{false, true} {
		slot := 4 + i
		scenarios = append(scenarios, named{fmt.Sprintf("reservation reverse=%v", reverse), []step{{
			pre: func() {
				building, units := center(slot), scan(w.g, w.human(slot)).villagers
				w.point(slot, building, gin.H{"location_kind": 2, "location_id": w.zoneKey(slot, 0, 0), "object_type": 2, "object_id": building})
				orders := []defs.OrderFromFrontend{harness.OrderGarrison(units[:1], building), harness.OrderProduce([]uint64{building}, 1)}
				if reverse {
					orders[0], orders[1] = orders[1], orders[0]
				}
				w.g.Submit(w.human(slot), orders...)
			},
			post: func() {
				building, entrant := center(slot), scan(w.g, w.human(slot)).villagers[0]
				newborn := w.fresh(slot, 1)
				if e := unitByID(w.g, w.human(slot), entrant); e.GarrisonedIn == nil || *e.GarrisonedIn != building || newborn.GarrisonedIn != nil {
					t.Fatalf("newborn competed with existing entrant: entrant %+v, newborn %+v", e, newborn)
				}
				if len(entity(w.g, w.human(slot), "buildings", building)["garrisoned_units"].([]uint64)) != 1 {
					t.Fatal("producer exceeded its garrison capacity")
				}
			},
		}}})
	}

	runScenarios(w, 3, scenarios)
}
